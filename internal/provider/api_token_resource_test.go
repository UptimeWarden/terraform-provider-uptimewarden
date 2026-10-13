package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func fakeAPITokenServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/user/api-tokens/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"token": {
				"publicId":    "tk_01HWTEST0000000000000001",
				"name":        "CI Token",
				"prefix":      "uptw_CITEST",
				"permissions": ["monitors:read","incidents:read"],
				"isActive":    true,
				"createdAt":   "2026-01-01T00:00:00.000Z",
				"updatedAt":   "2026-01-01T00:00:00.000Z"
			},
			"plainToken": "uptw_CITEST_SECRETTOKEN",
			"message":    "Token created"
		}`)
	})

	mux.HandleFunc("GET /api/user/api-tokens/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"tokens": [
				{
					"publicId":    "tk_01HWTEST0000000000000001",
					"name":        "CI Token",
					"prefix":      "uptw_CITEST",
					"permissions": ["monitors:read","incidents:read"],
					"isActive":    true,
					"createdAt":   "2026-01-01T00:00:00.000Z",
					"updatedAt":   "2026-01-01T00:00:00.000Z"
				}
			]
		}`)
	})

	mux.HandleFunc("PATCH /api/user/api-tokens/tk_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":    "tk_01HWTEST0000000000000001",
			"name":        "CI Token",
			"prefix":      "uptw_CITEST",
			"permissions": ["monitors:read","incidents:read"],
			"isActive":    true,
			"createdAt":   "2026-01-01T00:00:00.000Z",
			"updatedAt":   "2026-01-01T00:00:00.000Z"
		}`)
	})

	mux.HandleFunc("PATCH /api/user/api-tokens/tk_01HWTEST0000000000000001/permissions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":    "tk_01HWTEST0000000000000001",
			"name":        "CI Token",
			"prefix":      "uptw_CITEST",
			"permissions": ["monitors:read","incidents:read"],
			"isActive":    true,
			"createdAt":   "2026-01-01T00:00:00.000Z",
			"updatedAt":   "2026-01-01T00:00:00.000Z"
		}`)
	})

	mux.HandleFunc("DELETE /api/user/api-tokens/tk_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestAPITokenResource_Create(t *testing.T) {
	srv := fakeAPITokenServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_api_token" "test" {
  name        = "CI Token"
  permissions = ["monitors:read", "incidents:read"]
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_api_token.test", "id", "tk_01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_api_token.test", "name", "CI Token"),
					resource.TestCheckResourceAttr("uptimewarden_api_token.test", "prefix", "uptw_CITEST"),
					resource.TestCheckResourceAttr("uptimewarden_api_token.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("uptimewarden_api_token.test", "permissions.0", "monitors:read"),
					// Plain token is stored sensitive — verify it was set
					resource.TestCheckResourceAttrSet("uptimewarden_api_token.test", "token"),
				),
			},
		},
	})
}
