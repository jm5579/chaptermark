package exportimport

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jm5579/chaptermark/internal/store"
)

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

func track(t *testing.T, s *store.Store, muID int64, title string, chapter float64, updated time.Time) {
	t.Helper()
	ctx := context.Background()
	manual := 99.0
	sr := store.Series{MUID: muID, Title: title, Type: "manhwa", AltTitles: []string{title + " Alt"}, LatestManual: &manual,
		CoverFile: "1.jpg", DetailsRefreshedAt: updated}
	if err := s.UpsertSeries(ctx, &sr); err != nil {
		t.Fatal(err)
	}
	e := store.Entry{SeriesID: sr.ID, Status: "reading", MyChapter: chapter, MyRating: 8, Notes: "note", UpdatedAt: updated, AddedAt: updated}
	if err := s.SaveEntry(ctx, e); err != nil {
		t.Fatal(err)
	}
}

var t1 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
var t2 = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

func TestRoundTrip(t *testing.T) {
	src := newStore(t)
	track(t, src, 1, "Eleceed", 330, t1)
	track(t, src, 2, "Tower of God", 640, t1)
	var buf bytes.Buffer
	if err := Export(context.Background(), src, &buf, t2); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"app": "ChapterMark"`) {
		t.Fatalf("export header missing: %.200s", buf.String())
	}
	dst := newStore(t)
	track(t, dst, 3, "To be replaced", 1, t1)
	res, err := Import(context.Background(), dst, bytes.NewReader(buf.Bytes()), ModeReplace)
	if err != nil || res.Added != 2 {
		t.Fatalf("Import = %+v, %v", res, err)
	}
	items, _ := dst.ListItems(context.Background())
	if len(items) != 2 {
		t.Fatalf("%d items after replace", len(items))
	}
	for _, it := range items {
		if it.Entry.MyRating != 8 || it.Entry.Notes != "note" || it.Series.LatestManual == nil || len(it.Series.AltTitles) != 2 {
			t.Errorf("imported item lost data: %+v", it)
		}
		if !it.Series.DetailsRefreshedAt.IsZero() || it.Series.CoverFile != "" {
			t.Errorf("imported series should be marked for refresh: %+v", it.Series)
		}
	}
}

func TestTagsSurviveExportAndImport(t *testing.T) {
	ctx := context.Background()
	src := newStore(t)
	track(t, src, 1, "Eleceed", 330, t1)
	items, _ := src.ListItems(ctx)
	if err := src.SetSeriesTags(ctx, items[0].Series.ID, []string{"Favourites", "Binge later"}); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Export(ctx, src, &buf, t2); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"Favourites"`) {
		t.Fatal("tags missing from the export file")
	}
	dst := newStore(t)
	if _, err := Import(ctx, dst, &buf, ModeReplace); err != nil {
		t.Fatal(err)
	}
	got, _ := dst.ListItems(ctx)
	if len(got) != 1 || strings.Join(got[0].Entry.Tags, ",") != "Binge later,Favourites" {
		t.Fatalf("imported tags = %v", got[0].Entry.Tags)
	}
}

func TestMergeKeepsNewer(t *testing.T) {
	src := newStore(t)
	track(t, src, 1, "Eleceed", 340, t2) // newer than dst's
	track(t, src, 2, "Lookism", 10, t1)  // older than dst's
	track(t, src, 4, "New One", 5, t1)   // not in dst
	var buf bytes.Buffer
	Export(context.Background(), src, &buf, t2)

	dst := newStore(t)
	track(t, dst, 1, "Eleceed", 330, t1)
	track(t, dst, 2, "Lookism", 540, t2)
	res, err := Import(context.Background(), dst, &buf, ModeMerge)
	if err != nil {
		t.Fatal(err)
	}
	if res.Added != 1 || res.Updated != 1 || res.Skipped != 1 {
		t.Fatalf("result = %+v", res)
	}
	got := map[int64]float64{}
	items, _ := dst.ListItems(context.Background())
	for _, it := range items {
		got[it.Series.MUID] = it.Entry.MyChapter
	}
	if got[1] != 340 || got[2] != 540 || got[4] != 5 {
		t.Fatalf("chapters after merge = %v", got)
	}
}

func TestRejectsForeignFile(t *testing.T) {
	dst := newStore(t)
	for _, body := range []string{`{"app":"Other","version":1,"items":[]}`, `{"app":"ChapterMark","version":2,"items":[]}`, `not json`} {
		if _, err := Import(context.Background(), dst, strings.NewReader(body), ModeMerge); err == nil {
			t.Errorf("Import(%q) should fail", body)
		}
	}
}

func TestReplaceRejectedFileKeepsLibrary(t *testing.T) {
	dst := newStore(t)
	track(t, dst, 1, "Eleceed", 330, t1)
	Import(context.Background(), dst, strings.NewReader(`{"app":"Other"}`), ModeReplace)
	if items, _ := dst.ListItems(context.Background()); len(items) != 1 {
		t.Fatal("a rejected file must not wipe the library")
	}
}
