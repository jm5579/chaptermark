// Package mangadex links a MangaUpdates series to its MangaDex entry, which in
// turn carries AniList/MAL ids and a cover image.
package mangadex

import (
	"context"
	"net/url"
	"strconv"

	"github.com/jm5579/chaptermark/internal/httpx"
)

const DefaultBaseURL = "https://api.mangadex.org"

type Client struct {
	HTTP    *httpx.Client
	BaseURL string
}

type Links struct {
	MDID     string
	ALID     int
	MALID    int
	CoverURL string
}

// MUIDToBase36 converts a MangaUpdates series id to the form MangaDex stores in links.mu.
func MUIDToBase36(id int64) string {
	return strconv.FormatInt(id, 36)
}

// FindByMU searches MangaDex by title and returns the entry whose MangaUpdates
// link matches muID.
func (c *Client) FindByMU(ctx context.Context, title string, muID int64) (Links, bool, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	q := url.Values{}
	q.Set("title", title)
	q.Set("limit", "10")
	q.Add("includes[]", "cover_art")
	var resp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Links map[string]string `json:"links"`
			} `json:"attributes"`
			Relationships []struct {
				Type       string `json:"type"`
				Attributes struct {
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := c.HTTP.GetJSON(ctx, base+"/manga?"+q.Encode(), &resp); err != nil {
		return Links{}, false, err
	}
	want := MUIDToBase36(muID)
	for _, d := range resp.Data {
		if d.Attributes.Links["mu"] != want {
			continue
		}
		l := Links{MDID: d.ID}
		l.ALID, _ = strconv.Atoi(d.Attributes.Links["al"])
		l.MALID, _ = strconv.Atoi(d.Attributes.Links["mal"])
		for _, r := range d.Relationships {
			if r.Type == "cover_art" && r.Attributes.FileName != "" {
				l.CoverURL = "https://uploads.mangadex.org/covers/" + d.ID + "/" + r.Attributes.FileName + ".512.jpg"
			}
		}
		return l, true, nil
	}
	return Links{}, false, nil
}
