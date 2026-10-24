package webheaders

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, csp, path string) http.Header {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("<html></html>"))
	})
	rec := httptest.NewRecorder()
	Middleware(csp)(next).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec.Header()
}

func TestProductionHeaders(t *testing.T) {
	for _, path := range []string{"/", "/assets/index.js", "/covers/1.jpg"} {
		h := serve(t, CSP, path)
		if h.Get("Content-Security-Policy") != CSP {
			t.Errorf("%s: CSP = %q", path, h.Get("Content-Security-Policy"))
		}
		if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Referrer-Policy") != "no-referrer" {
			t.Errorf("%s: headers = %v", path, h)
		}
	}
}

func TestDevModeSkipsCSP(t *testing.T) {
	h := serve(t, "", "/")
	if h.Get("Content-Security-Policy") != "" {
		t.Fatalf("dev mode must not send a CSP (Vite's hot reload needs inline scripts)")
	}
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("nosniff missing in dev mode")
	}
}

func TestCSPIsStrict(t *testing.T) {
	for _, want := range []string{
		"default-src 'self'",
		"script-src 'self';", // no 'unsafe-inline' / 'unsafe-eval' for scripts
		"object-src 'none'",
		"base-uri 'none'",
		"form-action 'none'",
		"frame-src 'none'",
		"img-src 'self' data: https://cdn.mangaupdates.com https://uploads.mangadex.org",
	} {
		if !strings.Contains(CSP+";", want) {
			t.Errorf("CSP missing %q: %s", want, CSP)
		}
	}
}
