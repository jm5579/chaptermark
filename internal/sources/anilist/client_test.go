package anilist

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jm5579/chaptermark/internal/httpx"
)

func newTestClient(t *testing.T, body []byte, gotID *float64) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if id, ok := req.Variables["id"].(float64); ok {
			*gotID = id
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: httpx.New(0), BaseURL: srv.URL}
}

func TestScore(t *testing.T) {
	body, err := os.ReadFile("testdata/media_119257.json")
	if err != nil {
		t.Fatal(err)
	}
	var id float64
	c := newTestClient(t, body, &id)
	score, ok, err := c.Score(context.Background(), 119257)
	if err != nil || !ok || score != 8.6 {
		t.Fatalf("Score = %v, %v, %v; want 8.6", score, ok, err)
	}
	if id != 119257 {
		t.Fatalf("sent id %v", id)
	}
}

func TestScoreNull(t *testing.T) {
	var id float64
	c := newTestClient(t, []byte(`{"data":{"Media":{"averageScore":null}}}`), &id)
	_, ok, err := c.Score(context.Background(), 5)
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v; want false, nil", ok, err)
	}
}
