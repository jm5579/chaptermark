// Package mangaupdates talks to the public MangaUpdates API v1 (no key needed).
package mangaupdates

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jm5579/chaptermark/internal/chapters"
	"github.com/jm5579/chaptermark/internal/httpx"
	"github.com/jm5579/chaptermark/internal/store"
)

const DefaultBaseURL = "https://api.mangaupdates.com/v1"

type Client struct {
	HTTP    *httpx.Client
	BaseURL string
}

type SearchResult struct {
	ID       int64    `json:"id"`
	Title    string   `json:"title"`
	HitTitle string   `json:"hitTitle"`
	Type     string   `json:"type"`
	Year     string   `json:"year"`
	URL      string   `json:"url"`
	CoverURL string   `json:"coverUrl"`
	Rating   float64  `json:"rating"`
	Genres   []string `json:"genres"`
}

type SeriesDetail struct {
	ID        int64
	Title     string
	URL       string
	Type      string
	Year      string
	Status    string
	Completed bool
	Rating    float64
	Votes     int
	CoverURL  string
	AltTitles []string
	Genres    []string
}

type Release struct {
	Chapter float64
	Raw     string
	Group   string
	Date    string
	// Release rhythm from the same list: typical days between new chapters
	// (0 = not enough data) and the date of the most recent release.
	IntervalDays    float64
	LastReleaseDate string
}

// dated is one release: a chapter number and its release date (YYYY-MM-DD).
type dated struct {
	chapter float64
	date    string
}

// releaseInterval is the median number of days between the 10 most recent new
// chapters. Each chapter counts once, at the earliest date any group released
// it, so official releases that lag behind scanlations don't skew it.
func releaseInterval(releases []dated) float64 {
	first := map[float64]string{}
	for _, r := range releases {
		if d, ok := first[r.chapter]; !ok || r.date < d {
			first[r.chapter] = r.date
		}
	}
	chs := make([]float64, 0, len(first))
	for ch := range first {
		chs = append(chs, ch)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(chs)))
	if len(chs) > 11 {
		chs = chs[:11]
	}
	var gaps []float64
	for i := 0; i+1 < len(chs); i++ {
		newer, err1 := time.Parse("2006-01-02", first[chs[i]])
		older, err2 := time.Parse("2006-01-02", first[chs[i+1]])
		if err1 != nil || err2 != nil {
			continue
		}
		if days := newer.Sub(older).Hours() / 24; days > 0 {
			gaps = append(gaps, days)
		}
	}
	if len(gaps) < 3 {
		return 0
	}
	sort.Float64s(gaps)
	mid := len(gaps) / 2
	if len(gaps)%2 == 1 {
		return gaps[mid]
	}
	return (gaps[mid-1] + gaps[mid]) / 2
}

type image struct {
	URL struct {
		Original string `json:"original"`
		Thumb    string `json:"thumb"`
	} `json:"url"`
}

type seriesRecord struct {
	SeriesID       int64   `json:"series_id"`
	Title          string  `json:"title"`
	URL            string  `json:"url"`
	Type           string  `json:"type"`
	Year           string  `json:"year"`
	BayesianRating float64 `json:"bayesian_rating"`
	RatingVotes    int     `json:"rating_votes"`
	Status         string  `json:"status"`
	Completed      bool    `json:"completed"`
	Image          image   `json:"image"`
	Associated     []struct {
		Title string `json:"title"`
	} `json:"associated"`
	Genres []struct {
		Genre string `json:"genre"`
	} `json:"genres"`
}

func (r seriesRecord) genres() []string {
	out := []string{}
	for _, g := range r.Genres {
		if g.Genre != "" {
			out = append(out, g.Genre)
		}
	}
	return out
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

// MapType converts a MangaUpdates type name to one of ChapterMark's types.
func MapType(muType string) store.SeriesType {
	switch strings.ToLower(muType) {
	case "manhwa":
		return store.TypeManhwa
	case "manga":
		return store.TypeManga
	case "manhua":
		return store.TypeManhua
	}
	return store.TypeOther
}

// Search finds series by any of their names, leaving out series with any of
// excludeGenres. It returns one page of results and the total hit count.
func (c *Client) Search(ctx context.Context, q string, page int, excludeGenres []string) ([]SearchResult, int, error) {
	if page < 1 {
		page = 1
	}
	var resp struct {
		TotalHits int `json:"total_hits"`
		Results   []struct {
			Record   seriesRecord `json:"record"`
			HitTitle string       `json:"hit_title"`
		} `json:"results"`
	}
	body := map[string]any{"search": q, "page": page, "perpage": 25}
	if len(excludeGenres) > 0 {
		body["exclude_genre"] = excludeGenres
	}
	if err := c.HTTP.PostJSON(ctx, c.base()+"/series/search", body, &resp); err != nil {
		return nil, 0, err
	}
	out := make([]SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		out = append(out, SearchResult{
			ID: r.Record.SeriesID, Title: r.Record.Title, HitTitle: r.HitTitle, Type: r.Record.Type,
			Year: r.Record.Year, URL: r.Record.URL, CoverURL: r.Record.Image.URL.Original, Rating: r.Record.BayesianRating,
			Genres: r.Record.genres(),
		})
	}
	return out, resp.TotalHits, nil
}

// Series returns full details for one series, including all associated names.
func (c *Client) Series(ctx context.Context, id int64) (SeriesDetail, error) {
	var r seriesRecord
	if err := c.HTTP.GetJSON(ctx, fmt.Sprintf("%s/series/%d", c.base(), id), &r); err != nil {
		return SeriesDetail{}, err
	}
	d := SeriesDetail{
		ID: r.SeriesID, Title: r.Title, URL: r.URL, Type: r.Type, Year: r.Year, Status: r.Status,
		Completed: r.Completed, Rating: r.BayesianRating, Votes: r.RatingVotes, CoverURL: r.Image.URL.Original,
		Genres: r.genres(),
	}
	for _, a := range r.Associated {
		if a.Title != "" {
			d.AltTitles = append(d.AltTitles, a.Title)
		}
	}
	return d, nil
}

// LatestRelease returns the highest-numbered chapter among the most recent
// releases. MangaUpdates' "chap" ordering sorts as text ("99" > "311"), so we
// take the newest page by date and compare parsed numbers ourselves.
func (c *Client) LatestRelease(ctx context.Context, id int64) (Release, bool, error) {
	var resp struct {
		Results []struct {
			Record struct {
				Chapter     string `json:"chapter"`
				ReleaseDate string `json:"release_date"`
				Groups      []struct {
					Name string `json:"name"`
				} `json:"groups"`
			} `json:"record"`
		} `json:"results"`
	}
	body := map[string]any{
		"search": strconv.FormatInt(id, 10), "search_type": "series",
		"orderby": "date", "asc": "desc", "page": 1,
	}
	if err := c.HTTP.PostJSON(ctx, c.base()+"/releases/search", body, &resp); err != nil {
		return Release{}, false, err
	}
	var best Release
	found := false
	var all []dated
	lastRelease := ""
	for _, r := range resp.Results {
		if r.Record.ReleaseDate > lastRelease {
			lastRelease = r.Record.ReleaseDate
		}
		n, ok := chapters.Parse(r.Record.Chapter)
		if ok {
			all = append(all, dated{n, r.Record.ReleaseDate})
		}
		if !ok || (found && n <= best.Chapter) {
			continue
		}
		names := make([]string, 0, len(r.Record.Groups))
		for _, g := range r.Record.Groups {
			names = append(names, g.Name)
		}
		best = Release{Chapter: n, Raw: r.Record.Chapter, Group: strings.Join(names, " & "), Date: r.Record.ReleaseDate}
		found = true
	}
	best.IntervalDays = releaseInterval(all)
	best.LastReleaseDate = lastRelease
	return best, found, nil
}
