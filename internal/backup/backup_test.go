package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/jm5579/chaptermark/internal/store"
)

func sourceDB(t *testing.T) *sql.DB {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	sr := store.Series{MUID: 1, Title: "Eleceed", Type: "manhwa"}
	if err := s.UpsertSeries(ctx, &sr); err != nil {
		t.Fatal(err)
	}
	return s.DB()
}

var today = time.Date(2026, 10, 23, 15, 0, 0, 0, time.UTC)

func TestDailyCreatesOncePerDay(t *testing.T) {
	db := sourceDB(t)
	dir := t.TempDir()
	created, err := Daily(db, dir, 10, today)
	if err != nil || !created {
		t.Fatalf("first Daily = %v, %v", created, err)
	}
	created, err = Daily(db, dir, 10, today.Add(time.Hour))
	if err != nil || created {
		t.Fatalf("second Daily same day = %v, %v; want false", created, err)
	}

	backupPath := filepath.Join(dir, "library-2026-10-23.db")
	b, err := store.Open(backupPath)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	var n int
	if err := b.DB().QueryRow("SELECT COUNT(*) FROM series WHERE title = 'Eleceed'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("backup content: n=%d err=%v", n, err)
	}
}

func TestDailyPrunes(t *testing.T) {
	db := sourceDB(t)
	dir := t.TempDir()
	for i := 1; i <= 12; i++ {
		name := fmt.Sprintf("library-2026-09-%02d.db", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Daily(db, dir, 10, today); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	if len(names) != 10 {
		t.Fatalf("%d backups remain, want 10: %v", len(names), names)
	}
	if names[0] != "library-2026-09-04.db" || names[9] != "library-2026-10-23.db" {
		t.Fatalf("wrong backups kept: %v", names)
	}
}
