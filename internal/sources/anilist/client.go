// Package anilist fetches AniList's public average score and 18+ flag for a manga.
package anilist

import (
	"context"
	"math"

	"github.com/jm5579/chaptermark/internal/httpx"
)

const DefaultBaseURL = "https://graphql.anilist.co"

type Client struct {
	HTTP    *httpx.Client
	BaseURL string
}

// Media is what ChapterMark uses from an AniList manga entry.
type Media struct {
	Score    float64 // averageScore converted from /100 to /10 (one decimal)
	HasScore bool    // false while AniList has no score
	Adult    bool    // AniList's own 18+ flag
}

const mediaQuery = `query ($id: Int) { Media(id: $id, type: MANGA) { averageScore isAdult } }`

// Media looks up a manga's score and 18+ flag. ok is false when AniList
// returns no entry.
func (c *Client) Media(ctx context.Context, id int) (Media, bool, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	var resp struct {
		Data struct {
			Media *struct {
				AverageScore *int `json:"averageScore"`
				IsAdult      bool `json:"isAdult"`
			} `json:"Media"`
		} `json:"data"`
	}
	body := map[string]any{"query": mediaQuery, "variables": map[string]any{"id": id}}
	if err := c.HTTP.PostJSON(ctx, base, body, &resp); err != nil {
		return Media{}, false, err
	}
	if resp.Data.Media == nil {
		return Media{}, false, nil
	}
	m := Media{Adult: resp.Data.Media.IsAdult}
	if s := resp.Data.Media.AverageScore; s != nil {
		m.Score, m.HasScore = math.Round(float64(*s))/10, true
	}
	return m, true, nil
}
