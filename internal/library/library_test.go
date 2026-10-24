package library

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jm5579/chaptermark/internal/library/libtest"
	"github.com/jm5579/chaptermark/internal/sources/mangadex"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
	"github.com/jm5579/chaptermark/internal/store"
)

var now = time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)

func newService(t *testing.T) (*Service, *libtest.MU, *libtest.MD, *libtest.AL) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu, md, al := libtest.ORV()
	clock := now
	svc := &Service{Store: st, MU: mu, MD: md, AL: al, FetchCover: libtest.Covers, Now: func() time.Time { return clock }}
	return svc, mu, md, al
}

func TestAddFetchesEverything(t *testing.T) {
	svc, _, _, _ := newService(t)
	v, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 212, MyRating: 10})
	if err != nil {
		t.Fatal(err)
	}
	s := v.Series
	if s.Title != "Omniscient Reader" || s.Type != store.TypeManhwa || s.ALID != 119257 || s.ALRating != 8.6 || s.MDID != "9a414441" {
		t.Fatalf("series = %+v", s)
	}
	if s.CoverFile == "" || s.LatestChapter != 311 || s.LatestGroup != "Flame Comics" || len(s.AltTitles) != 4 {
		t.Fatalf("series = %+v", s)
	}
	if v.EffectiveType != store.TypeManhwa || v.NewCount != 99 || v.EffectiveLatest != 311 {
		t.Fatalf("view = type %s new %d latest %v", v.EffectiveType, v.NewCount, v.EffectiveLatest)
	}
	if !v.Entry.AddedAt.Equal(now) || !v.Entry.LastReadAt.Equal(now) || v.Entry.MyRating != 10 {
		t.Fatalf("entry = %+v", v.Entry)
	}
	if !s.DetailsRefreshedAt.Equal(now) {
		t.Fatalf("DetailsRefreshedAt = %v", s.DetailsRefreshedAt)
	}
}

func TestAddWithoutChapterLeavesLastReadEmpty(t *testing.T) {
	svc, _, _, _ := newService(t)
	v, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusPlan})
	if err != nil || !v.Entry.LastReadAt.IsZero() {
		t.Fatalf("LastReadAt = %v, err %v", v.Entry.LastReadAt, err)
	}
}

func TestAddTwiceNoDuplicate(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	first, err := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 50})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusDropped, MyChapter: 1})
	if err != nil {
		t.Fatal(err)
	}
	items, _ := svc.List(ctx)
	if len(items) != 1 || second.Series.ID != first.Series.ID || second.Entry.MyChapter != 50 || second.Entry.Status != store.StatusReading {
		t.Fatalf("duplicate add changed things: %d items, %+v", len(items), second.Entry)
	}
}

func TestAddOtherTypeLandsInVisibleTab(t *testing.T) {
	svc, mu, _, _ := newService(t)
	mu.Details[7] = mangaupdates.SeriesDetail{ID: 7, Title: "Some OEL Comic", Type: "OEL"}
	mu.Details[8] = mangaupdates.SeriesDetail{ID: 8, Title: "Another OEL Comic", Type: "OEL"}
	ctx := context.Background()
	v, err := svc.Add(ctx, 7, AddInput{Status: store.StatusReading})
	if err != nil || v.EffectiveType != store.TypeManga || v.Series.TypeOverride != store.TypeManga {
		t.Fatalf("default for other = %q (override %q), %v", v.EffectiveType, v.Series.TypeOverride, err)
	}
	v, err = svc.Add(ctx, 8, AddInput{Status: store.StatusReading, TypeOverride: store.TypeManhwa})
	if err != nil || v.EffectiveType != store.TypeManhwa {
		t.Fatalf("override = %q, %v", v.EffectiveType, err)
	}
	// Clearing the override on an "other" series must not hide it.
	v, err = svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading})
	if err != nil || v.EffectiveType != store.TypeManga {
		t.Fatalf("cleared override = %q, %v", v.EffectiveType, err)
	}
}

func TestAddSurvivesMDandALFailure(t *testing.T) {
	svc, _, md, al := newService(t)
	md.Err = errors.New("mangadex down")
	al.Err = errors.New("anilist down")
	v, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusReading})
	if err != nil {
		t.Fatalf("Add should survive MD/AL failures: %v", err)
	}
	if v.Series.ALRating != 0 || v.Series.MDID != "" || v.Series.LatestChapter != 311 {
		t.Fatalf("series = %+v", v.Series)
	}
}

func TestAddFailsWhenMangaUpdatesFails(t *testing.T) {
	svc, mu, _, _ := newService(t)
	mu.Err = errors.New("offline")
	if _, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusReading}); err == nil {
		t.Fatal("expected error")
	}
	items, _ := svc.List(context.Background())
	if len(items) != 0 {
		t.Fatalf("%d items stored after failed add", len(items))
	}
}

func TestAddRejectsInvalidInput(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	for _, in := range []AddInput{
		{Status: "nonsense"},
		{Status: store.StatusReading, MyRating: 11},
		{Status: store.StatusReading, MyChapter: -1},
		{Status: store.StatusReading, TypeOverride: "novel"},
	} {
		if _, err := svc.Add(ctx, libtest.ORVID, in); err == nil {
			t.Errorf("Add(%+v) should fail", in)
		}
	}
}

func TestUpdateEntryTouchesLastReadOnlyOnChapterChange(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 100})
	later := now.Add(48 * time.Hour)
	svc.Now = func() time.Time { return later }

	v, err := svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading, MyChapter: 100, MyRating: 9, Notes: "n"})
	if err != nil || !v.Entry.LastReadAt.Equal(now) || !v.Entry.UpdatedAt.Equal(later) || v.Entry.MyRating != 9 {
		t.Fatalf("rating-only update: %+v %v", v.Entry, err)
	}
	v, err = svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading, MyChapter: 101, MyRating: 9, Notes: "n"})
	if err != nil || !v.Entry.LastReadAt.Equal(later) || v.Entry.MyChapter != 101 {
		t.Fatalf("chapter update: %+v %v", v.Entry, err)
	}
}

func TestNewCountAndEffectiveLatest(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 300})
	manual := 320.0
	v, err := svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading, MyChapter: 300, LatestManual: &manual})
	if err != nil || v.EffectiveLatest != 320 || v.NewCount != 20 {
		t.Fatalf("EffectiveLatest=%v NewCount=%d err=%v", v.EffectiveLatest, v.NewCount, err)
	}
	v, _ = svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading, MyChapter: 300})
	if v.Series.LatestManual != nil || v.EffectiveLatest != 311 {
		t.Fatalf("clearing manual: %v %v", v.Series.LatestManual, v.EffectiveLatest)
	}
}

func TestSearchOnlineFlagsInLibraryAndHidesOther(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	res, err := svc.SearchOnline(ctx, "orv", 1, false)
	if err != nil || len(res) != 1 || res[0].InLibrary || res[0].MappedType != store.TypeManhwa {
		t.Fatalf("before add: %+v %v", res, err)
	}
	added, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading})
	res, _ = svc.SearchOnline(ctx, "orv", 1, false)
	if !res[0].InLibrary || res[0].SeriesID != added.Series.ID {
		t.Fatalf("after add: %+v", res[0])
	}
	res, _ = svc.SearchOnline(ctx, "orv", 1, true)
	if len(res) != 2 || res[1].MappedType != store.TypeOther {
		t.Fatalf("includeOther: %+v", res)
	}
}

func TestSearchLocalAndRemove(t *testing.T) {
	svc, _, _, _ := newService(t)
	ctx := context.Background()
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading})
	found, err := svc.SearchLocal(ctx, "전지적")
	if err != nil || len(found) != 1 || found[0].NewCount != 311 {
		t.Fatalf("SearchLocal = %+v %v", found, err)
	}
	if err := svc.Remove(ctx, v.Series.ID); err != nil {
		t.Fatal(err)
	}
	if items, _ := svc.List(ctx); len(items) != 0 {
		t.Fatalf("%d items after remove", len(items))
	}
}

func TestAddStoresLanguageTitles(t *testing.T) {
	svc, _, _, _ := newService(t)
	v, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusReading})
	if err != nil || v.Series.RomanizedTitle != "Jeonjijeok Dokja Sijeom" || v.Series.NativeTitle != "전지적 독자 시점" {
		t.Fatalf("titles = %q / %q, %v", v.Series.RomanizedTitle, v.Series.NativeTitle, err)
	}
}

func TestRefreshDetailsFillsTitlesForAlreadyLinkedSeries(t *testing.T) {
	svc, _, md, _ := newService(t)
	ctx := context.Background()
	// Linked to MangaDex before titles were stored (an older library).
	full := md.Links[libtest.ORVID]
	md.Links[libtest.ORVID] = mangadex.Links{MDID: full.MDID, ALID: full.ALID, MALID: full.MALID}
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading})
	if v.Series.MDID == "" || v.Series.RomanizedTitle != "" {
		t.Fatalf("setup: %+v", v.Series)
	}
	md.Links[libtest.ORVID] = full
	if err := svc.RefreshDetails(ctx, v.Series.ID); err != nil {
		t.Fatal(err)
	}
	v, _ = svc.Get(ctx, v.Series.ID)
	if v.Series.RomanizedTitle != "Jeonjijeok Dokja Sijeom" || v.Series.NativeTitle != "전지적 독자 시점" {
		t.Fatalf("titles after refresh = %q / %q", v.Series.RomanizedTitle, v.Series.NativeTitle)
	}
}

func TestRefreshDetailsKeepsEditsMadeDuringFetch(t *testing.T) {
	svc, mu, _, _ := newService(t)
	ctx := context.Background()
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 300})
	manual := 400.0
	// The owner edits the series while MangaUpdates is being queried.
	mu.OnSeries = func() {
		mu.OnSeries = nil
		if _, err := svc.UpdateEntry(ctx, v.Series.ID, EntryUpdate{Status: store.StatusReading, MyChapter: 300,
			TypeOverride: store.TypeManga, LatestManual: &manual}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.RefreshDetails(ctx, v.Series.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ctx, v.Series.ID)
	if got.Series.TypeOverride != store.TypeManga || got.Series.LatestManual == nil || *got.Series.LatestManual != 400 {
		t.Fatalf("refresh overwrote the owner's edit: override %q manual %v", got.Series.TypeOverride, got.Series.LatestManual)
	}
	if got.Series.LatestChapter != 311 || len(got.Series.AltTitles) != 4 {
		t.Fatalf("refresh lost data: %+v", got.Series)
	}
}

func TestRefreshLatestAndStale(t *testing.T) {
	svc, mu, _, _ := newService(t)
	ctx := context.Background()
	v, _ := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusReading, MyChapter: 311})
	if svc.DetailsStale(v.Item()) {
		t.Fatal("fresh series reported stale")
	}
	mu.Releases[libtest.ORVID] = mangaupdates.Release{Chapter: 312, Raw: "312", Group: "Asura", Date: "2026-06-01"}
	svc.Now = func() time.Time { return now.Add(8 * 24 * time.Hour) }
	if err := svc.RefreshLatest(ctx, v.Series.ID); err != nil {
		t.Fatal(err)
	}
	v, _ = svc.Get(ctx, v.Series.ID)
	if v.NewCount != 1 || v.Series.LatestGroup != "Asura" {
		t.Fatalf("after refresh: new %d group %q", v.NewCount, v.Series.LatestGroup)
	}
	if !svc.DetailsStale(v.Item()) {
		t.Fatal("8-day-old details should be stale")
	}
}
