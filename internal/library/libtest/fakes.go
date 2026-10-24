// Package libtest provides in-memory fakes of the data sources for tests of
// the library service and the checker.
package libtest

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/jm5579/chaptermark/internal/sources/mangadex"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
)

const ORVID int64 = 50369844984

// MU is a fake MangaUpdates client.
type MU struct {
	Results  []mangaupdates.SearchResult
	Details  map[int64]mangaupdates.SeriesDetail
	Releases map[int64]mangaupdates.Release
	Err      error // returned by every call when set

	mu    sync.Mutex
	Calls []int64 // series ids passed to LatestRelease, in order
}

func (f *MU) Search(ctx context.Context, q string, page int) ([]mangaupdates.SearchResult, int, error) {
	if f.Err != nil {
		return nil, 0, f.Err
	}
	return f.Results, len(f.Results), nil
}

func (f *MU) Series(ctx context.Context, id int64) (mangaupdates.SeriesDetail, error) {
	if f.Err != nil {
		return mangaupdates.SeriesDetail{}, f.Err
	}
	d, ok := f.Details[id]
	if !ok {
		return d, fmt.Errorf("series %d: %w", id, errors.New("HTTP 404"))
	}
	return d, nil
}

func (f *MU) LatestRelease(ctx context.Context, id int64) (mangaupdates.Release, bool, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, id)
	f.mu.Unlock()
	if f.Err != nil {
		return mangaupdates.Release{}, false, f.Err
	}
	r, ok := f.Releases[id]
	return r, ok, nil
}

// MD is a fake MangaDex client.
type MD struct {
	Links map[int64]mangadex.Links
	Err   error
}

func (f *MD) FindByMU(ctx context.Context, title string, muID int64) (mangadex.Links, bool, error) {
	if f.Err != nil {
		return mangadex.Links{}, false, f.Err
	}
	l, ok := f.Links[muID]
	return l, ok, nil
}

// AL is a fake AniList client.
type AL struct {
	Scores map[int]float64
	Err    error
}

func (f *AL) Score(ctx context.Context, id int) (float64, bool, error) {
	if f.Err != nil {
		return 0, false, f.Err
	}
	s, ok := f.Scores[id]
	return s, ok, nil
}

// Covers is a cover fetcher that pretends to save <id>.jpg.
func Covers(ctx context.Context, url string, seriesID int64) (string, error) {
	return fmt.Sprintf("%d.jpg", seriesID), nil
}

// ORV returns fakes pre-loaded with Omniscient Reader (MU id ORVID).
func ORV() (*MU, *MD, *AL) {
	mu := &MU{
		Results: []mangaupdates.SearchResult{
			{ID: ORVID, Title: "Omniscient Reader", HitTitle: "Jeonjijeok Dokja Sijeom", Type: "Manhwa", Year: "2020", Rating: 8.82},
			{ID: 25659407449, Title: "Omniscient Reader's Viewpoint (Novel)", Type: "Novel", Year: "2018", Rating: 8.06},
		},
		Details: map[int64]mangaupdates.SeriesDetail{
			ORVID: {ID: ORVID, Title: "Omniscient Reader", URL: "https://www.mangaupdates.com/series/n50wl4o", Type: "Manhwa",
				Year: "2020", Status: "311 Chapters (Hiatus)", Rating: 8.82, Votes: 949, CoverURL: "https://cdn.example/orv.jpg",
				AltTitles: []string{"Jeonjijeok Dokja Sijeom", "ORV", "전지적 독자 시점"}},
		},
		Releases: map[int64]mangaupdates.Release{
			ORVID: {Chapter: 311, Raw: "311", Group: "Flame Comics", Date: "2026-05-19"},
		},
	}
	md := &MD{Links: map[int64]mangadex.Links{ORVID: {MDID: "9a414441", ALID: 119257, MALID: 132214}}}
	al := &AL{Scores: map[int]float64{119257: 8.6}}
	return mu, md, al
}
