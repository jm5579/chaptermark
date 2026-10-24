package checker

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jm5579/chaptermark/internal/httpx"
	"github.com/jm5579/chaptermark/internal/library"
	"github.com/jm5579/chaptermark/internal/library/libtest"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
	"github.com/jm5579/chaptermark/internal/store"
)

var now = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

type fixture struct {
	chk *Checker
	lib *library.Service
	mu  *libtest.MU
	ids map[int64]int64 // MU id -> series id
}

// newFixture tracks one series per spec: MU id, status, rating.
func newFixture(t *testing.T, specs ...struct {
	muID   int64
	status store.Status
	rating int
}) *fixture {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	mu, md, al := libtest.ORV()
	clock := now
	lib := &library.Service{Store: st, MU: mu, MD: md, AL: al, FetchCover: libtest.Covers, Now: func() time.Time { return clock }}
	f := &fixture{lib: lib, mu: mu, ids: map[int64]int64{}}
	for _, s := range specs {
		mu.Details[s.muID] = mangaupdates.SeriesDetail{ID: s.muID, Title: fmt.Sprintf("Series %d", s.muID), Type: "Manhwa"}
		mu.Releases[s.muID] = mangaupdates.Release{Chapter: 10, Raw: "10"}
		v, err := lib.Add(ctx, s.muID, library.AddInput{Status: s.status, MyRating: s.rating, MyChapter: 5})
		if err != nil {
			t.Fatal(err)
		}
		f.ids[s.muID] = v.Series.ID
	}
	mu.ResetCalls()
	f.chk = &Checker{Lib: lib, Store: st, Interval: func() time.Duration { return 6 * time.Hour }, Now: func() time.Time { return clock }}
	return f
}

type spec = struct {
	muID   int64
	status store.Status
	rating int
}

func TestRunOrdersByRating(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusReading, 5}, spec{2, store.StatusOnHold, 9}, spec{3, store.StatusPlan, 7}, spec{4, store.StatusReading, 0})
	var last Progress
	var updated []int64
	err := f.chk.Run(context.Background(), func(p Progress) { last = p }, func(v library.ItemView) { updated = append(updated, v.Series.MUID) })
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{2, 3, 1, 4}
	if got := f.mu.CallsSnapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("check order = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(updated, want) {
		t.Fatalf("onItem order = %v", updated)
	}
	if last.Running || last.Done != 4 || last.Total != 4 || last.Error != "" || !last.LastCheckedAt.Equal(now) {
		t.Fatalf("final progress = %+v", last)
	}
	if v, ok, _ := f.lib.Store.GetMeta(context.Background(), "last_full_check_at"); !ok || v != "2026-10-23T12:00:00Z" {
		t.Fatalf("last_full_check_at = %q %v", v, ok)
	}
}

func TestRunSkipsCompletedAndDropped(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusCompleted, 9}, spec{2, store.StatusDropped, 9}, spec{3, store.StatusReading, 1})
	if err := f.chk.Run(context.Background(), func(Progress) {}, func(library.ItemView) {}); err != nil {
		t.Fatal(err)
	}
	if got := f.mu.CallsSnapshot(); !reflect.DeepEqual(got, []int64{3}) {
		t.Fatalf("checked %v, want [3]", got)
	}
}

func TestRunRefreshesNewChapters(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusReading, 5})
	f.mu.Releases[1] = mangaupdates.Release{Chapter: 25, Raw: "25", Group: "G"}
	var got library.ItemView
	f.chk.Run(context.Background(), func(Progress) {}, func(v library.ItemView) { got = v })
	if got.NewCount != 20 || got.Series.LatestGroup != "G" {
		t.Fatalf("updated view = new %d group %q", got.NewCount, got.Series.LatestGroup)
	}
}

func TestOfflineKeepsStoredLatest(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusReading, 5}, spec{2, store.StatusReading, 4}, spec{3, store.StatusReading, 3}, spec{4, store.StatusReading, 2})
	f.mu.Err = fmt.Errorf("%w: dial tcp: no such host", httpx.ErrOffline)
	var last Progress
	if err := f.chk.Run(context.Background(), func(p Progress) { last = p }, func(library.ItemView) {}); err != nil {
		t.Fatal(err)
	}
	if last.Error != "offline" || last.Running {
		t.Fatalf("progress = %+v", last)
	}
	if n := len(f.mu.CallsSnapshot()); n != 3 {
		t.Fatalf("made %d calls, want to stop after 3 offline failures", n)
	}
	v, _ := f.lib.Get(context.Background(), f.ids[1])
	if v.Series.LatestChapter != 10 {
		t.Fatalf("stored latest changed to %v", v.Series.LatestChapter)
	}
	if _, ok, _ := f.lib.Store.GetMeta(context.Background(), "last_full_check_at"); ok {
		t.Fatal("last_full_check_at must not be written when nothing succeeded")
	}
}

func TestSeriesRemovedDuringCheckIsNotAFailure(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusReading, 9}, spec{2, store.StatusReading, 5})
	// While series 1 is being checked, the owner removes series 2.
	f.mu.OnLatest = func(id int64) {
		if id == 1 {
			f.lib.Store.DeleteSeries(context.Background(), f.ids[2])
		}
	}
	var last Progress
	if err := f.chk.Run(context.Background(), func(p Progress) { last = p }, func(library.ItemView) {}); err != nil {
		t.Fatal(err)
	}
	if last.Error != "" || last.Failed != 0 || last.Done != last.Total {
		t.Fatalf("progress = %+v, want a clean finish", last)
	}
}

func TestDue(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if due, _ := f.chk.Due(ctx); !due {
		t.Fatal("never checked should be due")
	}
	f.lib.Store.SetMeta(ctx, "last_full_check_at", now.Add(-5*time.Hour).Format(time.RFC3339))
	if due, _ := f.chk.Due(ctx); due {
		t.Fatal("5h ago with 6h interval should not be due")
	}
	f.lib.Store.SetMeta(ctx, "last_full_check_at", now.Add(-7*time.Hour).Format(time.RFC3339))
	if due, _ := f.chk.Due(ctx); !due {
		t.Fatal("7h ago should be due")
	}
}

func TestRunNotReentrant(t *testing.T) {
	f := newFixture(t, spec{1, store.StatusReading, 5})
	f.mu.Block = make(chan struct{})
	done := make(chan struct{})
	go func() {
		f.chk.Run(context.Background(), func(Progress) {}, func(library.ItemView) {})
		close(done)
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !f.chk.Status().Running {
		if time.Now().After(deadline) {
			t.Fatal("first run never started")
		}
		time.Sleep(5 * time.Millisecond)
	}
	emitted := false
	if err := f.chk.Run(context.Background(), func(Progress) { emitted = true }, func(library.ItemView) {}); err != nil {
		t.Fatal(err)
	}
	if emitted {
		t.Fatal("second Run should be a no-op while the first is running")
	}
	close(f.mu.Block)
	<-done
	if n := len(f.mu.CallsSnapshot()); n != 1 {
		t.Fatalf("%d calls, want 1", n)
	}
}
