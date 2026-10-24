package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Tag is one of the owner's custom lists, with how many series carry it.
type Tag struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

const deleteOrphanTags = `DELETE FROM tags WHERE id NOT IN (SELECT tag_id FROM series_tags)`

// CleanTagNames trims names, drops empty ones and merges duplicates that differ
// only in case (keeping the first spelling).
func CleanTagNames(names []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		key := strings.ToLower(n)
		if n == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, n)
	}
	return out
}

// SetSeriesTags replaces a series' tags. Tags are matched case-insensitively,
// so "favourites" joins an existing "Favourites". Tags left unused are removed.
func (s *Store) SetSeriesTags(ctx context.Context, seriesID int64, names []string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM series_tags WHERE series_id = ?", seriesID); err != nil {
		return err
	}
	for _, name := range CleanTagNames(names) {
		if _, err := tx.ExecContext(ctx, "INSERT INTO tags(name) VALUES(?) ON CONFLICT(name) DO NOTHING", name); err != nil {
			return err
		}
		var tagID int64
		if err := tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ?", name).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO series_tags(series_id, tag_id) VALUES(?, ?)", seriesID, tagID); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, deleteOrphanTags); err != nil {
		return err
	}
	return tx.Commit()
}

// ListTags returns every tag in use, by name.
func (s *Store) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.id, t.name, COUNT(st.series_id) FROM tags t
		JOIN series_tags st ON st.tag_id = t.id GROUP BY t.id ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Count); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// RenameTag renames a tag. If another tag already has that name (in any
// case), the two are merged into the existing one.
func (s *Store) RenameTag(ctx context.Context, id int64, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("a tag needs a name")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existing int64
	err = tx.QueryRowContext(ctx, "SELECT id FROM tags WHERE name = ? AND id <> ?", name, id).Scan(&existing)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx, "UPDATE tags SET name = ? WHERE id = ?", name, id); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO series_tags(series_id, tag_id)
			SELECT series_id, ? FROM series_tags WHERE tag_id = ?`, existing, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// DeleteTag removes a tag from every series.
func (s *Store) DeleteTag(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM tags WHERE id = ?", id)
	return err
}

func (s *Store) seriesTags(ctx context.Context, seriesID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT t.name FROM series_tags st JOIN tags t ON t.id = st.tag_id
		WHERE st.series_id = ? ORDER BY t.name COLLATE NOCASE`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := []string{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		tags = append(tags, n)
	}
	return tags, rows.Err()
}

func (s *Store) allSeriesTags(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT st.series_id, t.name FROM series_tags st JOIN tags t ON t.id = st.tag_id
		ORDER BY t.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var id int64
		var n string
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = append(out[id], n)
	}
	return out, rows.Err()
}
