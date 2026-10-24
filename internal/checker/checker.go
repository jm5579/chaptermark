// Package checker looks up the latest chapters of everything the owner is
// reading, in the background, one series at a time.
package checker

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jm5579/chaptermark/internal/httpx"
	"github.com/jm5579/chaptermark/internal/library"
	"github.com/jm5579/chaptermark/internal/store"
)

const (
	metaLastCheck         = "last_full_check_at"
	offlineFailuresToStop = 3
)

type Progress struct {
	Running       bool      `json:"running"`
	Done          int       `json:"done"`
	Total         int       `json:"total"`
	Failed        int       `json:"failed"`
	LastCheckedAt time.Time `json:"lastCheckedAt"`
	Error         string    `json:"error"` // "", "offline" or "partial"
}

type Checker struct {
	Lib      *library.Service
	Store    *store.Store
	Interval func() time.Duration
	Now      func() time.Time

	running atomic.Bool
	mu      sync.Mutex
	status  Progress
}

func (c *Checker) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func (c *Checker) lastCheck(ctx context.Context) time.Time {
	v, ok, err := c.Store.GetMeta(ctx, metaLastCheck)
	if err != nil || !ok {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, v)
	return t
}

// Due reports whether the last full check is older than the interval.
func (c *Checker) Due(ctx context.Context) (bool, error) {
	last := c.lastCheck(ctx)
	return last.IsZero() || c.now().Sub(last) >= c.Interval(), nil
}

// Watch calls run on every tick whose check is due (see Due), until ctx ends.
// It keeps chapters fresh while the app stays open for days.
func (c *Checker) Watch(ctx context.Context, ticks <-chan time.Time, run func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			if due, _ := c.Due(ctx); due {
				run()
			}
		}
	}
}

// Status returns the progress of the current or most recent check.
func (c *Checker) Status() Progress {
	c.mu.Lock()
	p := c.status
	c.mu.Unlock()
	if p.LastCheckedAt.IsZero() {
		p.LastCheckedAt = c.lastCheck(context.Background())
	}
	return p
}

func (c *Checker) set(p Progress, emit func(Progress)) {
	c.mu.Lock()
	c.status = p
	c.mu.Unlock()
	emit(p)
}

func checkable(st store.Status) bool {
	return st == store.StatusReading || st == store.StatusOnHold || st == store.StatusPlan
}

// Run checks every Reading / On hold / Plan to read series, highest owner
// rating first. It is a no-op if a check is already running.
func (c *Checker) Run(ctx context.Context, emit func(Progress), onItem func(library.ItemView)) error {
	if !c.running.CompareAndSwap(false, true) {
		return nil
	}
	defer c.running.Store(false)

	items, err := c.Store.ListItems(ctx)
	if err != nil {
		return err
	}
	var todo []store.Item
	for _, it := range items {
		if checkable(it.Entry.Status) {
			todo = append(todo, it)
		}
	}
	sort.SliceStable(todo, func(i, j int) bool {
		a, b := todo[i].Entry, todo[j].Entry
		if a.MyRating != b.MyRating {
			return a.MyRating > b.MyRating
		}
		return a.LastReadAt.After(b.LastReadAt)
	})

	p := Progress{Running: true, Total: len(todo), LastCheckedAt: c.lastCheck(ctx)}
	c.set(p, emit)
	succeeded, offlineStreak := 0, 0
	for _, it := range todo {
		if ctx.Err() != nil {
			break
		}
		err := c.Lib.RefreshLatest(ctx, it.Series.ID)
		if errors.Is(err, store.ErrNotFound) {
			// Removed by the owner while the check was running: nothing to do.
			p.Done++
			c.set(p, emit)
			continue
		}
		if err == nil && c.Lib.DetailsStale(it) {
			if derr := c.Lib.RefreshDetails(ctx, it.Series.ID); derr != nil {
				log.Printf("refresh details %q: %v", it.Series.Title, derr)
			}
		}
		p.Done++
		if err != nil {
			log.Printf("check %q: %v", it.Series.Title, err)
			p.Failed++
			if errors.Is(err, httpx.ErrOffline) {
				offlineStreak++
			} else {
				offlineStreak = 0
			}
			if succeeded == 0 && offlineStreak >= offlineFailuresToStop {
				p.Error = "offline"
				break
			}
		} else {
			succeeded++
			offlineStreak = 0
			if v, gerr := c.Lib.Get(ctx, it.Series.ID); gerr == nil {
				onItem(v)
			}
		}
		c.set(p, emit)
	}

	if p.Error == "" && p.Failed > 0 {
		p.Error = "partial"
	}
	if succeeded > 0 || p.Failed == 0 {
		now := c.now()
		if err := c.Store.SetMeta(ctx, metaLastCheck, now.Format(time.RFC3339)); err != nil {
			return err
		}
		p.LastCheckedAt = now
	}
	p.Running = false
	c.set(p, emit)
	return nil
}
