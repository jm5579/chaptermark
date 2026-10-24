package mangaupdates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jm5579/chaptermark/internal/httpx"
	"github.com/jm5579/chaptermark/internal/store"
)

type fixtureServer struct {
	releases    string
	lastRelBody map[string]any
}

func newTestClient(t *testing.T, fs *fixtureServer) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var file string
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/series/search":
			file = "search_orv.json"
		case r.Method == http.MethodGet && r.URL.Path == "/series/50369844984":
			file = "series_orv.json"
		case r.Method == http.MethodPost && r.URL.Path == "/releases/search":
			json.NewDecoder(r.Body).Decode(&fs.lastRelBody)
			file = fs.releases
		default:
			http.NotFound(w, r)
			return
		}
		b, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(0), BaseURL: srv.URL}
}

func TestSearchFindsORVByKoreanName(t *testing.T) {
	c := newTestClient(t, &fixtureServer{})
	results, total, err := c.Search(context.Background(), "jeonjijeok dokja sijeom", 1)
	if err != nil {
		t.Fatal(err)
	}
	if total < 1 || len(results) == 0 {
		t.Fatalf("total=%d len=%d", total, len(results))
	}
	r := results[0]
	if r.ID != 50369844984 || r.Type != "Manhwa" || r.Rating != 8.82 || r.HitTitle != "Jeonjijeok Dokja Sijeom" {
		t.Fatalf("first result = %+v", r)
	}
	if r.Title != "Omniscient Reader" || !strings.HasPrefix(r.CoverURL, "https://") || r.URL == "" {
		t.Fatalf("first result = %+v", r)
	}
}

func TestSeriesDetail(t *testing.T) {
	c := newTestClient(t, &fixtureServer{})
	d, err := c.Series(context.Background(), 50369844984)
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Omniscient Reader" || d.Type != "Manhwa" || d.Completed || d.Votes < 900 || d.Rating != 8.82 {
		t.Fatalf("detail = %+v", d)
	}
	if len(d.AltTitles) < 8 || !contains(d.AltTitles, "ORV") {
		t.Fatalf("alt titles = %v", d.AltTitles)
	}
	if d.Year != "2020" || !strings.Contains(d.Status, "Chapters") || d.CoverURL == "" {
		t.Fatalf("detail = %+v", d)
	}
}

func TestLatestReleasePicksHighest(t *testing.T) {
	fs := &fixtureServer{releases: "releases_orv.json"}
	c := newTestClient(t, fs)
	rel, ok, err := c.LatestRelease(context.Background(), 50369844984)
	if err != nil || !ok {
		t.Fatalf("LatestRelease: ok=%v err=%v", ok, err)
	}
	if rel.Chapter != 311 || rel.Raw != "311" || rel.Date != "2026-05-19" || !strings.Contains(rel.Group, "Flame Comics") {
		t.Fatalf("release = %+v", rel)
	}
	if fs.lastRelBody["orderby"] != "date" || fs.lastRelBody["asc"] != "desc" ||
		fs.lastRelBody["search"] != "50369844984" || fs.lastRelBody["search_type"] != "series" {
		t.Fatalf("request body = %v", fs.lastRelBody)
	}
}

func TestLatestReleaseNone(t *testing.T) {
	c := newTestClient(t, &fixtureServer{releases: "releases_empty.json"})
	_, ok, err := c.LatestRelease(context.Background(), 50369844984)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v; want false, nil", ok, err)
	}
}

func TestMapType(t *testing.T) {
	cases := map[string]store.SeriesType{
		"Manhwa": store.TypeManhwa, "Manga": store.TypeManga, "Manhua": store.TypeManhua,
		"OEL": store.TypeOther, "Novel": store.TypeOther, "Doujinshi": store.TypeOther, "": store.TypeOther,
	}
	for in, want := range cases {
		if got := MapType(in); got != want {
			t.Errorf("MapType(%q) = %q, want %q", in, got, want)
		}
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
