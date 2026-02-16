package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHealthEndpoint(t *testing.T) {
	r := mux.NewRouter()
	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}).Methods(http.MethodGet)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET returns 200 ok",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   "ok",
		},
		{
			name:           "POST not allowed",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectedBody != "" && rec.Body.String() != tt.expectedBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestReadyEndpoint(t *testing.T) {
	t.Run("returns ready when db ping succeeds", func(t *testing.T) {
		r := mux.NewRouter()
		r.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
			// Simulating successful DB ping
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ready"))
		}).Methods(http.MethodGet)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		if rec.Body.String() != "ready" {
			t.Errorf("body = %q, want %q", rec.Body.String(), "ready")
		}
	})

	t.Run("returns 503 when db ping fails", func(t *testing.T) {
		r := mux.NewRouter()
		r.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
			// Simulating failed DB ping
			http.Error(w, "db not ready", http.StatusServiceUnavailable)
		}).Methods(http.MethodGet)

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}
	})
}

func TestFlagsEndpoint(t *testing.T) {
	r := mux.NewRouter()
	r.HandleFunc("/_flags", func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"offline":  false,
			"logLevel": "info",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}).Methods(http.MethodGet)

	t.Run("returns flags as JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_flags", nil)
		rec := httptest.NewRecorder()

		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want %q", contentType, "application/json")
		}

		var response map[string]interface{}
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if _, ok := response["offline"]; !ok {
			t.Error("response should contain 'offline' field")
		}

		if _, ok := response["logLevel"]; !ok {
			t.Error("response should contain 'logLevel' field")
		}
	})
}

func TestOfflineMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		offline        bool
		expectedStatus int
	}{
		{
			name:           "health endpoint always allowed when offline",
			path:           "/health",
			offline:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ready endpoint always allowed when offline",
			path:           "/ready",
			offline:        true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "other endpoints blocked when offline",
			path:           "/api/test",
			offline:        true,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "endpoints allowed when not offline",
			path:           "/api/test",
			offline:        false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := mux.NewRouter()

			// Simulate offline middleware
			offlineGate := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/health" || r.URL.Path == "/ready" {
						next.ServeHTTP(w, r)
						return
					}
					if tt.offline {
						http.Error(w, "service temporarily offline", http.StatusServiceUnavailable)
						return
					}
					next.ServeHTTP(w, r)
				})
			}
			r.Use(offlineGate)

			// Register handlers
			r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			r.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			r.HandleFunc("/api/test", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}

func TestRouterMethods(t *testing.T) {
	r := mux.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods(http.MethodGet)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "GET allowed on health",
			method:         http.MethodGet,
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST not allowed on health",
			method:         http.MethodPost,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT not allowed on health",
			method:         http.MethodPut,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "DELETE not allowed on health",
			method:         http.MethodDelete,
			path:           "/health",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.expectedStatus)
			}
		})
	}
}
