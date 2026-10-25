// Package library is the app's core: adding series, recording progress,
// searching, and refreshing data from the sources.
package library

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jm5579/chaptermark/internal/sources/anilist"
	"github.com/jm5579/chaptermark/internal/sources/mangadex"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
	"github.com/jm5579/chaptermark/internal/store"
)

const detailsMaxAge = 7 * 24 * time.Hour

type MU interface {
	Search(ctx context.Context, q string, page int, excludeGenres []string) ([]mangaupdates.SearchResult, int, error)
	Series(ctx context.Context, id int64) (mangaupdates.SeriesDetail, error)
	LatestRelease(ctx context.Context, id int64) (mangaupdates.Release, bool, error)
}

type MD interface {
	FindByMU(ctx context.Context, title string, muID int64) (mangadex.Links, bool, error)
	ByID(ctx context.Context, mdID string) (mangadex.Links, bool, error)
}

type AL interface {
	Media(ctx context.Context, id int) (anilist.Media, bool, error)
}

// CoverFetcher downloads a cover for a series and returns the saved file name.
type CoverFetcher func(ctx context.Context, url string, seriesID int64) (string, error)

type Service struct {
	Store      *store.Store
	MU         MU
	MD         MD
	AL         AL
	FetchCover CoverFetcher
	CoversDir  string // optional; cover files are deleted from here on Remove
	Now        func() time.Time
}

type AddInput struct {
	Status       store.Status     `json:"status"`
	MyChapter    float64          `json:"myChapter"`
	MyRating     int              `json:"myRating"`
	TypeOverride store.SeriesType `json:"typeOverride"`
}

type EntryUpdate struct {
	Status       store.Status     `json:"status"`
	MyChapter    float64          `json:"myChapter"`
	MyRating     int              `json:"myRating"`
	Notes        string           `json:"notes"`
	ReadURL      string           `json:"readUrl"`
	TypeOverride store.SeriesType `json:"typeOverride"`
	LatestManual *float64         `json:"latestManual"`
	Tags         []string         `json:"tags"`
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func validStatus(st store.Status) bool {
	switch st {
	case store.StatusReading, store.StatusPlan, store.StatusCompleted, store.StatusOnHold, store.StatusDropped:
		return true
	}
	return false
}

func validate(st store.Status, chapter float64, rating int, override store.SeriesType) error {
	if !validStatus(st) {
		return fmt.Errorf("unknown status %q", st)
	}
	if chapter < 0 {
		return fmt.Errorf("chapter can't be negative")
	}
	if rating < 0 || rating > 10 {
		return fmt.Errorf("rating must be 1-10 (or 0 for unrated)")
	}
	switch override {
	case "", store.TypeManhwa, store.TypeManga, store.TypeManhua:
		return nil
	}
	return fmt.Errorf("unknown type %q", override)
}

func (s *Service) List(ctx context.Context) ([]ItemView, error) {
	items, err := s.Store.ListItems(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ItemView, len(items))
	for i, it := range items {
		out[i] = viewOf(it)
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, seriesID int64) (ItemView, error) {
	it, err := s.Store.GetItem(ctx, seriesID)
	if err != nil {
		return ItemView{}, err
	}
	return viewOf(it), nil
}

func (s *Service) SearchLocal(ctx context.Context, q string) ([]ItemView, error) {
	items, err := s.Store.SearchLocal(ctx, q, 20)
	if err != nil {
		return nil, err
	}
	out := make([]ItemView, len(items))
	for i, it := range items {
		out[i] = viewOf(it)
	}
	return out, nil
}

// SearchOnline searches MangaUpdates. Types other than manga/manhwa/manhua are
// dropped unless includeOther is set, and series with an 18+ genre unless
// includeAdult is set. MangaUpdates leaves those out itself, so a page stays
// full; they are checked again here in case it ever doesn't.
func (s *Service) SearchOnline(ctx context.Context, q string, page int, includeOther, includeAdult bool) ([]OnlineResult, error) {
	var exclude []string
	if !includeAdult {
		exclude = AdultGenres
	}
	results, _, err := s.MU.Search(ctx, q, page, exclude)
	if err != nil {
		return nil, err
	}
	out := []OnlineResult{}
	for _, r := range results {
		mapped := mangaupdates.MapType(r.Type)
		if (mapped == store.TypeOther && !includeOther) || (!includeAdult && adultGenre(r.Genres)) {
			continue
		}
		or := OnlineResult{ID: r.ID, Title: r.Title, HitTitle: r.HitTitle, Type: r.Type, MappedType: mapped,
			Year: r.Year, URL: r.URL, CoverURL: r.CoverURL, Rating: r.Rating}
		if existing, ok, err := s.Store.SeriesByMUID(ctx, r.ID); err == nil && ok {
			or.InLibrary, or.SeriesID = true, existing.ID
		}
		out = append(out, or)
	}
	return out, nil
}

// applyDetails copies MangaUpdates details onto sr, keeping owner-set fields.
func applyDetails(sr *store.Series, d mangaupdates.SeriesDetail) {
	sr.Title, sr.MUURL, sr.Year, sr.PubStatus = d.Title, d.URL, d.Year, d.Status
	sr.Type, sr.Completed = mangaupdates.MapType(d.Type), d.Completed
	sr.MURating, sr.MUVotes = d.Rating, d.Votes
	sr.AltTitles, sr.Genres = d.AltTitles, d.Genres
	if d.CoverURL != "" {
		sr.CoverURL = d.CoverURL
	}
}

// enrich adds MangaDex links and content rating, the AniList score and adult
// flag, and the cover. Failures here are logged, never fatal: MangaUpdates
// data alone is enough.
func (s *Service) enrich(ctx context.Context, sr *store.Series) {
	if s.MD != nil && (sr.MDID == "" || (sr.RomanizedTitle == "" && sr.NativeTitle == "") || sr.ContentRating == "") {
		var links mangadex.Links
		var ok bool
		var err error
		if sr.MDID == "" {
			links, ok, err = s.MD.FindByMU(ctx, sr.Title, sr.MUID)
		} else {
			// Linked before titles or the content rating were stored: fetch them by id.
			links, ok, err = s.MD.ByID(ctx, sr.MDID)
		}
		if err != nil {
			log.Printf("mangadex lookup for %q: %v", sr.Title, err)
		} else if ok {
			sr.MDID, sr.ALID, sr.MALID = links.MDID, links.ALID, links.MALID
			sr.RomanizedTitle, sr.NativeTitle = links.Romanized, links.Native
			sr.ContentRating = links.ContentRating
			if sr.CoverURL == "" {
				sr.CoverURL = links.CoverURL
			}
		}
	}
	if sr.ALID > 0 && s.AL != nil {
		if m, ok, err := s.AL.Media(ctx, sr.ALID); err != nil {
			log.Printf("anilist lookup for %q: %v", sr.Title, err)
		} else if ok {
			if m.HasScore {
				sr.ALRating = m.Score
			}
			sr.ALAdult = m.Adult
		}
	}
}

func (s *Service) fetchCover(ctx context.Context, sr *store.Series) {
	if sr.CoverURL == "" || s.FetchCover == nil || sr.ID == 0 {
		return
	}
	name, err := s.FetchCover(ctx, sr.CoverURL, sr.ID)
	if err != nil {
		log.Printf("cover for %q: %v", sr.Title, err)
		return
	}
	sr.CoverFile = name
}

// Add starts tracking a MangaUpdates series. Adding one that is already tracked
// returns it unchanged.
func (s *Service) Add(ctx context.Context, muID int64, in AddInput) (ItemView, error) {
	if in.Status == "" {
		in.Status = store.StatusReading
	}
	if err := validate(in.Status, in.MyChapter, in.MyRating, in.TypeOverride); err != nil {
		return ItemView{}, err
	}
	if existing, ok, err := s.Store.SeriesByMUID(ctx, muID); err != nil {
		return ItemView{}, err
	} else if ok {
		return s.Get(ctx, existing.ID)
	}

	d, err := s.MU.Series(ctx, muID)
	if err != nil {
		return ItemView{}, fmt.Errorf("couldn't load the series from MangaUpdates: %w", err)
	}
	now := s.now()
	sr := store.Series{MUID: muID, TypeOverride: in.TypeOverride, DetailsRefreshedAt: now}
	applyDetails(&sr, d)
	if sr.Type == store.TypeOther && sr.TypeOverride == "" {
		sr.TypeOverride = store.TypeManga
	}
	s.enrich(ctx, &sr)
	if rel, ok, err := s.MU.LatestRelease(ctx, muID); err != nil {
		log.Printf("latest release for %q: %v (the checker will retry)", sr.Title, err)
	} else {
		sr.ChaptersCheckedAt = now
		if ok {
			sr.LatestChapter, sr.LatestRaw, sr.LatestGroup, sr.LatestDate = rel.Chapter, rel.Raw, rel.Group, rel.Date
			sr.ReleaseIntervalDays, sr.LastReleaseDate = rel.IntervalDays, rel.LastReleaseDate
		}
	}
	if err := s.Store.UpsertSeries(ctx, &sr); err != nil {
		return ItemView{}, err
	}
	e := store.Entry{SeriesID: sr.ID, Status: in.Status, MyChapter: in.MyChapter, MyRating: in.MyRating,
		AddedAt: now, UpdatedAt: now}
	if in.MyChapter > 0 {
		e.LastReadAt = now
	}
	if err := s.Store.SaveEntry(ctx, e); err != nil {
		s.Store.DeleteSeries(ctx, sr.ID)
		return ItemView{}, err
	}
	s.fetchCover(ctx, &sr)
	if sr.CoverFile != "" {
		if err := s.Store.UpsertSeries(ctx, &sr); err != nil {
			return ItemView{}, err
		}
	}
	return s.Get(ctx, sr.ID)
}

// UpdateEntry saves the owner's edits. "Last read" moves to now only when the
// chapter changes.
func (s *Service) UpdateEntry(ctx context.Context, seriesID int64, in EntryUpdate) (ItemView, error) {
	if err := validate(in.Status, in.MyChapter, in.MyRating, in.TypeOverride); err != nil {
		return ItemView{}, err
	}
	if in.LatestManual != nil && *in.LatestManual < 0 {
		return ItemView{}, fmt.Errorf("latest chapter can't be negative")
	}
	if err := validateTags(in.Tags); err != nil {
		return ItemView{}, err
	}
	it, err := s.Store.GetItem(ctx, seriesID)
	if err != nil {
		return ItemView{}, err
	}
	now := s.now()
	e := it.Entry
	if in.MyChapter != e.MyChapter {
		e.LastReadAt = now
	}
	e.Status, e.MyChapter, e.MyRating, e.Notes, e.ReadURL = in.Status, in.MyChapter, in.MyRating, in.Notes, in.ReadURL
	e.UpdatedAt = now
	if err := s.Store.SaveEntry(ctx, e); err != nil {
		return ItemView{}, err
	}
	override := in.TypeOverride
	if override == "" && (it.Series.Type == store.TypeOther || it.Series.Type == "") {
		override = store.TypeManga
	}
	if override != it.Series.TypeOverride {
		sr := it.Series
		sr.TypeOverride = override
		if err := s.Store.UpsertSeries(ctx, &sr); err != nil {
			return ItemView{}, err
		}
	}
	if err := s.Store.SetLatestManual(ctx, seriesID, in.LatestManual); err != nil {
		return ItemView{}, err
	}
	if err := s.Store.SetSeriesTags(ctx, seriesID, in.Tags); err != nil {
		return ItemView{}, err
	}
	return s.Get(ctx, seriesID)
}

const (
	maxTagsPerSeries = 20
	maxTagLength     = 40
)

func validateTags(tags []string) error {
	clean := store.CleanTagNames(tags)
	if len(clean) > maxTagsPerSeries {
		return fmt.Errorf("a series can have at most %d tags", maxTagsPerSeries)
	}
	for _, t := range clean {
		if len([]rune(t)) > maxTagLength {
			return fmt.Errorf("tag names can be at most %d characters", maxTagLength)
		}
	}
	return nil
}

// Tags lists the owner's tags with how many series use each.
func (s *Service) Tags(ctx context.Context) ([]store.Tag, error) { return s.Store.ListTags(ctx) }

// RenameTag renames a tag (merging into an existing tag of the same name).
func (s *Service) RenameTag(ctx context.Context, id int64, name string) error {
	if len([]rune(strings.TrimSpace(name))) > maxTagLength {
		return fmt.Errorf("tag names can be at most %d characters", maxTagLength)
	}
	return s.Store.RenameTag(ctx, id, name)
}

// DeleteTag removes a tag from every series.
func (s *Service) DeleteTag(ctx context.Context, id int64) error { return s.Store.DeleteTag(ctx, id) }

func (s *Service) Remove(ctx context.Context, seriesID int64) error {
	it, err := s.Store.GetItem(ctx, seriesID)
	if err != nil {
		return err
	}
	if err := s.Store.DeleteSeries(ctx, seriesID); err != nil {
		return err
	}
	if s.CoversDir != "" && it.Series.CoverFile != "" {
		os.Remove(filepath.Join(s.CoversDir, it.Series.CoverFile))
	}
	return nil
}

// RefreshDetails reloads names, ratings, links and the cover for a series.
func (s *Service) RefreshDetails(ctx context.Context, seriesID int64) error {
	it, err := s.Store.GetItem(ctx, seriesID)
	if err != nil {
		return err
	}
	d, err := s.MU.Series(ctx, it.Series.MUID)
	if err != nil {
		return err
	}
	sr := it.Series
	applyDetails(&sr, d)
	s.enrich(ctx, &sr)
	s.fetchCover(ctx, &sr)
	sr.DetailsRefreshedAt = s.now()
	// Only metadata is written: the owner may have edited this series while
	// the network calls above were running.
	return s.Store.UpdateDetails(ctx, &sr)
}

// RefreshLatest looks up the newest release for a series and stores it.
func (s *Service) RefreshLatest(ctx context.Context, seriesID int64) error {
	it, err := s.Store.GetItem(ctx, seriesID)
	if err != nil {
		return err
	}
	rel, ok, err := s.MU.LatestRelease(ctx, it.Series.MUID)
	if err != nil {
		return err
	}
	l := store.Latest{Chapter: it.Series.LatestChapter, Raw: it.Series.LatestRaw, Group: it.Series.LatestGroup, Date: it.Series.LatestDate,
		IntervalDays: it.Series.ReleaseIntervalDays, LastReleaseDate: it.Series.LastReleaseDate}
	if ok {
		l = store.Latest{Chapter: rel.Chapter, Raw: rel.Raw, Group: rel.Group, Date: rel.Date,
			IntervalDays: rel.IntervalDays, LastReleaseDate: rel.LastReleaseDate}
	}
	return s.Store.SetLatest(ctx, seriesID, l, s.now())
}

// DetailsStale reports whether a series' details are older than 7 days.
func (s *Service) DetailsStale(it store.Item) bool {
	return s.now().Sub(it.Series.DetailsRefreshedAt) > detailsMaxAge
}
