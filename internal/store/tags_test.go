package store

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func tagNames(tags []Tag) map[string]int {
	out := map[string]int{}
	for _, t := range tags {
		out[t.Name] = t.Count
	}
	return out
}

func TestSetLatestStoresReleaseRhythm(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	sr := addSeries(t, s, orv(), "reading")
	l := Latest{Chapter: 311, Raw: "311", Group: "Asura", Date: "2026-05-19", IntervalDays: 7, LastReleaseDate: "2026-06-02"}
	if err := s.SetLatest(ctx, sr.ID, l, time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	it, _ := s.GetItem(ctx, sr.ID)
	if it.Series.ReleaseIntervalDays != 7 || it.Series.LastReleaseDate != "2026-06-02" {
		t.Fatalf("rhythm = %v days, last %q", it.Series.ReleaseIntervalDays, it.Series.LastReleaseDate)
	}
}

func TestSeriesTags(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	a := addSeries(t, s, orv(), "reading")
	b := addSeries(t, s, Series{MUID: 2, Title: "Eleceed", Type: "manhwa"}, "reading")

	// Trimmed, empty names dropped, duplicates (any case) merged, sorted by name.
	if err := s.SetSeriesTags(ctx, a.ID, []string{"Favourites", " binge later ", "favourites", ""}); err != nil {
		t.Fatal(err)
	}
	it, _ := s.GetItem(ctx, a.ID)
	if want := []string{"binge later", "Favourites"}; !reflect.DeepEqual(it.Entry.Tags, want) {
		t.Fatalf("tags = %v, want %v", it.Entry.Tags, want)
	}

	// The same tag in another case reuses the existing one.
	if err := s.SetSeriesTags(ctx, b.ID, []string{"FAVOURITES"}); err != nil {
		t.Fatal(err)
	}
	tags, err := s.ListTags(ctx)
	if err != nil || !reflect.DeepEqual(tagNames(tags), map[string]int{"binge later": 1, "Favourites": 2}) {
		t.Fatalf("ListTags = %v, %v", tags, err)
	}
	items, _ := s.ListItems(ctx)
	for _, it := range items {
		if it.Series.ID == b.ID && !reflect.DeepEqual(it.Entry.Tags, []string{"Favourites"}) {
			t.Fatalf("ListItems tags for b = %v", it.Entry.Tags)
		}
	}

	// A tag no series uses any more disappears.
	if err := s.SetSeriesTags(ctx, a.ID, []string{"Favourites"}); err != nil {
		t.Fatal(err)
	}
	tags, _ = s.ListTags(ctx)
	if !reflect.DeepEqual(tagNames(tags), map[string]int{"Favourites": 2}) {
		t.Fatalf("after removing 'binge later': %v", tags)
	}
}

func TestRenameAndDeleteTags(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	a := addSeries(t, s, orv(), "reading")
	b := addSeries(t, s, Series{MUID: 2, Title: "Eleceed", Type: "manhwa"}, "reading")
	s.SetSeriesTags(ctx, a.ID, []string{"Favourites", "Binge later"})
	s.SetSeriesTags(ctx, b.ID, []string{"Binge later"})
	id := func(name string) int64 {
		tags, _ := s.ListTags(ctx)
		for _, t := range tags {
			if t.Name == name {
				return t.ID
			}
		}
		t.Fatalf("tag %q not found in %v", name, tags)
		return 0
	}

	if err := s.RenameTag(ctx, id("Favourites"), "Faves"); err != nil {
		t.Fatal(err)
	}
	// Renaming onto an existing name (any case) merges the two tags.
	if err := s.RenameTag(ctx, id("Binge later"), "faves"); err != nil {
		t.Fatal(err)
	}
	tags, _ := s.ListTags(ctx)
	if !reflect.DeepEqual(tagNames(tags), map[string]int{"Faves": 2}) {
		t.Fatalf("after rename/merge: %v", tags)
	}
	if err := s.RenameTag(ctx, id("Faves"), "   "); err == nil {
		t.Fatal("an empty name should be refused")
	}

	if err := s.DeleteTag(ctx, id("Faves")); err != nil {
		t.Fatal(err)
	}
	it, _ := s.GetItem(ctx, a.ID)
	if len(it.Entry.Tags) != 0 {
		t.Fatalf("tags after delete = %v", it.Entry.Tags)
	}
}

func TestDeletingASeriesDropsItsOrphanTags(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	a := addSeries(t, s, orv(), "reading")
	s.SetSeriesTags(ctx, a.ID, []string{"Only here"})
	if err := s.DeleteSeries(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if tags, _ := s.ListTags(ctx); len(tags) != 0 {
		t.Fatalf("orphan tags left: %v", tags)
	}
}
