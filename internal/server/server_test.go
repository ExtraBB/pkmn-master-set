package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	h := New().Routes()

	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"index", http.MethodGet, "/", "<h1>pkmn-master-set</h1>"},
		{"ping fragment", http.MethodPost, "/ping", "pong"},
		{"static asset", http.MethodGet, "/static/css/app.css", "--accent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tt.want) {
				t.Errorf("body does not contain %q:\n%s", tt.want, rec.Body.String())
			}
		})
	}
}
