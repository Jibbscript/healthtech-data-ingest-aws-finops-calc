package cost

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jibbscript/throne-backend-poc/internal/domain"
)

const SourceSnapshot = "snapshot-2026-05"

type Service struct {
	DataDir  string
	Snapshot map[string]domain.Pricing
}

func New(dataDir string) *Service { return &Service{DataDir: dataDir, Snapshot: DefaultSnapshot()} }

func DefaultSnapshot() map[string]domain.Pricing {
	return map[string]domain.Pricing{
		"fargate:us-east-1":                    {Region: "us-east-1", Source: SourceSnapshot, FargateVCPUPerHour: 0.04048, FargateGBPerHour: 0.004445, Currency: "USD"},
		"s3:us-east-1:standard":                {Region: "us-east-1", Source: SourceSnapshot, S3Class: "standard", S3GBMonth: 0.023, Currency: "USD"},
		"s3:us-east-1:ia":                      {Region: "us-east-1", Source: SourceSnapshot, S3Class: "ia", S3GBMonth: 0.0125, Currency: "USD"},
		"s3:us-east-1:glacier":                 {Region: "us-east-1", Source: SourceSnapshot, S3Class: "glacier", S3GBMonth: 0.004, Currency: "USD"},
		"sqs:us-east-1":                        {Region: "us-east-1", Source: SourceSnapshot, SQSRequestPerMillion: 0.40, Currency: "USD"},
		"rds:us-east-1:db.m6g.large":           {Region: "us-east-1", Source: SourceSnapshot, RDSInstancePerHour: 0.152, Currency: "USD"},
		"data-transfer:us-east-1:out-internet": {Region: "us-east-1", Source: SourceSnapshot, DataTransferGB: 0.09, Currency: "USD"},
	}
}

func (s *Service) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/v1/pricing/fargate", func(w http.ResponseWriter, r *http.Request) { s.write(w, s.Snapshot["fargate:"+region(r)]) })
	mux.HandleFunc("/v1/pricing/s3", func(w http.ResponseWriter, r *http.Request) {
		class := r.URL.Query().Get("class")
		if class == "" {
			class = "standard"
		}
		s.write(w, s.Snapshot["s3:"+region(r)+":"+class])
	})
	mux.HandleFunc("/v1/pricing/sqs", func(w http.ResponseWriter, r *http.Request) { s.write(w, s.Snapshot["sqs:"+region(r)]) })
	mux.HandleFunc("/v1/pricing/rds", func(w http.ResponseWriter, r *http.Request) {
		inst := r.URL.Query().Get("instance")
		if inst == "" {
			inst = "db.m6g.large"
		}
		s.write(w, s.Snapshot["rds:"+region(r)+":"+inst])
	})
	mux.HandleFunc("/v1/pricing/data-transfer", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("direction")
		if dir == "" {
			dir = "out-internet"
		}
		s.write(w, s.Snapshot["data-transfer:"+region(r)+":"+dir])
	})
	return cors(mux)
}

func region(r *http.Request) string {
	v := r.URL.Query().Get("region")
	if v == "" {
		return "us-east-1"
	}
	return v
}
func (s *Service) write(w http.ResponseWriter, p domain.Pricing) {
	w.Header().Set("content-type", "application/json")
	if p.Currency == "" {
		http.Error(w, "pricing not found", 404)
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "content-type,authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) RefreshSnapshot() (string, error) {
	if s.DataDir == "" {
		s.DataDir = ".cache"
	}
	path := filepath.Join(s.DataDir, "pricing.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	payload := map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339), "source": SourceSnapshot, "prices": s.Snapshot}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, b, 0o600)
}

type ModelInputs struct {
	DAU                int
	CapturesPerUserDay float64
	PayloadMB          float64
	CompressionRatio   float64
	Tiering            bool
	Spot               bool
}
type ModelBreakdown struct {
	Compute       float64 `json:"compute"`
	Storage       float64 `json:"storage"`
	Requests      float64 `json:"requests"`
	Observability float64 `json:"observability"`
	Total         float64 `json:"total"`
	PerUser       float64 `json:"per_user"`
}

func ComputeMonthlyCost(in ModelInputs) ModelBreakdown {
	if in.DAU <= 0 {
		in.DAU = 300
	}
	if in.CapturesPerUserDay <= 0 {
		in.CapturesPerUserDay = 3
	}
	if in.PayloadMB <= 0 {
		in.PayloadMB = 25
	}
	if in.CompressionRatio <= 0 {
		in.CompressionRatio = 5
	}
	captures := float64(in.DAU) * in.CapturesPerUserDay * 30
	gbMonth := captures * (in.PayloadMB / in.CompressionRatio) / 1024
	storageRate := 0.023
	if in.Tiering {
		storageRate = 0.0074
	}
	computePerCapture := 0.00046
	if in.Spot {
		computePerCapture -= 0.00002
	}
	b := ModelBreakdown{Compute: captures * computePerCapture, Storage: gbMonth * storageRate, Requests: captures * 0.000011, Observability: float64(in.DAU) * 0.02}
	b.Total = b.Compute + b.Storage + b.Requests + b.Observability
	b.PerUser = b.Total / float64(in.DAU)
	return b
}
