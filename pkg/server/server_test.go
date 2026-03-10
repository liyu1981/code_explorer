package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerSetup(t *testing.T) {
	// We pass nil index because we just want to test route registration
	// Some handlers might panic if index is nil when called,
	// but SetupRoutes itself should be fine.
	handler := New(nil)

	if handler == nil {
		t.Fatalf("New() returned nil handler")
	}

	// Test a few basic routes
	tests := []struct {
		method string
		path   string
		status int
	}{
		{"GET", "/api/version", http.StatusOK},
		{"GET", "/health", http.StatusOK},
		{"GET", "/non-existent", http.StatusNotFound},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != tt.status {
			t.Errorf("%s %s got status %d; want %d", tt.method, tt.path, w.Code, tt.status)
		}
	}
}
