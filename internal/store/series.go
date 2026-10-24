package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/jm5579/chaptermark/internal/textnorm"
)

const seriesCols = `s.id, s.mu_id, s.md_id, s.al_id, s.mal_id, s.title, s.mu_url, s.year, s.pub_status,
	s.type, s.type_override, s.completed, s.cover_file, s.cover_url, s.mu_rating, s.mu_votes, s.al_rating,
	s.latest_chapter, s.latest_raw, s.latest_group, s.latest_date, s.latest_manual,
	s.details_refreshed_at, s.chapters_checked_at`

const entryCols = `e.series_id, e.status, e.my_chapter, e.my_rating, e.notes, e.read_url,
	e.last_read_at, e.added_at, e.updated_at`

type scanner interface{ Scan(dest ...any) error }

func scanItem(row scanner, withEntry bool) (Item, error) {
	var it Item
	sr := &it.Series
	var manual sql.NullFloat64
	var detailsAt, checkedAt string
	dest := []any{&sr.ID, &sr.MUID, &sr.MDID, &sr.ALID, &sr.MALID, &sr.Title, &sr.MUURL, &sr.Year, &sr.PubStatus,
		&sr.Type, &sr.TypeOverride, &sr.Completed, &sr.CoverFile, &sr.CoverURL, &sr.MURating, &sr.MUVotes, &sr.ALRating,
		&sr.LatestChapter, &sr.LatestRaw, &sr.LatestGroup, &sr.LatestDate, &manual, &detailsAt, &checkedAt}
	e := &it.Entry
	var lastRead, added, updated string
	if withEntry {
		dest = append(dest, &e.SeriesID, &e.Status, &e.MyChapter, &e.MyRating, &e.Notes, &e.ReadURL, &lastRead, &added, &updated)
	}
	if err := row.Scan(dest...); err != nil {
		return Item{}, err
	}
	if manual.Valid {
		v := manual.Float64
		sr.LatestManual = &v
	}
	sr.DetailsRefreshedAt, sr.ChaptersCheckedAt = parseTime(detailsAt), parseTime(checkedAt)
	e.LastReadAt, e.AddedAt, e.UpdatedAt = parseTime(lastRead), parseTime(added), parseTime(updated)
	return it, nil
}

func nullable(v *float64) any {
	if v == nil {
		return nil
	}
	return *v
}

// UpsertSeries inserts or updates sr by MUID, sets sr.ID, and replaces its
// alternate titles (the main title is always first).
func (s *Store) UpsertSeries(ctx context.Context, sr *Series) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `INSERT INTO series (mu_id, md_id, al_id, mal_id, title, mu_url, year, pub_status,
		type, type_override, completed, cover_file, cover_url, mu_rating, mu_votes, al_rating,
		latest_chapter, latest_raw, latest_group, latest_date, latest_manual, details_refreshed_at, chapters_checked_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(mu_id) DO UPDATE SET md_id=excluded.md_id, al_id=excluded.al_id, mal_id=excluded.mal_id,
		title=excluded.title, mu_url=excluded.mu_url, year=excluded.year, pub_status=excluded.pub_status,
		type=excluded.type, type_override=excluded.type_override, completed=excluded.completed,
		cover_file=excluded.cover_file, cover_url=excluded.cover_url, mu_rating=excluded.mu_rating,
		mu_votes=excluded.mu_votes, al_rating=excluded.al_rating, latest_chapter=excluded.latest_chapter,
		latest_raw=excluded.latest_raw, latest_group=excluded.latest_group, latest_date=excluded.latest_date,
		latest_manual=excluded.latest_manual, details_refreshed_at=excluded.details_refreshed_at,
		chapters_checked_at=excluded.chapters_checked_at
		RETURNING id`,
		sr.MUID, sr.MDID, sr.ALID, sr.MALID, sr.Title, sr.MUURL, sr.Year, sr.PubStatus,
		sr.Type, sr.TypeOverride, sr.Completed, sr.CoverFile, sr.CoverURL, sr.MURating, sr.MUVotes, sr.ALRating,
		sr.LatestChapter, sr.LatestRaw, sr.LatestGroup, sr.LatestDate, nullable(sr.LatestManual),
		fmtTime(sr.DetailsRefreshedAt), fmtTime(sr.ChaptersCheckedAt)).Scan(&sr.ID)
	if err != nil {
		return err
	}
	titles, err := replaceAltTitles(ctx, tx, sr.ID, sr.Title, sr.AltTitles)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	sr.AltTitles = titles
	return nil
}

// UpdateDetails writes only the public metadata of sr (names, type, ratings,
// links, cover) and its alternate titles. Owner-set fields (type override,
// manual latest) and the latest-chapter columns are left as they are in the
// database, so a slow refresh can't undo an edit made while it was running.
func (s *Store) UpdateDetails(ctx context.Context, sr *Series) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE series SET md_id = ?, al_id = ?, mal_id = ?, title = ?, mu_url = ?,
		year = ?, pub_status = ?, type = ?, completed = ?, cover_file = ?, cover_url = ?, mu_rating = ?,
		mu_votes = ?, al_rating = ?, details_refreshed_at = ? WHERE id = ?`,
		sr.MDID, sr.ALID, sr.MALID, sr.Title, sr.MUURL, sr.Year, sr.PubStatus, sr.Type, sr.Completed,
		sr.CoverFile, sr.CoverURL, sr.MURating, sr.MUVotes, sr.ALRating, fmtTime(sr.DetailsRefreshedAt), sr.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	titles, err := replaceAltTitles(ctx, tx, sr.ID, sr.Title, sr.AltTitles)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	sr.AltTitles = titles
	return nil
}

func replaceAltTitles(ctx context.Context, tx *sql.Tx, seriesID int64, title string, alts []string) ([]string, error) {
	if _, err := tx.ExecContext(ctx, "DELETE FROM alt_titles WHERE series_id = ?", seriesID); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	titles := []string{}
	for _, t := range append([]string{title}, alts...) {
		t = strings.TrimSpace(t)
		n := textnorm.Normalize(t)
		if t == "" || n == "" || seen[t] {
			continue
		}
		seen[t] = true
		if _, err := tx.ExecContext(ctx, "INSERT INTO alt_titles(series_id, pos, title, normalized) VALUES(?,?,?,?)",
			seriesID, len(titles), t, n); err != nil {
			return nil, err
		}
		titles = append(titles, t)
	}
	return titles, nil
}

func (s *Store) altTitles(ctx context.Context, seriesID int64) ([]string, error) {
	q := "SELECT series_id, title FROM alt_titles"
	args := []any{}
	if seriesID != 0 {
		q += " WHERE series_id = ?"
		args = append(args, seriesID)
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY series_id, pos", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id int64
		var t string
		if err := rows.Scan(&id, &t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) allAltTitles(ctx context.Context) (map[int64][]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT series_id, title FROM alt_titles ORDER BY series_id, pos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]string{}
	for rows.Next() {
		var id int64
		var t string
		if err := rows.Scan(&id, &t); err != nil {
			return nil, err
		}
		out[id] = append(out[id], t)
	}
	return out, rows.Err()
}

func (s *Store) SeriesByMUID(ctx context.Context, muID int64) (Series, bool, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+seriesCols+" FROM series s WHERE s.mu_id = ?", muID)
	it, err := scanItem(row, false)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, false, nil
	}
	if err != nil {
		return Series{}, false, err
	}
	it.Series.AltTitles, err = s.altTitles(ctx, it.Series.ID)
	return it.Series, err == nil, err
}

func (s *Store) GetItem(ctx context.Context, seriesID int64) (Item, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+seriesCols+", "+entryCols+
		" FROM series s JOIN entry e ON e.series_id = s.id WHERE s.id = ?", seriesID)
	it, err := scanItem(row, true)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, ErrNotFound
	}
	if err != nil {
		return Item{}, err
	}
	it.Series.AltTitles, err = s.altTitles(ctx, seriesID)
	return it, err
}

func (s *Store) ListItems(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+seriesCols+", "+entryCols+
		" FROM series s JOIN entry e ON e.series_id = s.id ORDER BY s.title")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		it, err := scanItem(rows, true)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	alts, err := s.allAltTitles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Series.AltTitles = alts[items[i].Series.ID]
	}
	return items, nil
}

// SaveEntry inserts or replaces the owner's entry for e.SeriesID.
func (s *Store) SaveEntry(ctx context.Context, e Entry) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO entry (series_id, status, my_chapter, my_rating, notes, read_url,
		last_read_at, added_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(series_id) DO UPDATE SET status=excluded.status, my_chapter=excluded.my_chapter,
		my_rating=excluded.my_rating, notes=excluded.notes, read_url=excluded.read_url,
		last_read_at=excluded.last_read_at, added_at=excluded.added_at, updated_at=excluded.updated_at`,
		e.SeriesID, e.Status, e.MyChapter, e.MyRating, e.Notes, e.ReadURL,
		fmtTime(e.LastReadAt), fmtTime(e.AddedAt), fmtTime(e.UpdatedAt))
	return err
}

func (s *Store) DeleteSeries(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM series WHERE id = ?", id)
	return err
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// SearchLocal finds tracked series whose title or any alternate title contains
// query, ignoring case, accents and punctuation.
func (s *Store) SearchLocal(ctx context.Context, query string, limit int) ([]Item, error) {
	q := textnorm.Normalize(query)
	if q == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT s.id FROM series s JOIN entry e ON e.series_id = s.id
		WHERE s.id IN (SELECT series_id FROM alt_titles WHERE normalized LIKE ? ESCAPE '\')
		ORDER BY s.title LIMIT ?`, "%"+escapeLike(q)+"%", limit)
	if err != nil {
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	items := make([]Item, 0, len(ids))
	for _, id := range ids {
		it, err := s.GetItem(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

// SetLatest records a freshly fetched latest release. A manual override is
// cleared once the fetched chapter passes it.
func (s *Store) SetLatest(ctx context.Context, id int64, l Latest, checkedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE series SET latest_chapter = ?, latest_raw = ?, latest_group = ?,
		latest_date = ?, chapters_checked_at = ?,
		latest_manual = CASE WHEN latest_manual IS NOT NULL AND ? > latest_manual THEN NULL ELSE latest_manual END
		WHERE id = ?`, l.Chapter, l.Raw, l.Group, l.Date, fmtTime(checkedAt), l.Chapter, id)
	return err
}

func (s *Store) SetLatestManual(ctx context.Context, id int64, v *float64) error {
	_, err := s.db.ExecContext(ctx, "UPDATE series SET latest_manual = ? WHERE id = ?", nullable(v), id)
	return err
}
