package covers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jm5579/chaptermark/internal/httpx"
)

// allowLocalTestServer lets downloads reach the plain-HTTP 127.0.0.1 test
// servers for the duration of a test; production only allows the HTTPS image hosts.
func allowLocalTestServer(t *testing.T) {
	t.Helper()
	oldHosts, oldHTTPS := allowedHosts, requireHTTPS
	allowedHosts = map[string]bool{"127.0.0.1": true}
	requireHTTPS = false
	t.Cleanup(func() { allowedHosts, requireHTTPS = oldHosts, oldHTTPS })
}

func TestAllowedURL(t *testing.T) {
	for _, ok := range []string{
		"https://cdn.mangaupdates.com/image/i519012.jpg",
		"https://uploads.mangadex.org/covers/9a414441/be18dc9a.jpg.512.jpg",
		"https://CDN.MangaUpdates.com/image/x.jpg",
	} {
		if err := AllowedURL(ok); err != nil {
			t.Errorf("AllowedURL(%q) = %v, want allowed", ok, err)
		}
	}
	for _, bad := range []string{
		"http://cdn.mangaupdates.com/image/x.jpg",         // not HTTPS
		"https://evil.example/x.jpg",                      // unknown host
		"https://cdn.mangaupdates.com.evil.example/x.jpg", // look-alike host
		"https://192.168.1.1/admin",                       // home network
		"https://127.0.0.1:8080/x.jpg",
		"file:///C:/Windows/win.ini",
		"",
		"::not a url::",
	} {
		if err := AllowedURL(bad); err == nil {
			t.Errorf("AllowedURL(%q) = nil, want refused", bad)
		}
	}
}

func TestDownloadRefusesDisallowedURLWithoutRequesting(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	// srv is plain HTTP on 127.0.0.1, which production rules don't allow.
	if _, err := Download(context.Background(), httpx.New(0), srv.URL+"/x.jpg", t.TempDir(), 7); err == nil {
		t.Fatal("expected the URL to be refused")
	}
	if hit {
		t.Fatal("a refused URL must not be requested at all")
	}
}

func TestCheckRedirect(t *testing.T) {
	to := func(raw string) *http.Request {
		r, _ := http.NewRequest("GET", raw, nil)
		return r
	}
	if err := CheckRedirect(to("https://uploads.mangadex.org/covers/a/b.jpg"), nil); err != nil {
		t.Errorf("redirect to an image host refused: %v", err)
	}
	if err := CheckRedirect(to("http://192.168.0.1/"), nil); err == nil {
		t.Error("redirect to the home network allowed")
	}
	via := make([]*http.Request, 5)
	if err := CheckRedirect(to("https://cdn.mangaupdates.com/x.jpg"), via); err == nil {
		t.Error("more than 5 redirects allowed")
	}
}

func imageServer(t *testing.T, contentType string) string {
	t.Helper()
	allowLocalTestServer(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Write([]byte("IMG"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/cover"
}

func TestDownloadSavesByContentType(t *testing.T) {
	dir := t.TempDir()
	name, err := Download(context.Background(), httpx.New(0), imageServer(t, "image/png"), dir, 7)
	if err != nil || name != "7.png" {
		t.Fatalf("Download = %q, %v", name, err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "7.png")); string(b) != "IMG" {
		t.Fatalf("file content %q", b)
	}
}

func TestDownloadReplacesOldExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "7.jpg"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(dir, "17.jpg"), []byte("other"), 0o644)
	if _, err := Download(context.Background(), httpx.New(0), imageServer(t, "image/webp"), dir, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "7.jpg")); !os.IsNotExist(err) {
		t.Fatal("old 7.jpg should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "17.jpg")); err != nil {
		t.Fatal("17.jpg must not be touched")
	}
	if _, err := os.Stat(filepath.Join(dir, "7.webp")); err != nil {
		t.Fatal("7.webp missing")
	}
}

func TestDownloadRejectsNonImage(t *testing.T) {
	_, err := Download(context.Background(), httpx.New(0), imageServer(t, "text/html"), t.TempDir(), 7)
	if err == nil {
		t.Fatal("expected error for text/html")
	}
}

func TestHandlerServesFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "7.jpg"), []byte("JPEG"), 0o644)
	srv := httptest.NewServer(Handler(dir))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/covers/7.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	// no-cache: the webview must revalidate, so a re-downloaded cover (same
	// file name) is never shown stale.
	if resp.StatusCode != 200 || string(b) != "JPEG" || resp.Header.Get("Cache-Control") != "no-cache" {
		t.Fatalf("status=%d body=%q cache=%q", resp.StatusCode, b, resp.Header.Get("Cache-Control"))
	}
	lastMod := resp.Header.Get("Last-Modified")
	if lastMod == "" {
		t.Fatal("Last-Modified missing, so revalidation can't work")
	}

	// Revalidating an unchanged file is a cheap 304; a replaced file is re-sent.
	req, _ := http.NewRequest("GET", srv.URL+"/covers/7.jpg", nil)
	req.Header.Set("If-Modified-Since", lastMod)
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotModified {
		t.Fatalf("revalidation status = %d, want 304", resp2.StatusCode)
	}
	later := time.Now().Add(2 * time.Hour)
	os.WriteFile(filepath.Join(dir, "7.jpg"), []byte("NEWJPEG"), 0o644)
	os.Chtimes(filepath.Join(dir, "7.jpg"), later, later)
	resp3, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b3, _ := io.ReadAll(resp3.Body)
	resp3.Body.Close()
	if resp3.StatusCode != 200 || string(b3) != "NEWJPEG" {
		t.Fatalf("replaced cover: status=%d body=%q", resp3.StatusCode, b3)
	}
}

func TestMiddlewareRoutesCoversBeforeAssets(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "7.jpg"), []byte("JPEG"), 0o644)
	spa := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("index.html")) })
	srv := httptest.NewServer(Middleware(dir)(spa))
	defer srv.Close()
	for path, want := range map[string]string{"/covers/7.jpg": "JPEG", "/": "index.html", "/assets/app.js": "index.html"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if string(b) != want {
			t.Errorf("GET %s = %q, want %q", path, b, want)
		}
	}
}

func TestHandlerRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(filepath.Dir(dir), "library.db"), []byte("secret"), 0o644)
	srv := httptest.NewServer(Handler(dir))
	defer srv.Close()
	for _, p := range []string{"/covers/..%2Flibrary.db", "/covers/../library.db", "/covers/7.exe", "/other/7.jpg"} {
		resp, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", p, resp.StatusCode)
		}
	}
}
