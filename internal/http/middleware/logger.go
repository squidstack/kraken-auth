package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type opts struct {
	skips map[string]struct{}
}

type Option func(*opts)

func WithSkips(paths ...string) Option {
	return func(o *opts) {
		for _, p := range paths {
			o.skips[p] = struct{}{}
		}
	}
}

func LogRequests(options ...Option) func(http.Handler) http.Handler {
	o := &opts{skips: make(map[string]struct{})}
	for _, fn := range options {
		fn(o)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := o.skips[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			ww := &wrap{ResponseWriter: w, status: 200}
			next.ServeHTTP(ww, r)
			d := time.Since(start)
			log.Printf("%s %s status=%d dur=%s ua=%q", r.Method, r.URL.String(), ww.status, d, r.UserAgent())
		})
	}
}

type wrap struct {
	http.ResponseWriter
	status int
}

func (w *wrap) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// optional helper if you later want wildcard skips (not used above)
func hasPrefixIn(path string, set map[string]struct{}) bool {
	for p := range set {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}
