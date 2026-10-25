package library

import (
	"strings"

	"github.com/jm5579/chaptermark/internal/store"
)

// AdultGenres are the MangaUpdates genres that mark a series as 18+. Ecchi and
// Mature are left out on purpose: they cover fan service and dark themes in
// otherwise ordinary stories. Explicit Yaoi and Yuri also carry Smut or Adult.
var AdultGenres = []string{"Hentai", "Smut", "Adult", "Lolicon", "Shotacon"}

func adultGenre(genres []string) bool {
	for _, g := range genres {
		for _, a := range AdultGenres {
			if strings.EqualFold(g, a) {
				return true
			}
		}
	}
	return false
}

// IsAdult reports whether any source marks a series as 18+: an adult
// MangaUpdates genre, MangaDex's erotica or pornographic rating, or AniList's
// adult flag. A series whose labels aren't known yet counts as not 18+.
func IsAdult(s store.Series) bool {
	return adultGenre(s.Genres) || s.ContentRating == "erotica" || s.ContentRating == "pornographic" || s.ALAdult
}
