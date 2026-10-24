package mangadex

import (
	"context"
	"encoding/json"
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
	if links.Romanized != "Jeonjijeok Dokja Sijeom" || links.Native != "전지적 독자 시점" {
		t.Fatalf("titles = %q / %q", links.Romanized, links.Native)
	}
}

func TestByID(t *testing.T) {
	var search struct {
		Data []json.RawMessage `json:"data"`
	}
	b, _ := os.ReadFile("testdata/search_orv.json")
	json.Unmarshal(b, &search)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manga/9a414441-bbad-43f1-a3a7-dc262ca790a3" {
			http.NotFound(w, r)
			return
		}
		w.Write(append(append([]byte(`{"result":"ok","data":`), search.Data[0]...), '}'))
	}))
	defer srv.Close()
	c := &Client{HTTP: httpx.New(0), BaseURL: srv.URL}
	links, ok, err := c.ByID(context.Background(), "9a414441-bbad-43f1-a3a7-dc262ca790a3")
	if err != nil || !ok {
		t.Fatalf("ByID: ok=%v err=%v", ok, err)
	}
	if links.ALID != 119257 || links.Romanized != "Jeonjijeok Dokja Sijeom" || links.Native != "전지적 독자 시점" {
		t.Fatalf("links = %+v", links)
	}
	if _, ok, err := c.ByID(context.Background(), "missing"); ok || err == nil {
		t.Fatalf("missing id: ok=%v err=%v", ok, err)
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
