package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/provider"
)

const testMonitorID = "01HWTEST0000000000000001"

// testProviderFactories wires the provider for unit tests.
func testProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"uptimewarden": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// fakeAPI is a stateful stand-in for /api/v1/monitors. It behaves like the
// live API where it matters to the provider:
//   - POST applies the default timeout (30) whatever the body says;
//   - PATCH applies the fields it is sent (unless patchIgnoresTimeout);
//   - a monitor created without a region has none (the field is omitted).
type fakeAPI struct {
	mu                  sync.Mutex
	monitor             map[string]any
	patchIgnoresTimeout bool
	posts               []map[string]any
	patches             []map[string]any
}

func newFakeAPI(t *testing.T) (*fakeAPI, *httptest.Server) {
	t.Helper()
	api := &fakeAPI{}
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/monitors", func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		api.posts = append(api.posts, body)
		m := map[string]any{
			"publicId":        testMonitorID,
			"name":            body["name"],
			"type":            body["type"],
			"url":             body["url"],
			"interval":        300.0,
			"timeout":         30.0, // the create path ignores a requested timeout
			"isActive":        true,
			"managedBySource": "terraform",
		}
		if v, ok := body["interval"]; ok {
			m["interval"] = v
		}
		if v, ok := body["region"]; ok {
			m["region"] = v
		}
		if v, ok := body["isActive"]; ok {
			m["isActive"] = v
		}
		api.monitor = m
		writeEnvelope(w, http.StatusCreated, m)
	})

	mux.HandleFunc("GET /api/v1/monitors/"+testMonitorID, func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		if api.monitor == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		writeEnvelope(w, http.StatusOK, api.monitor)
	})

	mux.HandleFunc("PATCH /api/v1/monitors/"+testMonitorID, func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		api.patches = append(api.patches, body)
		for k, v := range body {
			if k == "timeout" && api.patchIgnoresTimeout {
				continue
			}
			api.monitor[k] = v
		}
		writeEnvelope(w, http.StatusOK, api.monitor)
	})

	mux.HandleFunc("DELETE /api/v1/monitors/"+testMonitorID, func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		defer api.mu.Unlock()
		api.monitor = nil
		writeEnvelope(w, http.StatusOK, map[string]any{"message": "deleted"})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return api, srv
}

func writeEnvelope(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": data})
}

func providerBlock(url string) string {
	return fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "test-token"
  base_url  = %q
}
`, url)
}

func TestMonitorResource_CreateReadUpdateDelete(t *testing.T) {
	_, srv := newFakeAPI(t)
	const res = "uptimewarden_monitor.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name     = "Test Monitor"
  type     = "http"
  url      = "https://example.com"
  interval = 60
  region   = "gwc"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", testMonitorID),
					resource.TestCheckResourceAttr(res, "name", "Test Monitor"),
					resource.TestCheckResourceAttr(res, "type", "http"),
					resource.TestCheckResourceAttr(res, "interval", "60"),
					resource.TestCheckResourceAttr(res, "timeout", "30"),
					resource.TestCheckResourceAttr(res, "region", "gwc"),
					resource.TestCheckResourceAttr(res, "is_active", "true"),
				),
			},
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name     = "Test Monitor Updated"
  type     = "http"
  url      = "https://example.com/health"
  interval = 120
  region   = "sea"
}
`,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "name", "Test Monitor Updated"),
					resource.TestCheckResourceAttr(res, "url", "https://example.com/health"),
					resource.TestCheckResourceAttr(res, "interval", "120"),
					resource.TestCheckResourceAttr(res, "timeout", "30"),
					resource.TestCheckResourceAttr(res, "region", "sea"),
				),
			},
		},
	})
}

// A configured timeout survives the create path that ignores it: the provider
// sets it with a PATCH and the plan is empty afterwards (no perpetual diff,
// no "inconsistent result after apply").
func TestMonitorResource_TimeoutSetAfterCreate(t *testing.T) {
	api, srv := newFakeAPI(t)
	const res = "uptimewarden_monitor.test"
	config := providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name    = "Slow endpoint"
  type    = "http"
  url     = "https://example.com/slow"
  timeout = 60
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check:  resource.TestCheckResourceAttr(res, "timeout", "60"),
			},
			{Config: config, PlanOnly: true},
		},
	})
	if len(api.patches) == 0 {
		t.Fatal("expected a follow-up PATCH carrying the timeout")
	}
	if got := api.patches[0]["timeout"]; got != 60.0 {
		t.Fatalf("follow-up PATCH timeout: want 60, got %v", got)
	}
}

// Unset timeout and region read back what the API reports: the API default
// timeout and a null region for an automatically placed monitor.
func TestMonitorResource_UnsetTimeoutAndRegionReadBack(t *testing.T) {
	api, srv := newFakeAPI(t)
	const res = "uptimewarden_monitor.test"
	config := providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name = "Homepage"
  type = "http"
  url  = "https://example.com"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(res, "timeout", "30"),
					resource.TestCheckNoResourceAttr(res, "region"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
	if _, sent := api.posts[0]["timeout"]; sent {
		t.Fatalf("unset timeout must not be sent on create: %v", api.posts[0])
	}
	if _, sent := api.posts[0]["region"]; sent {
		t.Fatalf("unset region must not be sent on create: %v", api.posts[0])
	}
	if len(api.patches) != 0 {
		t.Fatalf("no follow-up PATCH expected when timeout is unset, got %v", api.patches)
	}
}

// When the API does not apply a configured value, the error names the field
// and both values instead of Terraform's generic inconsistency error.
func TestMonitorResource_TimeoutNotAppliedIsReported(t *testing.T) {
	api, srv := newFakeAPI(t)
	api.patchIgnoresTimeout = true
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name    = "Slow endpoint"
  type    = "http"
  url     = "https://example.com/slow"
  timeout = 90
}
`,
				ExpectError: regexp.MustCompile(`timeout: configured 90, API reports 30`),
			},
		},
	})
}

func TestMonitorResource_InvalidType(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
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

// uaen (Dubai) is retired; the API refuses it, so the provider does too.
func TestMonitorResource_RetiredRegionRejected(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name   = "Dubai"
  type   = "http"
  url    = "https://example.com"
  region = "uaen"
}
`,
				ExpectError: regexp.MustCompile(`value must be one of:`),
			},
		},
	})
}

func TestMonitorResource_Import(t *testing.T) {
	_, srv := newFakeAPI(t)
	config := providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "imported" {
  name     = "Test Monitor"
  type     = "http"
  url      = "https://example.com"
  interval = 60
  region   = "gwc"
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config},
			{
				Config:            config,
				ResourceName:      "uptimewarden_monitor.imported",
				ImportState:       true,
				ImportStateId:     testMonitorID,
				ImportStateVerify: true,
			},
		},
	})
}
