package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"jplearn/internal/config"
)

func newTestServer(t *testing.T, apiKey string, enableEPUB bool) *Server {
	t.Helper()
	cfg := &config.Config{
		APIKey:           apiKey,
		EnableEPUBUpload: enableEPUB,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(cfg, nil, nil, nil, nil, logger)
}

func TestAPIKeyMiddleware(t *testing.T) {
	const key = "secret-key"

	tests := []struct {
		name       string
		method     string
		header     http.Header
		apiKey     string
		wantStatus int
	}{
		{
			name:       "write without key is rejected",
			method:     http.MethodPost,
			apiKey:     key,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "write with wrong key is rejected",
			method:     http.MethodPost,
			header:     http.Header{"X-Api-Key": []string{"nope"}},
			apiKey:     key,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "write with correct key is allowed",
			method:     http.MethodPost,
			header:     http.Header{"X-Api-Key": []string{key}},
			apiKey:     key,
			wantStatus: http.StatusOK,
		},
		{
			name:       "write with bearer token is allowed",
			method:     http.MethodPost,
			header:     http.Header{"Authorization": []string{"Bearer " + key}},
			apiKey:     key,
			wantStatus: http.StatusOK,
		},
		{
			name:       "read without key is allowed",
			method:     http.MethodGet,
			apiKey:     key,
			wantStatus: http.StatusOK,
		},
		{
			name:       "write allowed when no key configured",
			method:     http.MethodPost,
			apiKey:     "",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			h := apiKeyMiddleware(tt.apiKey)(next)

			req := httptest.NewRequest(tt.method, "/api/articles", nil)
			if tt.header != nil {
				req.Header = tt.header
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestEPUBRouteGating(t *testing.T) {
	t.Run("disabled returns 404", func(t *testing.T) {
		srv := newTestServer(t, "", false)
		req := httptest.NewRequest(http.MethodPost, "/api/epubs/upload", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("enabled reaches handler", func(t *testing.T) {
		srv := newTestServer(t, "", true)
		// No multipart body: the handler runs and rejects with 400, proving the
		// route is registered rather than 404.
		req := httptest.NewRequest(http.MethodPost, "/api/epubs/upload", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
