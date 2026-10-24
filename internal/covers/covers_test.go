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

func imageServer(t *testing.T, contentType string) string {
	t.Helper()
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
