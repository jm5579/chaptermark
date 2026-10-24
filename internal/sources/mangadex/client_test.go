package mangadex

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jm5579/chaptermark/internal/httpx"
)

func TestMUIDToBase36(t *testing.T) {
	if got := MUIDToBase36(50369844984); got != "n50wl4o" {
		t.Fatalf("MUIDToBase36 = %q", got)
	}
}

func newTestClient(t *testing.T, gotQuery *string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga" {
			http.NotFound(w, r)
			return
		}
		*gotQuery = r.URL.RawQuery
		b, _ := os.ReadFile("testdata/search_orv.json")
		w.Write(b)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(0), BaseURL: srv.URL}
}

func TestFindByMUMatchesLink(t *testing.T) {
	var q string
	c := newTestClient(t, &q)
	links, ok, err := c.FindByMU(context.Background(), "Omniscient Reader", 50369844984)
	if err != nil || !ok {
		t.Fatalf("FindByMU: ok=%v err=%v", ok, err)
	}
	if links.MDID != "9a414441-bbad-43f1-a3a7-dc262ca790a3" || links.ALID != 119257 || links.MALID != 132214 {
		t.Fatalf("links = %+v", links)
	}
	want := "https://uploads.mangadex.org/covers/9a414441-bbad-43f1-a3a7-dc262ca790a3/be18dc9a-7f1c-4ca5-b318-ffff2d7d58c3.jpg.512.jpg"
	if links.CoverURL != want {
		t.Fatalf("CoverURL = %q", links.CoverURL)
	}
	if q == "" {
		t.Fatal("no query sent")
	}
}

func TestFindByMUNoMatch(t *testing.T) {
	var q string
	c := newTestClient(t, &q)
	_, ok, err := c.FindByMU(context.Background(), "Omniscient Reader", 1)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v; want false, nil", ok, err)
	}
}
