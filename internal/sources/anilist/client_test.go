package anilist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jm5579/chaptermark/internal/httpx"
)

func newTestClient(t *testing.T, body []byte, gotID *float64, gotQuery *string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if id, ok := req.Variables["id"].(float64); ok {
			*gotID = id
		}
		*gotQuery = req.Query
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(0), BaseURL: srv.URL}
}

func TestMedia(t *testing.T) {
	body, err := os.ReadFile("testdata/media_119257.json")
	if err != nil {
		t.Fatal(err)
	}
	var id float64
	var query string
	c := newTestClient(t, body, &id, &query)
	m, ok, err := c.Media(context.Background(), 119257)
	if err != nil || !ok || m != (Media{Score: 8.6, HasScore: true}) {
		t.Fatalf("Media = %+v, %v, %v; want score 8.6, not adult", m, ok, err)
	}
	if id != 119257 || !strings.Contains(query, "isAdult") {
		t.Fatalf("sent id %v, query %q", id, query)
	}
}

func TestMediaAdultWithoutScore(t *testing.T) {
	var id float64
	var query string
	c := newTestClient(t, []byte(`{"data":{"Media":{"averageScore":null,"isAdult":true}}}`), &id, &query)
	m, ok, err := c.Media(context.Background(), 5)
	if err != nil || !ok || m != (Media{Adult: true}) {
		t.Fatalf("Media = %+v, %v, %v; want no score, adult", m, ok, err)
	}
}

func TestMediaMissing(t *testing.T) {
	var id float64
	var query string
	c := newTestClient(t, []byte(`{"data":{"Media":null}}`), &id, &query)
	if _, ok, err := c.Media(context.Background(), 5); err != nil || ok {
		t.Fatalf("ok=%v err=%v; want false, nil", ok, err)
	}
}
