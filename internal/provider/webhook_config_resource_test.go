package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func fakeWebhookConfigServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/webhooks/configs/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"publicId":    "wh_01HWTEST0000000000000001",
			"typeId":      1,
			"name":        "My Webhook",
			"url":         "https://hooks.example.com/alert",
			"httpMethod":  "POST",
			"timeout":     30,
			"retryCount":  3,
			"retryDelay":  60,
			"isActive":    true
		}`)
	})

	mux.HandleFunc("GET /api/webhooks/configs/wh_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":    "wh_01HWTEST0000000000000001",
			"typeId":      1,
			"name":        "My Webhook",
			"url":         "https://hooks.example.com/alert",
			"httpMethod":  "POST",
			"timeout":     30,
			"retryCount":  3,
			"retryDelay":  60,
			"isActive":    true
		}`)
	})

	mux.HandleFunc("PATCH /api/webhooks/configs/wh_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{
			"publicId":    "wh_01HWTEST0000000000000001",
			"typeId":      1,
			"name":        "My Webhook Updated",
			"url":         "https://hooks.example.com/alert",
			"httpMethod":  "POST",
			"timeout":     30,
			"retryCount":  3,
			"retryDelay":  60,
			"isActive":    true
		}`)
	})

	mux.HandleFunc("DELETE /api/webhooks/configs/wh_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestWebhookConfigResource_CRUD(t *testing.T) {
	srv := fakeWebhookConfigServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_webhook_config" "test" {
  type_id     = 1
  name        = "My Webhook"
  url         = "https://hooks.example.com/alert"
  http_method = "POST"
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "id", "wh_01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "type_id", "1"),
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "http_method", "POST"),
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "is_active", "true"),
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "timeout", "30"),
					resource.TestCheckResourceAttr("uptimewarden_webhook_config.test", "retry_count", "3"),
				),
			},
		},
	})
}

func TestWebhookConfigResource_InvalidMethod(t *testing.T) {
	srv := fakeWebhookConfigServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_webhook_config" "test" {
  type_id     = 1
  name        = "Bad"
  url         = "https://example.com"
  http_method = "CONNECT"
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`value must be one of:`)},
		},
	})
}
