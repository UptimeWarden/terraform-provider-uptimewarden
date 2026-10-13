package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func fakeStatusPageServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/status-pages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"publicId":           "sp_01HWTEST0000000000000001",
			"name":               "Acme Status",
			"title":              "Acme System Status",
			"slug":               "acme-status",
			"isPublic":           true,
			"allowSubscriptions": true,
			"showHistory":        true,
			"historyDays":        90,
			"showResponseTime":   true,
			"showUptime":         true,
			"hidePoweredBy":      false
		}`)
	})

	mux.HandleFunc("GET /api/status-pages/sp_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":           "sp_01HWTEST0000000000000001",
			"name":               "Acme Status",
			"title":              "Acme System Status",
			"slug":               "acme-status",
			"isPublic":           true,
			"allowSubscriptions": true,
			"showHistory":        true,
			"historyDays":        90,
			"showResponseTime":   true,
			"showUptime":         true,
			"hidePoweredBy":      false
		}`)
	})

	mux.HandleFunc("PATCH /api/status-pages/sp_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":           "sp_01HWTEST0000000000000001",
			"name":               "Acme Status",
			"title":              "Acme System Status",
			"slug":               "acme-status",
			"isPublic":           true,
			"allowSubscriptions": true,
			"showHistory":        true,
			"historyDays":        90,
			"showResponseTime":   true,
			"showUptime":         true,
			"hidePoweredBy":      false
		}`)
	})

	mux.HandleFunc("DELETE /api/status-pages/sp_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestStatusPageResource_CRUD(t *testing.T) {
	srv := fakeStatusPageServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_status_page" "test" {
  name  = "Acme Status"
  title = "Acme System Status"
  slug  = "acme-status"
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_status_page.test", "id", "sp_01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_status_page.test", "slug", "acme-status"),
					resource.TestCheckResourceAttr("uptimewarden_status_page.test", "is_public", "true"),
					resource.TestCheckResourceAttr("uptimewarden_status_page.test", "history_days", "90"),
				),
			},
		},
	})
}
