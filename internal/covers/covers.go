// Package covers caches cover images in data/covers so the list renders offline.
package covers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jm5579/chaptermark/internal/httpx"
)

var validName = regexp.MustCompile(`^\d+\.(jpg|png|webp)$`)

func extFor(contentType string) (string, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	switch {
	case ct == "image/png":
		return "png", nil
	case ct == "image/webp":
		return "webp", nil
	case strings.HasPrefix(ct, "image/"), ct == "application/octet-stream", ct == "":
		return "jpg", nil
	}
	return "", fmt.Errorf("not an image: %q", contentType)
}

// Download saves the image at url as <seriesID>.<ext> in dir, replacing any
// earlier cover for that series, and returns the file name.
func Download(ctx context.Context, c *httpx.Client, url, dir string, seriesID int64) (string, error) {
	data, contentType, err := c.GetBytes(ctx, url)
	if err != nil {
		return "", err
	}
	ext, err := extFor(contentType)
	if err != nil {
		return "", err
	}
	id := strconv.FormatInt(seriesID, 10)
	name := id + "." + ext
	tmp := filepath.Join(dir, name+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	for _, old := range []string{"jpg", "png", "webp"} {
		if old != ext {
			os.Remove(filepath.Join(dir, id+"."+old))
		}
	}
	if err := os.Rename(tmp, filepath.Join(dir, name)); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return name, nil
}

// Middleware serves /covers/ requests from dir and passes everything else to
// next. It runs in front of the app's assets, so covers work both in the built
// exe and under the dev server (which would otherwise answer every path).
func Middleware(dir string) func(next http.Handler) http.Handler {
	covers := Handler(dir)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/covers/") {
				covers.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Handler serves GET /covers/<seriesID>.<ext> from dir; anything else is 404.
func Handler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := strings.CutPrefix(r.URL.Path, "/covers/")
		if !ok || !validName.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "max-age=86400")
		http.ServeFile(w, r, filepath.Join(dir, name))
	})
}
