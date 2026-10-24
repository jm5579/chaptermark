// Package mangadex links a MangaUpdates series to its MangaDex entry, which in
// turn carries AniList/MAL ids, a cover image, and language-tagged titles.
package mangadex

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/jm5579/chaptermark/internal/httpx"
)

const DefaultBaseURL = "https://api.mangadex.org"

type Client struct {
	HTTP    *httpx.Client
	BaseURL string
}

type Links struct {
	MDID      string
	ALID      int
	MALID     int
	CoverURL  string
	Romanized string // e.g. "Jeonjijeok Dokja Sijeom" (the "<lang>-ro" title)
	Native    string // e.g. "전지적 독자 시점" (the original-language title)
}

// manga is the part of a MangaDex manga record we use.
type manga struct {
	ID         string `json:"id"`
	Attributes struct {
		Title            map[string]string   `json:"title"`
		AltTitles        []map[string]string `json:"altTitles"`
		OriginalLanguage string              `json:"originalLanguage"`
		Links            map[string]string   `json:"links"`
	} `json:"attributes"`
	Relationships []struct {
		Type       string `json:"type"`
		Attributes struct {
			FileName string `json:"fileName"`
		} `json:"attributes"`
	} `json:"relationships"`
}

// MUIDToBase36 converts a MangaUpdates series id to the form MangaDex stores in links.mu.
func MUIDToBase36(id int64) string {
	return strconv.FormatInt(id, 36)
}

func (c *Client) base() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return DefaultBaseURL
}

// FindByMU searches MangaDex by title and returns the entry whose MangaUpdates
// link matches muID.
func (c *Client) FindByMU(ctx context.Context, title string, muID int64) (Links, bool, error) {
	q := url.Values{}
	q.Set("title", title)
	q.Set("limit", "10")
	q.Add("includes[]", "cover_art")
	var resp struct {
		Data []manga `json:"data"`
	}
	if err := c.HTTP.GetJSON(ctx, c.base()+"/manga?"+q.Encode(), &resp); err != nil {
		return Links{}, false, err
	}
	want := MUIDToBase36(muID)
	for _, m := range resp.Data {
		if m.Attributes.Links["mu"] == want {
			return m.links(), true, nil
		}
	}
	return Links{}, false, nil
}

// ByID fetches a MangaDex entry already linked to a series.
func (c *Client) ByID(ctx context.Context, mdID string) (Links, bool, error) {
	var resp struct {
		Data manga `json:"data"`
	}
	if err := c.HTTP.GetJSON(ctx, c.base()+"/manga/"+url.PathEscape(mdID)+"?includes[]=cover_art", &resp); err != nil {
		return Links{}, false, err
	}
	return resp.Data.links(), resp.Data.ID != "", nil
}

func (m manga) links() Links {
	l := Links{MDID: m.ID}
	l.ALID, _ = strconv.Atoi(m.Attributes.Links["al"])
	l.MALID, _ = strconv.Atoi(m.Attributes.Links["mal"])
	for _, r := range m.Relationships {
		if r.Type == "cover_art" && r.Attributes.FileName != "" {
			l.CoverURL = "https://uploads.mangadex.org/covers/" + m.ID + "/" + r.Attributes.FileName + ".512.jpg"
		}
	}
	l.Romanized, l.Native = m.titles()
	return l
}

// titles picks the romanized title ("ko-ro", "ja-ro", "zh-ro"…) and the
// longest original-language title (the shortest is often an abbreviation).
func (m manga) titles() (romanized, native string) {
	lang := m.Attributes.OriginalLanguage
	if lang == "" {
		return "", ""
	}
	all := append([]map[string]string{m.Attributes.Title}, m.Attributes.AltTitles...)
	for _, t := range all {
		for k, v := range t {
			isNative := k == lang || (strings.HasPrefix(k, lang+"-") && !strings.HasSuffix(k, "-ro"))
			switch {
			case k == lang+"-ro" && romanized == "":
				romanized = v
			case isNative && len([]rune(v)) > len([]rune(native)):
				native = v
			}
		}
	}
	return romanized, native
}
