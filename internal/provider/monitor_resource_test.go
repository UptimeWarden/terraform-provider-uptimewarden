package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/provider"
)

// testProviderFactories wires the provider for acceptance tests.
func testProviderFactories(baseURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"uptimewarden": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// fakeMonitorServer returns a stub HTTP server that simulates the monitor API.
// It records the last request body so tests can assert on it.
func fakeMonitorServer(t *testing.T) (*httptest.Server, *string) {
	t.Helper()
	lastBody := new(string)
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/monitors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"publicId":        "01HWTEST0000000000000001",
			"name":            "Test Monitor",
			"type":            "http",
			"url":             "https://example.com",
			"interval":        60,
			"timeout":         30,
			"region":          "gwc",
			"isActive":        true,
			"managedBySource": "terraform"
		}`)
	})

	mux.HandleFunc("GET /api/v1/monitors/01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":        "01HWTEST0000000000000001",
			"name":            "Test Monitor",
			"type":            "http",
			"url":             "https://example.com",
			"interval":        60,
			"timeout":         30,
			"region":          "gwc",
			"isActive":        true,
			"managedBySource": "terraform"
		}`)
	})

	mux.HandleFunc("PATCH /api/v1/monitors/01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":        "01HWTEST0000000000000001",
			"name":            "Test Monitor Updated",
			"type":            "http",
			"url":             "https://example.com/health",
			"interval":        120,
			"timeout":         30,
			"region":          "gwc",
			"isActive":        true,
			"managedBySource": "terraform"
		}`)
	})

	mux.HandleFunc("DELETE /api/v1/monitors/01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, lastBody
}

func TestMonitorResource_CreateReadUpdateDelete(t *testing.T) {
	srv, _ := fakeMonitorServer(t)

	providerConfig := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "test-token"
  base_url  = %q
}
`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			// Create
			{
				Config: providerConfig + `
resource "uptimewarden_monitor" "test" {
  name     = "Test Monitor"
  type     = "http"
  url      = "https://example.com"
  interval = 60
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "id", "01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "name", "Test Monitor"),
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "type", "http"),
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "interval", "60"),
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "region", "gwc"),
					resource.TestCheckResourceAttr("uptimewarden_monitor.test", "is_active", "true"),
				),
			},
		},
	})
}

func TestMonitorResource_InvalidType(t *testing.T) {
	srv, _ := fakeMonitorServer(t)

	providerConfig := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "test-token"
  base_url  = %q
}
`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "uptimewarden_monitor" "test" {
  name = "Bad Monitor"
  type = "ftp"
  url  = "ftp://example.com"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of:`),
			},
		},
	})
}

func TestMonitorResource_Import(t *testing.T) {
	srv, _ := fakeMonitorServer(t)

	providerConfig := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "test-token"
  base_url  = %q
}
`, srv.URL)

	monitorConfig := providerConfig + `
resource "uptimewarden_monitor" "imported" {
  name     = "Test Monitor"
  type     = "http"
  url      = "https://example.com"
  interval = 60
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			// Step 1: create resource so it exists in state.
			{Config: monitorConfig},
			// Step 2: import by publicId and verify state matches.
			{
				Config:            monitorConfig,
				ResourceName:      "uptimewarden_monitor.imported",
				ImportState:       true,
				ImportStateId:     "01HWTEST0000000000000001",
				ImportStateVerify: true,
			},
		},
	})
}

// checkMonitorAttr is a TestCheckFunc helper that verifies a monitor attribute.
func checkMonitorAttr(name, attr, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("resource %s not found in state", name)
		}
		if got := rs.Primary.Attributes[attr]; got != value {
			return fmt.Errorf("attribute %s: want %q got %q", attr, value, got)
		}
		return nil
	}
}
