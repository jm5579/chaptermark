// Package webheaders adds security headers to everything the app's embedded
// browser loads from the app itself.
package webheaders

import "net/http"

// CSP limits the page to the app's own scripts and styles and to cover images
// from the two image hosts. If text from an API or an import file were ever
// treated as HTML, the browser would still refuse to run any script from it,
// which keeps page scripts away from the app's native functions.
const CSP = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: https://cdn.mangaupdates.com https://uploads.mangadex.org; " +
	"connect-src 'self'; " +
	"font-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"form-action 'none'; " +
	"frame-src 'none'"

// Middleware sets the security headers on every response. csp is empty in
// development, where Vite's hot reload needs inline scripts.
func Middleware(csp string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			if csp != "" {
				h.Set("Content-Security-Policy", csp)
			}
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "no-referrer")
			next.ServeHTTP(w, r)
		})
	}
}
