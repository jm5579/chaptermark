package covers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
	if resp.StatusCode != 200 || string(b) != "JPEG" || resp.Header.Get("Cache-Control") != "max-age=86400" {
		t.Fatalf("status=%d body=%q cache=%q", resp.StatusCode, b, resp.Header.Get("Cache-Control"))
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
