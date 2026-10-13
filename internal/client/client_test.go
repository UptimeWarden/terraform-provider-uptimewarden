package client_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/client"
)

func TestGetMonitor_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"success":false,"error":{"code":"MONITOR_NOT_FOUND","message":"Monitor not found"}}`)
	}))
	defer srv.Close()

	_, err := client.New(srv.URL, "test-token").GetMonitor(context.Background(), "01HWTEST0000000000000009")
	if !client.IsNotFoundErr(err) {
		t.Errorf("IsNotFoundErr: want true; err=%v", err)
	}
}

func TestDeleteMonitor_AlreadyGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	err := client.New(srv.URL, "test-token").DeleteMonitor(context.Background(), "gone")
	if !client.IsNotFoundErr(err) {
		t.Errorf("IsNotFoundErr: want true; err=%v", err)
	}
}

func TestAPIError_NonEnvelopeBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"message":"Invalid token"}`)
	}))
	defer srv.Close()

	_, err := client.New(srv.URL, "bad-token").GetMonitor(context.Background(), "any")
	if err == nil || client.IsNotFoundErr(err) {
		t.Fatalf("want a non-404 error, got %v", err)
	}
	if got := err.Error(); got != "UptimeWarden API 401: Invalid token" {
		t.Errorf("message: %q", got)
	}
}
