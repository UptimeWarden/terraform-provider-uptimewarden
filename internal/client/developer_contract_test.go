package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

// Developer contract: API tokens (`uptw_`) authenticate only on /api/v1/*.
// The Terraform client talks to that surface, by public id, and unwraps
// {success,data}.

const pid = "01HWTEST0000000000000001"

func TestCreateMonitor_V1PathEnvelopeAndSource(t *testing.T) {
	var gotPath, gotAuth, gotSource string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotSource = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("X-Source")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"success":true,"data":{"publicId":%q,"name":"My API","type":"http","timeout":45}}`, pid)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	m, err := c.CreateMonitor(context.Background(), map[string]any{"name": "My API", "type": "http", "timeout": 45})
	if err != nil {
		t.Fatalf("CreateMonitor: %v", err)
	}
	if gotPath != "/api/v1/monitors" || gotAuth != "Bearer uptw_test" || gotSource != "terraform" {
		t.Errorf("request: path %s auth %q source %q", gotPath, gotAuth, gotSource)
	}
	if body["managedBySource"] != "terraform" || body["timeout"] != 45.0 {
		t.Errorf("body: %v", body)
	}
	if m.PublicID() != pid || m["timeout"] != 45.0 {
		t.Errorf("unwrap: %v", m)
	}
}

func TestMonitorPaths_UsePublicID(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"data":{"publicId":%q,"name":"x"}}`, pid)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	ctx := context.Background()
	if _, err := c.GetMonitor(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if _, err := c.UpdateMonitor(ctx, pid, map[string]any{"name": "x"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteMonitor(ctx, pid); err != nil {
		t.Fatal(err)
	}
	want := []string{"GET /api/v1/monitors/" + pid, "PATCH /api/v1/monitors/" + pid, "DELETE /api/v1/monitors/" + pid}
	if strings.Join(calls, ",") != strings.Join(want, ",") {
		t.Errorf("calls: got %v want %v", calls, want)
	}
}

func TestAPIError_V1Envelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"success":false,"error":{"code":"VALIDATION_ERROR","message":"Invalid monitoring region","details":["Valid regions: ae, brs, gwc, sea, wus3"]}}`)
	}))
	defer srv.Close()

	_, err := client.New(srv.URL, "uptw_test").CreateMonitor(context.Background(), map[string]any{"region": "uaen"})
	want := "UptimeWarden API 400 VALIDATION_ERROR: Invalid monitoring region (Valid regions: ae, brs, gwc, sea, wus3)"
	if err == nil || err.Error() != want {
		t.Fatalf("error: got %v, want %q", err, want)
	}
}

func TestReferenceLists(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/escalation-policies", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":[{"publicId":"01HWPOL0000000000000000001","name":"P1","stepCount":2,"createdAt":"c","updatedAt":"u"}]}`)
	})
	mux.HandleFunc("GET /api/v1/webhooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":[{"publicId":"01HWHOOK000000000000000001","name":"Slack","description":null,"messageFormat":"slack","isActive":true,"enabledAlertTypes":["down"],"enabledMonitorTypes":[],"createdAt":"c"}]}`)
	})
	mux.HandleFunc("GET /api/v1/regions", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"success":true,"data":[{"code":"gwc","name":"Sweden Central","description":"Europe","isDefault":true}],"plan":{"regionSelection":false,"maxRegions":1,"maxVerificationRegions":2,"defaultConfirmingRegions":2,"allowConsensusOverride":false}}`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := client.New(srv.URL, "uptw_test")
	ctx := context.Background()

	pols, err := c.ListEscalationPolicies(ctx)
	if err != nil || len(pols) != 1 || pols[0].Name != "P1" || pols[0].StepCount != 2 {
		t.Fatalf("policies: %v %v", pols, err)
	}
	hooks, err := c.ListWebhooks(ctx)
	if err != nil || len(hooks) != 1 || hooks[0].PublicID != "01HWHOOK000000000000000001" || hooks[0].Description != nil {
		t.Fatalf("webhooks: %v %v", hooks, err)
	}
	regs, err := c.ListRegions(ctx)
	if err != nil || len(regs.Regions) != 1 || regs.Plan == nil || regs.Plan.MaxVerificationRegions != 2 {
		t.Fatalf("regions: %+v %v", regs, err)
	}
}

// Region labels follow the regions table the dashboard reads (migrations
// 0272 and 0372): ae Sydney, brs São Paulo, gwc Sweden Central, sea East Asia,
// wus3 Phoenix. uaen (Dubai) is retired and must not be offered.
func TestRegionDocs_MatchRegionsTable(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repo := filepath.Join(filepath.Dir(thisFile), "..", "..")
	for _, rel := range []string{
		"internal/provider/monitor_fields.go",
		"docs/resources/monitor.md",
		"README.md",
	} {
		src, err := os.ReadFile(filepath.Join(repo, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		text := string(src)
		for _, want := range []string{
			"`ae` (Sydney)", "`brs` (São Paulo)", "`gwc` (Sweden Central)", "`sea` (East Asia)", "`wus3` (Phoenix)",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s: missing region label %s", rel, want)
			}
		}
		for _, stale := range []string{"uaen", "Frankfurt", "Singapore", "Dubai"} {
			if strings.Contains(text, stale) {
				t.Errorf("%s: stale region reference %q", rel, stale)
			}
		}
	}
}
