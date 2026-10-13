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
// The Terraform client talks to that surface and unwraps {success,data}.

func TestCreateMonitor_V1PathAndEnvelope(t *testing.T) {
	var gotPath, gotAuth, gotSource string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotSource = r.Header.Get("X-Source")
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"success": true,
			"data": {
				"id": 42,
				"publicId": "01HWTEST0000000000000001",
				"name": "My API",
				"type": "http",
				"url": "https://api.example.com/health",
				"interval": 60,
				"timeout": 30,
				"region": "gwc",
				"isActive": true,
				"managedBySource": "terraform"
			}
		}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	active := true
	m, err := c.CreateMonitor(context.Background(), client.CreateMonitorRequest{
		Name:     "My API",
		Type:     "http",
		URL:      "https://api.example.com/health",
		Interval: 60,
		Region:   "gwc",
		IsActive: &active,
	})
	if err != nil {
		t.Fatalf("CreateMonitor: %v", err)
	}
	if gotPath != "/api/v1/monitors" {
		t.Errorf("path: want /api/v1/monitors, got %s", gotPath)
	}
	if gotAuth != "Bearer uptw_test" {
		t.Errorf("Authorization: want Bearer uptw_test, got %q", gotAuth)
	}
	if gotSource != "terraform" {
		t.Errorf("X-Source: want terraform, got %q", gotSource)
	}
	if body["managedBySource"] != "terraform" {
		t.Errorf("managedBySource: want terraform, got %v", body["managedBySource"])
	}
	if m.PublicID != "01HWTEST0000000000000001" {
		t.Errorf("PublicID: got %q", m.PublicID)
	}
}

func TestGetMonitor_V1ULIDPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":{"id":42,"publicId":"01HWTEST0000000000000001","name":"x","type":"http","url":"https://x","interval":60,"timeout":30,"isActive":true}}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	m, err := c.GetMonitor(context.Background(), "01HWTEST0000000000000001")
	if err != nil {
		t.Fatalf("GetMonitor: %v", err)
	}
	if gotPath != "/api/v1/monitors/01HWTEST0000000000000001" {
		t.Errorf("path: want /api/v1/monitors/{publicId}, got %s", gotPath)
	}
	if m.PublicID != "01HWTEST0000000000000001" {
		t.Errorf("publicId unwrap: got %q", m.PublicID)
	}
}

func TestDeleteMonitor_V1ULIDPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"success":true,"message":"Monitor deleted successfully"}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	if err := c.DeleteMonitor(context.Background(), "01HWTEST0000000000000001"); err != nil {
		t.Fatalf("DeleteMonitor: %v", err)
	}
	if gotPath != "/api/v1/monitors/01HWTEST0000000000000001" {
		t.Errorf("want /api/v1/monitors/{publicId}, got %s", gotPath)
	}
}

func TestUpdateMonitor_V1Patch(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPatch {
			t.Errorf("method: want PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"success":true,"data":{"id":42,"publicId":"01HWTEST0000000000000001","name":"renamed","type":"http","url":"https://x","interval":60,"timeout":30,"isActive":true}}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "uptw_test")
	m, err := c.UpdateMonitor(context.Background(), "01HWTEST0000000000000001", client.UpdateMonitorRequest{
		Name: "renamed",
	})
	if err != nil {
		t.Fatalf("UpdateMonitor: %v", err)
	}
	if gotPath != "/api/v1/monitors/01HWTEST0000000000000001" {
		t.Errorf("path: want /api/v1/monitors/{publicId}, got %s", gotPath)
	}
	if m.Name != "renamed" {
		t.Errorf("name unwrap: got %q", m.Name)
	}
}

func TestRegionDocs_AEIsAustraliaEastNotDubai(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "provider")
	src, err := os.ReadFile(filepath.Join(root, "monitor_resource.go"))
	if err != nil {
		t.Fatalf("read monitor_resource.go: %v", err)
	}
	text := string(src)
	if strings.Contains(text, "`ae` (Dubai)") {
		t.Fatalf("schema still labels ae as Dubai; canonical is Australia East")
	}
	if !strings.Contains(text, "`ae` (Australia East)") {
		t.Fatalf("schema must label ae as Australia East")
	}
}
