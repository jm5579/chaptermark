// Package exportimport saves the whole library to one JSON file and restores it.
package exportimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jm5579/chaptermark/internal/store"
)

const (
	appName = "ChapterMark"
	version = 1
)

type Mode string

const (
	ModeMerge   Mode = "merge"
	ModeReplace Mode = "replace"
)

type File struct {
	App        string       `json:"app"`
	Version    int          `json:"version"`
	ExportedAt time.Time    `json:"exportedAt"`
	Items      []store.Item `json:"items"`
}

type Result struct {
	Added   int `json:"added"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

var ErrNotChapterMark = errors.New("this file isn't a ChapterMark export")

func Export(ctx context.Context, st *store.Store, w io.Writer, now time.Time) error {
	items, err := st.ListItems(ctx)
	if err != nil {
		return err
	}
	if items == nil {
		items = []store.Item{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(File{App: appName, Version: version, ExportedAt: now.UTC(), Items: items})
}

// Import reads an export. Replace wipes the library first; merge adds unknown
// series and, for known ones, keeps whichever entry was updated more recently.
// The file is fully validated before anything is changed.
func Import(ctx context.Context, st *store.Store, r io.Reader, mode Mode) (Result, error) {
	var f File
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return Result{}, fmt.Errorf("%w: %v", ErrNotChapterMark, err)
	}
	if f.App != appName {
		return Result{}, ErrNotChapterMark
	}
	if f.Version > version {
		return Result{}, fmt.Errorf("this export is from a newer version of ChapterMark (format %d)", f.Version)
	}
	if mode != ModeMerge && mode != ModeReplace {
		return Result{}, fmt.Errorf("unknown import mode %q", mode)
	}

	if mode == ModeReplace {
		existing, err := st.ListItems(ctx)
		if err != nil {
			return Result{}, err
		}
		for _, it := range existing {
			if err := st.DeleteSeries(ctx, it.Series.ID); err != nil {
				return Result{}, err
			}
		}
	}

	var res Result
	for _, it := range f.Items {
		if it.Series.MUID == 0 || it.Series.Title == "" {
			res.Skipped++
			continue
		}
		existing, found, err := st.SeriesByMUID(ctx, it.Series.MUID)
		if err != nil {
			return res, err
		}
		if found {
			cur, err := st.GetItem(ctx, existing.ID)
			if err != nil {
				return res, err
			}
			if !it.Entry.UpdatedAt.After(cur.Entry.UpdatedAt) {
				res.Skipped++
				continue
			}
			existing.TypeOverride = it.Series.TypeOverride
			existing.LatestManual = it.Series.LatestManual
			if err := st.UpsertSeries(ctx, &existing); err != nil {
				return res, err
			}
			it.Entry.SeriesID = existing.ID
			if err := st.SaveEntry(ctx, it.Entry); err != nil {
				return res, err
			}
			res.Updated++
			continue
		}
		sr := it.Series
		sr.ID = 0
		sr.CoverFile = ""                   // covers aren't in the file; re-download
		sr.DetailsRefreshedAt = time.Time{} // so the checker refreshes details soon
		if err := st.UpsertSeries(ctx, &sr); err != nil {
			return res, err
		}
		it.Entry.SeriesID = sr.ID
		if err := st.SaveEntry(ctx, it.Entry); err != nil {
			return res, err
		}
		res.Added++
	}
	return res, nil
}
