package store

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return s
}

func orv() Series {
	return Series{
		MUID:      50369844984,
		Title:     "Omniscient Reader",
		Type:      "manhwa",
		AltTitles: []string{"Jeonjijeok Dokja Sijeom", "ORV", "전지적 독자 시점", "Omniscient Reader's Viewpoint"},
	}
}

func addSeries(t *testing.T, s *Store, sr Series, status Status) Series {
	t.Helper()
	ctx := context.Background()
	if err := s.UpsertSeries(ctx, &sr); err != nil {
		t.Fatalf("UpsertSeries: %v", err)
	}
	if err := s.SaveEntry(ctx, Entry{SeriesID: sr.ID, Status: status}); err != nil {
		t.Fatalf("SaveEntry: %v", err)
	}
	return sr
}

func TestOpenPathWithURICharacters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Manga #1 (100%25 done) & more")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "library.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	s.Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database not created at the exact path: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Dir(dir))
	for _, e := range entries {
		if e.Name() != filepath.Base(dir) {
			t.Errorf("stray file created next to the folder: %q", e.Name())
		}
	}
}

func TestMigrateIdempotent(t *testing.T) {
	s := newStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var v int
	if err := s.DB().QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 1 {
		t.Fatalf("user_version = %d, %v; want 1", v, err)
	}
}

func TestUpsertSeriesByMUID(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	first := orv()
	if err := s.UpsertSeries(ctx, &first); err != nil {
		t.Fatal(err)
	}
	second := orv()
	second.AltTitles = []string{"ORV"}
	second.MURating = 8.82
	if err := s.UpsertSeries(ctx, &second); err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("ID changed: %d -> %d", first.ID, second.ID)
	}
	got, ok, err := s.SeriesByMUID(ctx, 50369844984)
	if err != nil || !ok {
		t.Fatalf("SeriesByMUID: %v %v", ok, err)
	}
	if got.MURating != 8.82 {
		t.Errorf("MURating = %v", got.MURating)
	}
	want := []string{"Omniscient Reader", "ORV"}
	if !reflect.DeepEqual(got.AltTitles, want) {
		t.Errorf("AltTitles = %v, want %v", got.AltTitles, want)
	}
}

func TestSaveAndListItems(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	manual := 320.0
	sr := orv()
	sr.MDID, sr.ALID, sr.MALID = "9a414441", 119257, 132214
	sr.Year, sr.PubStatus, sr.Completed = "2020", "311 Chapters (Hiatus)", false
	sr.CoverFile, sr.CoverURL = "1.jpg", "https://example/cover.jpg"
	sr.MURating, sr.MUVotes, sr.ALRating = 8.82, 949, 8.6
	sr.LatestChapter, sr.LatestRaw, sr.LatestGroup, sr.LatestDate = 311, "311", "Flame Comics", "2026-05-19"
	sr.LatestManual = &manual
	sr.TypeOverride = "manga"
	sr.DetailsRefreshedAt = time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	if err := s.UpsertSeries(ctx, &sr); err != nil {
		t.Fatal(err)
	}
	e := Entry{
		SeriesID: sr.ID, Status: "reading", MyChapter: 212.5, MyRating: 10,
		Notes: "great", ReadURL: "https://read.example/orv",
		LastReadAt: time.Date(2026, 7, 1, 12, 30, 0, 0, time.UTC),
		AddedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 7, 1, 12, 30, 0, 0, time.UTC),
	}
	if err := s.SaveEntry(ctx, e); err != nil {
		t.Fatal(err)
	}
	other := addSeries(t, s, Series{MUID: 2, Title: "Eleceed", Type: "manhwa"}, "plan")

	items, err := s.ListItems(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d", len(items))
	}
	var got Item
	for _, it := range items {
		if it.Series.ID == sr.ID {
			got = it
		}
	}
	if got.Series.LatestManual == nil || *got.Series.LatestManual != 320 {
		t.Errorf("LatestManual = %v", got.Series.LatestManual)
	}
	got.Series.LatestManual, sr.LatestManual = nil, nil
	if sr.AltTitles[0] != "Omniscient Reader" || len(sr.AltTitles) != 5 {
		t.Errorf("UpsertSeries should put the title first: %v", sr.AltTitles)
	}
	if !reflect.DeepEqual(got.Series, sr) {
		t.Errorf("series round trip:\n got %+v\nwant %+v", got.Series, sr)
	}
	if !reflect.DeepEqual(got.Entry, e) {
		t.Errorf("entry round trip:\n got %+v\nwant %+v", got.Entry, e)
	}
	plain, err := s.GetItem(ctx, other.ID)
	if err != nil || plain.Series.LatestManual != nil || !plain.Entry.LastReadAt.IsZero() {
		t.Errorf("GetItem(other) = %+v, %v", plain, err)
	}
}

func TestDeleteCascades(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := addSeries(t, s, orv(), "reading")
	if err := s.DeleteSeries(ctx, sr.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	s.DB().QueryRow("SELECT (SELECT COUNT(*) FROM entry) + (SELECT COUNT(*) FROM alt_titles)").Scan(&n)
	if n != 0 {
		t.Fatalf("%d orphan rows left", n)
	}
}

func TestSearchLocalAltNames(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := addSeries(t, s, orv(), "reading")
	addSeries(t, s, Series{MUID: 2, Title: "Eleceed", Type: "manhwa"}, "reading")
	for _, q := range []string{"jeonjijeok", "orv", "전지적", "omniscient reader", "Omniscient Readers Viewpoint", "  OMNISCIENT  "} {
		items, err := s.SearchLocal(ctx, q, 20)
		if err != nil {
			t.Fatalf("SearchLocal(%q): %v", q, err)
		}
		if len(items) != 1 || items[0].Series.ID != sr.ID {
			t.Errorf("SearchLocal(%q) = %d items", q, len(items))
		}
	}
}

func TestSearchLocalLiteralWildcards(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	addSeries(t, s, orv(), "reading")
	pct := addSeries(t, s, Series{MUID: 3, Title: "100% Perfect Girl", Type: "manhwa"}, "reading")
	for _, q := range []string{"%", "_", "%%", "\\"} {
		items, err := s.SearchLocal(ctx, q, 20)
		if err != nil || len(items) != 0 {
			t.Errorf("SearchLocal(%q) = %d items, %v; want none", q, len(items), err)
		}
	}
	items, err := s.SearchLocal(ctx, "100%", 20)
	if err != nil || len(items) != 1 || items[0].Series.ID != pct.ID {
		t.Errorf("SearchLocal(100%%) = %v, %v", items, err)
	}
}

func TestSetLatestClearsManual(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := addSeries(t, s, orv(), "reading")
	manual := 320.0
	if err := s.SetLatestManual(ctx, sr.ID, &manual); err != nil {
		t.Fatal(err)
	}
	checked := time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC)
	if err := s.SetLatest(ctx, sr.ID, Latest{Chapter: 311, Raw: "311", Group: "Asura", Date: "2026-05-19"}, checked); err != nil {
		t.Fatal(err)
	}
	it, _ := s.GetItem(ctx, sr.ID)
	if it.Series.LatestManual == nil || *it.Series.LatestManual != 320 || it.Series.LatestChapter != 311 {
		t.Fatalf("manual should survive a lower fetch: %+v", it.Series)
	}
	if !it.Series.ChaptersCheckedAt.Equal(checked) {
		t.Errorf("ChaptersCheckedAt = %v", it.Series.ChaptersCheckedAt)
	}
	if err := s.SetLatest(ctx, sr.ID, Latest{Chapter: 321, Raw: "321"}, checked); err != nil {
		t.Fatal(err)
	}
	it, _ = s.GetItem(ctx, sr.ID)
	if it.Series.LatestManual != nil {
		t.Fatalf("manual should clear once fetched exceeds it: %v", *it.Series.LatestManual)
	}
}

func TestMeta(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	if _, ok, err := s.GetMeta(ctx, "k"); ok || err != nil {
		t.Fatalf("missing key: ok=%v err=%v", ok, err)
	}
	s.SetMeta(ctx, "k", "v1")
	s.SetMeta(ctx, "k", "v2")
	if v, ok, _ := s.GetMeta(ctx, "k"); !ok || v != "v2" {
		t.Fatalf("GetMeta = %q, %v", v, ok)
	}
}
