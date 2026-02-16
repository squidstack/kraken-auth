package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogRequests(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	t.Run("logs regular requests", func(t *testing.T) {
		buf.Reset()
		middleware := LogRequests()
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("User-Agent", "test-agent")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		output := buf.String()
		if !strings.Contains(output, "GET") {
			t.Errorf("log should contain method GET, got: %q", output)
		}
		if !strings.Contains(output, "/test") {
			t.Errorf("log should contain path /test, got: %q", output)
		}
		if !strings.Contains(output, "status=200") {
			t.Errorf("log should contain status=200, got: %q", output)
		}
		if !strings.Contains(output, "test-agent") {
			t.Errorf("log should contain user-agent, got: %q", output)
		}
	})

	t.Run("logs different status codes", func(t *testing.T) {
		buf.Reset()
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		middleware := LogRequests()
		wrapped := middleware(errorHandler)

		req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		output := buf.String()
		if !strings.Contains(output, "status=404") {
			t.Errorf("log should contain status=404, got: %q", output)
		}
	})

	t.Run("skips configured paths", func(t *testing.T) {
		buf.Reset()
		middleware := LogRequests(WithSkips("/health", "/ready"))
		wrapped := middleware(handler)

		tests := []struct {
			name       string
			path       string
			shouldLog  bool
		}{
			{"skips /health", "/health", false},
			{"skips /ready", "/ready", false},
			{"logs /api", "/api", true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf.Reset()
				req := httptest.NewRequest(http.MethodGet, tt.path, nil)
				rec := httptest.NewRecorder()

				wrapped.ServeHTTP(rec, req)

				output := buf.String()
				hasLog := len(output) > 0 && strings.Contains(output, tt.path)
				if hasLog != tt.shouldLog {
					t.Errorf("path %q: shouldLog=%v, got output=%q", tt.path, tt.shouldLog, output)
				}
			})
		}
	})

	t.Run("multiple skip options", func(t *testing.T) {
		buf.Reset()
		middleware := LogRequests(WithSkips("/health"), WithSkips("/metrics"))
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if buf.Len() > 0 {
			t.Errorf("should skip /health, got output: %q", buf.String())
		}

		buf.Reset()
		req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
		rec = httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if buf.Len() > 0 {
			t.Errorf("should skip /metrics, got output: %q", buf.String())
		}
	})
}

func TestWrap(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		w := &wrap{ResponseWriter: rec, status: 200}

		w.WriteHeader(http.StatusCreated)

		if w.status != http.StatusCreated {
			t.Errorf("wrap.status = %d, want %d", w.status, http.StatusCreated)
		}
		if rec.Code != http.StatusCreated {
			t.Errorf("underlying recorder.Code = %d, want %d", rec.Code, http.StatusCreated)
		}
	})

	t.Run("defaults to 200 when WriteHeader not called", func(t *testing.T) {
		rec := httptest.NewRecorder()
		w := &wrap{ResponseWriter: rec, status: 200}

		w.Write([]byte("body"))

		if w.status != 200 {
			t.Errorf("wrap.status = %d, want 200", w.status)
		}
	})
}

func TestWithSkips(t *testing.T) {
	t.Run("adds paths to skip set", func(t *testing.T) {
		o := &opts{skips: make(map[string]struct{})}
		fn := WithSkips("/health", "/ready", "/metrics")
		fn(o)

		expected := []string{"/health", "/ready", "/metrics"}
		for _, path := range expected {
			if _, exists := o.skips[path]; !exists {
				t.Errorf("path %q should be in skips map", path)
			}
		}

		if len(o.skips) != 3 {
			t.Errorf("skips map should have 3 entries, got %d", len(o.skips))
		}
	})
}

func TestHasPrefixIn(t *testing.T) {
	set := map[string]struct{}{
		"/api/":    {},
		"/admin/":  {},
		"/health":  {},
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"exact match", "/health", true},
		{"prefix match", "/api/users", true},
		{"prefix match admin", "/admin/settings", true},
		{"no match", "/public", false},
		{"no match empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasPrefixIn(tt.path, set)
			if got != tt.expected {
				t.Errorf("hasPrefixIn(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}
