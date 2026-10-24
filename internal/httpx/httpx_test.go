package httpx

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fastClient(minInterval time.Duration) *Client {
	c := New(minInterval)
	c.Sleep = func(time.Duration) {}
	return c
}

func TestSendsUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.UserAgent()
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	var out struct{ OK bool }
	if err := fastClient(0).GetJSON(context.Background(), srv.URL, &out); err != nil || !out.OK {
		t.Fatalf("GetJSON: %v %+v", err, out)
	}
	if ua != UserAgent {
		t.Fatalf("User-Agent = %q", ua)
	}
}

func TestPostJSONSendsBody(t *testing.T) {
	var ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		var in map[string]string
		if err := decode(r.Body, &in); err != nil || in["search"] != "orv" {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"hits":3}`))
	}))
	defer srv.Close()
	var out struct{ Hits int }
	if err := fastClient(0).PostJSON(context.Background(), srv.URL, map[string]string{"search": "orv"}, &out); err != nil || out.Hits != 3 {
		t.Fatalf("PostJSON: %v %+v", err, out)
	}
	if ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestThrottleSpacing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{}`)) }))
	defer srv.Close()
	c := New(50 * time.Millisecond)
	start := time.Now()
	for i := 0; i < 3; i++ {
		if err := c.GetJSON(context.Background(), srv.URL, &struct{}{}); err != nil {
			t.Fatal(err)
		}
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond {
		t.Fatalf("3 calls took %v, want >= 100ms", elapsed)
	}
}

func TestRetriesOn429ThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if err := fastClient(0).GetJSON(context.Background(), srv.URL, &struct{}{}); err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d, want 2", calls.Load())
	}
}

func TestGivesUpAfter3Retries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	err := fastClient(0).GetJSON(context.Background(), srv.URL, &struct{}{})
	var se *StatusError
	if !errors.As(err, &se) || se.Code != 503 {
		t.Fatalf("err = %v, want StatusError 503", err)
	}
	if calls.Load() != 4 {
		t.Fatalf("calls = %d, want 4", calls.Load())
	}
}

func TestNoRetryOn404(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	err := fastClient(0).GetJSON(context.Background(), srv.URL, &struct{}{})
	var se *StatusError
	if !errors.As(err, &se) || se.Code != 404 || calls.Load() != 1 {
		t.Fatalf("err = %v calls = %d", err, calls.Load())
	}
}

func TestOfflineError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()
	err := fastClient(0).GetJSON(context.Background(), url, &struct{}{})
	if !errors.Is(err, ErrOffline) {
		t.Fatalf("err = %v, want ErrOffline", err)
	}
}

func TestRejectsOversizedResponses(t *testing.T) {
	big := `{"a":"` + strings.Repeat("x", 4096) + `"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(big))
	}))
	defer srv.Close()
	c := fastClient(0)
	c.MaxBody = 1024
	if err := c.GetJSON(context.Background(), srv.URL, &struct{ A string }{}); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("GetJSON err = %v, want ErrTooLarge", err)
	}
	if _, _, err := c.GetBytes(context.Background(), srv.URL); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("GetBytes err = %v, want ErrTooLarge", err)
	}
	c.MaxBody = 1 << 20
	if err := c.GetJSON(context.Background(), srv.URL, &struct{ A string }{}); err != nil {
		t.Fatalf("a response under the limit failed: %v", err)
	}
}

func TestGetBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte("PNGDATA"))
	}))
	defer srv.Close()
	b, ct, err := fastClient(0).GetBytes(context.Background(), srv.URL)
	if err != nil || string(b) != "PNGDATA" || ct != "image/png" {
		t.Fatalf("GetBytes = %q %q %v", b, ct, err)
	}
}
