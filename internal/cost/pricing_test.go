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

func TestComputeMonthlyCostLevers(t *testing.T) {
	base := ComputeMonthlyCost(ModelInputs{DAU: 100000, Tiering: false, CompressionRatio: 1, Spot: false})
	levered := ComputeMonthlyCost(ModelInputs{DAU: 100000, Tiering: true, CompressionRatio: 5, Spot: true})
	if levered.PerUser >= base.PerUser {
		t.Fatalf("expected levers to lower per-user cost: base=%+v levered=%+v", base, levered)
	}
}
