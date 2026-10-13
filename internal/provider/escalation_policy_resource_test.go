package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func fakeEscalationPolicyServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	policy := `{
		"publicId":  "ep_01HWTEST0000000000000001",
		"name":      "P1 On-Call",
		"createdAt": "2026-01-01T00:00:00.000Z",
		"updatedAt": "2026-01-01T00:00:00.000Z",
		"steps": [
			{"stepOrder":0,"targetType":"user","targetId":1,"waitMinutes":5,"channels":["email","push"]}
		]
	}`

	mux.HandleFunc("POST /api/on-call/policies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, policy)
	})

	// List endpoint — used by GetEscalationPolicy
	mux.HandleFunc("GET /api/on-call/policies", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[%s]", policy)
	})

	mux.HandleFunc("PUT /api/on-call/policies/ep_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("DELETE /api/on-call/policies/ep_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestEscalationPolicyResource_CRUD(t *testing.T) {
	srv := fakeEscalationPolicyServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_escalation_policy" "test" {
  name = "P1 On-Call"
  steps = [
    {
      step_order  = 0
      target_type = "user"
      target_id   = 1
      wait_minutes = 5
      channels    = ["email", "push"]
    }
  ]
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_escalation_policy.test", "id", "ep_01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_escalation_policy.test", "name", "P1 On-Call"),
					resource.TestCheckResourceAttr("uptimewarden_escalation_policy.test", "steps.#", "1"),
					resource.TestCheckResourceAttr("uptimewarden_escalation_policy.test", "steps.0.target_type", "user"),
					resource.TestCheckResourceAttr("uptimewarden_escalation_policy.test", "steps.0.wait_minutes", "5"),
				),
			},
		},
	})
}

func TestEscalationPolicyResource_InvalidTargetType(t *testing.T) {
	srv := fakeEscalationPolicyServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_escalation_policy" "test" {
  name = "Bad"
  steps = [
    { step_order = 0, target_type = "team", target_id = 1 }
  ]
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`value must be one of:`)},
		},
	})
}

// Ensure json import is used
var _ = json.Marshal
