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
	if err := s.DB().QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != 4 {
		t.Fatalf("user_version = %d, %v; want 4", v, err)
	}
}

func TestMigrateToV4RefreshesEverythingOnce(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// A library at schema v3, checked and refreshed recently.
	for _, m := range migrations[:3] {
		if _, err := s.DB().Exec(m); err != nil {
			t.Fatal(err)
		}
	}
	s.DB().Exec("PRAGMA user_version = 3")
	s.DB().Exec(`INSERT INTO series (id, mu_id, title, type, details_refreshed_at)
		VALUES (1, 42, 'Eleceed', 'manhwa', '2026-10-20T10:00:00Z')`)
	s.DB().Exec(`INSERT INTO entry (series_id, status) VALUES (1, 'completed')`)
	s.SetMeta(ctx, "last_full_check_at", "2026-10-24T10:00:00Z")
	s.SetMeta(ctx, "check_interval_hours", "12")

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate v3 to v4: %v", err)
	}
	// The 18+ labels are new, so every series is due a details refresh and
	// a check runs at the next start.
	it, err := s.GetItem(ctx, 1)
	if err != nil || !it.Series.DetailsRefreshedAt.IsZero() || it.Series.Genres != nil {
		t.Fatalf("after upgrade: %+v, %v", it.Series, err)
	}
	if _, ok, _ := s.GetMeta(ctx, "last_full_check_at"); ok {
		t.Fatal("last check time should be cleared")
	}
	if v, ok, _ := s.GetMeta(ctx, "check_interval_hours"); !ok || v != "12" {
		t.Fatalf("other settings must survive: %q %v", v, ok)
	}
}

func TestContentLabelsRoundTrip(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := orv()
	sr.Genres, sr.ContentRating = []string{"Action", "Fantasy"}, "safe"
	sr = addSeries(t, s, sr, "reading")
	it, err := s.GetItem(ctx, sr.ID)
	if err != nil || !reflect.DeepEqual(it.Series.Genres, []string{"Action", "Fantasy"}) ||
		it.Series.ContentRating != "safe" || it.Series.ALAdult {
		t.Fatalf("GetItem: %+v, %v", it.Series, err)
	}

	sr.Genres, sr.ContentRating, sr.ALAdult = []string{"Adult", "Smut"}, "erotica", true
	if err := s.UpdateDetails(ctx, &sr); err != nil {
		t.Fatal(err)
	}
	items, err := s.ListItems(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("ListItems: %v, %v", items, err)
	}
	got := items[0].Series
	if !reflect.DeepEqual(got.Genres, []string{"Adult", "Smut"}) || got.ContentRating != "erotica" || !got.ALAdult {
		t.Fatalf("after UpdateDetails: genres %v, rating %q, AniList adult %v", got.Genres, got.ContentRating, got.ALAdult)
	}
	byMU, _, _ := s.SeriesByMUID(ctx, sr.MUID)
	if !reflect.DeepEqual(byMU.Genres, got.Genres) || byMU.ContentRating != "erotica" || !byMU.ALAdult {
		t.Fatalf("SeriesByMUID = %+v", byMU)
	}
}

func TestMigrateFromV1KeepsData(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// A library created by v1 of the schema.
	if _, err := s.DB().Exec(migrations[0]); err != nil {
		t.Fatal(err)
	}
	s.DB().Exec("PRAGMA user_version = 1")
	s.DB().Exec(`INSERT INTO series (id, mu_id, title, type) VALUES (1, 42, 'Eleceed', 'manhwa')`)
	s.DB().Exec(`INSERT INTO entry (series_id, status, my_chapter) VALUES (1, 'reading', 330)`)

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate v1 to v2: %v", err)
	}
	it, err := s.GetItem(ctx, 1)
	if err != nil || it.Series.Title != "Eleceed" || it.Entry.MyChapter != 330 || it.Series.RomanizedTitle != "" {
		t.Fatalf("after upgrade: %+v, %v", it, err)
	}
}

func TestUpdateDetailsWritesTitles(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := addSeries(t, s, orv(), "reading")
	sr.RomanizedTitle, sr.NativeTitle = "Jeonjijeok Dokja Sijeom", "전지적 독자 시점"
	if err := s.UpdateDetails(ctx, &sr); err != nil {
		t.Fatal(err)
	}
	got, _, _ := s.SeriesByMUID(ctx, sr.MUID)
	if got.RomanizedTitle != "Jeonjijeok Dokja Sijeom" || got.NativeTitle != "전지적 독자 시점" {
		t.Fatalf("titles = %q / %q", got.RomanizedTitle, got.NativeTitle)
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
	sr.RomanizedTitle, sr.NativeTitle = "Jeonjijeok Dokja Sijeom", "전지적 독자 시점"
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
