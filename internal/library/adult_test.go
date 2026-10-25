package library

import (
	"context"
	"reflect"
	"testing"

	"github.com/jm5579/chaptermark/internal/library/libtest"
	"github.com/jm5579/chaptermark/internal/sources/mangadex"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
	"github.com/jm5579/chaptermark/internal/store"
)

func TestIsAdult(t *testing.T) {
	cases := []struct {
		name string
		s    store.Series
		want bool
	}{
		{"ordinary", store.Series{Genres: []string{"Action", "Fantasy"}, ContentRating: "safe"}, false},
		{"no labels yet", store.Series{}, false},
		{"hentai", store.Series{Genres: []string{"Hentai"}}, true},
		{"smut", store.Series{Genres: []string{"Drama", "Smut"}}, true},
		{"adult", store.Series{Genres: []string{"Adult", "Romance"}}, true},
		{"lolicon", store.Series{Genres: []string{"Lolicon"}}, true},
		{"shotacon", store.Series{Genres: []string{"Shotacon"}}, true},
		{"any case", store.Series{Genres: []string{"smut"}}, true},
		{"ecchi stays", store.Series{Genres: []string{"Ecchi", "Harem"}}, false},
		{"mature stays", store.Series{Genres: []string{"Mature", "Horror"}}, false},
		{"yaoi alone stays", store.Series{Genres: []string{"Yaoi", "Romance"}}, false},
		{"MangaDex erotica", store.Series{ContentRating: "erotica"}, true},
		{"MangaDex pornographic", store.Series{ContentRating: "pornographic"}, true},
		{"MangaDex suggestive stays", store.Series{ContentRating: "suggestive"}, false},
		{"AniList adult", store.Series{ALAdult: true}, true},
	}
	for _, c := range cases {
		if got := IsAdult(c.s); got != c.want {
			t.Errorf("%s: IsAdult = %v, want %v", c.name, got, c.want)
		}
	}
}

func adultResults(mu *libtest.MU) {
	mu.Results = []mangaupdates.SearchResult{
		{ID: libtest.ORVID, Title: "Omniscient Reader", Type: "Manhwa", Genres: []string{"Action", "Fantasy"}},
		{ID: 2, Title: "Secret Class", Type: "Manhwa", Genres: []string{"Adult", "Drama", "Smut"}},
		{ID: 3, Title: "A Fan-service Comedy", Type: "Manga", Genres: []string{"Comedy", "Ecchi"}},
	}
}

func resultIDs(res []OnlineResult) []int64 {
	ids := []int64{}
	for _, r := range res {
		ids = append(ids, r.ID)
	}
	return ids
}

func TestSearchOnlineHidesAdultUnlessAllowed(t *testing.T) {
	svc, mu, _, _ := newService(t)
	ctx := context.Background()
	adultResults(mu)

	res, err := svc.SearchOnline(ctx, "secret", 1, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := resultIDs(res); !reflect.DeepEqual(got, []int64{libtest.ORVID, 3}) {
		t.Fatalf("results = %v, want ORV and the ecchi comedy only", got)
	}
	if !reflect.DeepEqual(mu.LastExclude, AdultGenres) {
		t.Fatalf("asked MangaUpdates to exclude %v, want %v", mu.LastExclude, AdultGenres)
	}

	res, _ = svc.SearchOnline(ctx, "secret", 1, false, true)
	if got := resultIDs(res); !reflect.DeepEqual(got, []int64{libtest.ORVID, 2, 3}) {
		t.Fatalf("with 18+ allowed, results = %v", got)
	}
	if mu.LastExclude != nil {
		t.Fatalf("with 18+ allowed, excluded %v", mu.LastExclude)
	}
}

func TestAddStoresAdultLabels(t *testing.T) {
	svc, mu, md, al := newService(t)
	d := mu.Details[libtest.ORVID]
	d.Genres = []string{"Action", "Adult"}
	mu.Details[libtest.ORVID] = d
	l := md.Links[libtest.ORVID]
	l.ContentRating = "erotica"
	md.Links[libtest.ORVID] = l
	al.Adult = map[int]bool{119257: true}

	v, err := svc.Add(context.Background(), libtest.ORVID, AddInput{Status: store.StatusReading})
	if err != nil {
		t.Fatal(err)
	}
	s := v.Series
	if !reflect.DeepEqual(s.Genres, []string{"Action", "Adult"}) || s.ContentRating != "erotica" || !s.ALAdult || !v.Adult {
		t.Fatalf("genres %v, rating %q, AniList adult %v, view adult %v", s.Genres, s.ContentRating, s.ALAdult, v.Adult)
	}
	if s.ALRating != 8.6 {
		t.Fatalf("AniList score lost: %v", s.ALRating)
	}
}

func TestRefreshDetailsFillsInAdultLabels(t *testing.T) {
	svc, mu, md, al := newService(t)
	ctx := context.Background()
	// Added before 18+ labels existed: linked to MangaDex, but no rating yet.
	v, err := svc.Add(ctx, libtest.ORVID, AddInput{Status: store.StatusCompleted})
	if err != nil || v.Adult || v.Series.ContentRating != "" {
		t.Fatalf("added: adult %v, rating %q, %v", v.Adult, v.Series.ContentRating, err)
	}

	d := mu.Details[libtest.ORVID]
	d.Genres = []string{"Smut"}
	mu.Details[libtest.ORVID] = d
	md.Links[libtest.ORVID] = mangadex.Links{MDID: "9a414441", ALID: 119257, MALID: 132214,
		Romanized: "Jeonjijeok Dokja Sijeom", Native: "전지적 독자 시점", ContentRating: "pornographic"}
	al.Adult = map[int]bool{119257: true}

	if err := svc.RefreshDetails(ctx, v.Series.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.Get(ctx, v.Series.ID)
	if !reflect.DeepEqual(got.Series.Genres, []string{"Smut"}) || got.Series.ContentRating != "pornographic" ||
		!got.Series.ALAdult || !got.Adult {
		t.Fatalf("after refresh: genres %v, rating %q, AniList adult %v, view adult %v",
			got.Series.Genres, got.Series.ContentRating, got.Series.ALAdult, got.Adult)
	}
}
