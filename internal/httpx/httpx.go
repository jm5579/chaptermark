// Package httpx is a small HTTP client that is polite to free APIs: it sends an
// identifying User-Agent, spaces out requests, and backs off on 429/5xx.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const UserAgent = "ChapterMark/1.0 (+https://github.com/jm5579/chaptermark)"

const maxRetries = 3

// ErrOffline wraps connection-level failures (no network, DNS, refused, timeout).
var ErrOffline = errors.New("offline")

type StatusError struct {
	Code int
	URL  string
}

func (e *StatusError) Error() string { return fmt.Sprintf("HTTP %d from %s", e.Code, e.URL) }

type Client struct {
	HTTP        *http.Client
	MinInterval time.Duration
	BaseBackoff time.Duration
	Sleep       func(time.Duration)

	mu   sync.Mutex
	next time.Time
}

// New returns a client that waits at least minInterval between requests.
func New(minInterval time.Duration) *Client {
	return &Client{
		HTTP:        &http.Client{Timeout: 15 * time.Second},
		MinInterval: minInterval,
		BaseBackoff: 2 * time.Second,
		Sleep:       time.Sleep,
	}
}

func (c *Client) throttle() {
	c.mu.Lock()
	now := time.Now()
	wait := c.next.Sub(now)
	if wait < 0 {
		wait = 0
	}
	c.next = now.Add(wait + c.MinInterval)
	c.mu.Unlock()
	if wait > 0 {
		c.Sleep(wait)
	}
}

func (c *Client) do(ctx context.Context, method, url string, body []byte, accept string) (*http.Response, error) {
	for attempt := 0; ; attempt++ {
		c.throttle()
		var rd io.Reader
		if body != nil {
			rd = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, rd)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", accept)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, fmt.Errorf("%w: %v", ErrOffline, err)
		}
		if resp.StatusCode < 300 {
			return resp, nil
		}
		resp.Body.Close()
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if !retryable || attempt >= maxRetries {
			return nil, &StatusError{Code: resp.StatusCode, URL: url}
		}
		c.Sleep(c.backoff(attempt, resp.Header.Get("Retry-After")))
	}
}

func (c *Client) backoff(attempt int, retryAfter string) time.Duration {
	if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return c.BaseBackoff << attempt
}

func decode(r io.Reader, out any) error {
	return json.NewDecoder(r).Decode(out)
}

func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	resp, err := c.do(ctx, http.MethodGet, url, nil, "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp.Body, out)
}

func (c *Client) PostJSON(ctx context.Context, url string, body, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, url, b, "application/json")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp.Body, out)
}

// GetBytes downloads url (capped at 10 MB) and returns its body and Content-Type.
func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, string, error) {
	resp, err := c.do(ctx, http.MethodGet, url, nil, "*/*")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	return b, resp.Header.Get("Content-Type"), err
}
