package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func fakeOnCallScheduleServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	schedule := `{
		"publicId":  "oc_01HWTEST0000000000000001",
		"name":      "Primary On-Call",
		"timezone":  "UTC",
		"createdAt": "2026-01-01T00:00:00.000Z",
		"updatedAt": "2026-01-01T00:00:00.000Z",
		"members": [
			{"userId": 1, "dayOfWeek": 1},
			{"userId": 2, "dayOfWeek": 4}
		]
	}`

	mux.HandleFunc("POST /api/on-call/schedules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, schedule)
	})

	mux.HandleFunc("GET /api/on-call/schedules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[%s]", schedule)
	})

	mux.HandleFunc("PUT /api/on-call/schedules/oc_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("DELETE /api/on-call/schedules/oc_01HWTEST0000000000000001", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestOnCallScheduleResource_CRUD(t *testing.T) {
	srv := fakeOnCallScheduleServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_on_call_schedule" "test" {
  name     = "Primary On-Call"
  timezone = "UTC"
  members = [
    { user_id = 1, day_of_week = 1 },
    { user_id = 2, day_of_week = 4 },
  ]
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "id", "oc_01HWTEST0000000000000001"),
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "name", "Primary On-Call"),
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "timezone", "UTC"),
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "members.#", "2"),
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "members.0.user_id", "1"),
					resource.TestCheckResourceAttr("uptimewarden_on_call_schedule.test", "members.0.day_of_week", "1"),
				),
			},
		},
	})
}

func TestOnCallScheduleResource_InvalidDayOfWeek(t *testing.T) {
	srv := fakeOnCallScheduleServer(t)
	cfg := fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "t"
  base_url  = %q
}
resource "uptimewarden_on_call_schedule" "test" {
  name = "Bad"
  members = [
    { user_id = 1, day_of_week = 7 }
  ]
}`, srv.URL)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(srv.URL),
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`Attribute members\[0\]\.day_of_week value must be between 0 and 6`)},
		},
	})
}
