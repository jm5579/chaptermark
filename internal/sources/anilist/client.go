// Package anilist fetches AniList's public average score for a manga.
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

const scoreQuery = `query ($id: Int) { Media(id: $id, type: MANGA) { averageScore } }`

// Score returns AniList's averageScore converted from /100 to /10 (one decimal).
// ok is false when AniList has no score.
func (c *Client) Score(ctx context.Context, id int) (float64, bool, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	var resp struct {
		Data struct {
			Media *struct {
				AverageScore *int `json:"averageScore"`
			} `json:"Media"`
		} `json:"data"`
	}
	body := map[string]any{"query": scoreQuery, "variables": map[string]any{"id": id}}
	if err := c.HTTP.PostJSON(ctx, base, body, &resp); err != nil {
		return 0, false, err
	}
	if resp.Data.Media == nil || resp.Data.Media.AverageScore == nil {
		return 0, false, nil
	}
	return math.Round(float64(*resp.Data.Media.AverageScore)) / 10, true, nil
}
