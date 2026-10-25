package library

import (
	"github.com/jm5579/chaptermark/internal/chapters"
	"github.com/jm5579/chaptermark/internal/store"
)

// ItemView is a tracked series as the UI sees it, with derived fields.
type ItemView struct {
	Series          store.Series     `json:"series"`
	Entry           store.Entry      `json:"entry"`
	EffectiveType   store.SeriesType `json:"effectiveType"`
	EffectiveLatest float64          `json:"effectiveLatest"`
	NewCount        int              `json:"newCount"`
	Adult           bool             `json:"adult"` // marked 18+ by any source (see IsAdult)
}

func (v ItemView) Item() store.Item { return store.Item{Series: v.Series, Entry: v.Entry} }

// OnlineResult is a MangaUpdates search hit, flagged if already tracked.
type OnlineResult struct {
	ID         int64            `json:"id"`
	Title      string           `json:"title"`
	HitTitle   string           `json:"hitTitle"`
	Type       string           `json:"type"`
	MappedType store.SeriesType `json:"mappedType"`
	Year       string           `json:"year"`
	URL        string           `json:"url"`
	CoverURL   string           `json:"coverUrl"`
	Rating     float64          `json:"rating"`
	InLibrary  bool             `json:"inLibrary"`
	SeriesID   int64            `json:"seriesId"`
}

// EffectiveType is the type tab a series appears under: the owner's override,
// else MangaUpdates' type, else manga for types outside the three tabs.
func EffectiveType(s store.Series) store.SeriesType {
	if s.TypeOverride != "" {
		return s.TypeOverride
	}
	if s.Type == store.TypeOther || s.Type == "" {
		return store.TypeManga
	}
	return s.Type
}

func viewOf(it store.Item) ItemView {
	latest := chapters.Effective(it.Series.LatestChapter, it.Series.LatestManual)
	return ItemView{
		Series:          it.Series,
		Entry:           it.Entry,
		EffectiveType:   EffectiveType(it.Series),
		EffectiveLatest: latest,
		NewCount:        chapters.NewCount(latest, it.Entry.MyChapter),
		Adult:           IsAdult(it.Series),
	}
}
