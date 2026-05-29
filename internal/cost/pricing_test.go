package cost

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFargateEndpointUsesSnapshot(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/pricing/fargate?region=us-east-1", nil)
	New(t.TempDir()).Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["fargate_vcpu_per_hour"] != 0.04048 {
		t.Fatalf("unexpected price: %#v", got)
	}
}

func TestPricingEndpointsApplyQueryDefaults(t *testing.T) {
	svc := New(t.TempDir())
	cases := []struct {
		name  string
		path  string
		field string
		want  any
	}{
		{"s3 defaults to standard class", "/v1/pricing/s3?region=us-east-1", "s3_class", "standard"},
		{"s3 default standard rate", "/v1/pricing/s3?region=us-east-1", "s3_gb_month", 0.023},
		{"rds defaults to db.m6g.large", "/v1/pricing/rds?region=us-east-1", "rds_instance_per_hour", 0.152},
		{"data-transfer defaults to out-internet", "/v1/pricing/data-transfer?region=us-east-1", "data_transfer_gb", 0.09},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			svc.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
			}
			var got map[string]any
			if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if got[tc.field] != tc.want {
				t.Fatalf("%s = %#v, want %#v (body=%s)", tc.field, got[tc.field], tc.want, rr.Body.String())
			}
		})
	}
}

func TestPricingEndpointReturns404ForUnknownKey(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/pricing/fargate?region=eu-west-1", nil)
	New(t.TempDir()).Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("want 404 for unknown region, got %d body=%s", rr.Code, rr.Body.String())
	}
}
