package client_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

func TestCreateMonitor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/monitors" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-Source"); got != "terraform" {
			t.Errorf("X-Source header: want terraform, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header: want Bearer test-token, got %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["managedBySource"] != "terraform" {
			t.Errorf("managedBySource: want terraform, got %v", body["managedBySource"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{
			"success": true,
			"data": {
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

	c := client.New(srv.URL, "test-token")
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
	if m.PublicID != "01HWTEST0000000000000001" {
		t.Errorf("PublicID: want 01HWTEST0000000000000001, got %q", m.PublicID)
	}
	if m.ManagedBySource != "terraform" {
		t.Errorf("ManagedBySource: want terraform, got %q", m.ManagedBySource)
	}
}

func TestGetMonitor_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"Monitor not found"}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	_, err := c.GetMonitor(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !client.IsNotFoundErr(err) {
		t.Errorf("IsNotFoundErr: want true, got false; err=%v", err)
	}
}

func TestDeleteMonitor_AlreadyGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"Monitor not found"}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	err := c.DeleteMonitor(context.Background(), "gone")
	// 404 from delete is still an APIError — resource code handles it
	if !client.IsNotFoundErr(err) {
		t.Errorf("IsNotFoundErr: want true, got false; err=%v", err)
	}
}

func TestAPIError_Format(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Invalid token"}`)
	}))
	defer srv.Close()

	c := client.New(srv.URL, "bad-token")
	_, err := c.GetMonitor(context.Background(), "any")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if client.IsNotFoundErr(err) {
		t.Error("401 should not be IsNotFoundErr")
	}
}
