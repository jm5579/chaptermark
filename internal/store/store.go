// Package store keeps the owner's library in a single SQLite file.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type SeriesType string

const (
	TypeManhwa SeriesType = "manhwa"
	TypeManga  SeriesType = "manga"
	TypeManhua SeriesType = "manhua"
	TypeOther  SeriesType = "other"
)

type Status string

const (
	StatusReading   Status = "reading"
	StatusPlan      Status = "plan"
	StatusCompleted Status = "completed"
	StatusOnHold    Status = "on_hold"
	StatusDropped   Status = "dropped"
)

// Series is public metadata about a title, mostly from MangaUpdates.
type Series struct {
	ID                 int64      `json:"id"`
	MUID               int64      `json:"muId"`
	MDID               string     `json:"mdId"`
	ALID               int        `json:"alId"`
	MALID              int        `json:"malId"`
	Title              string     `json:"title"`
	MUURL              string     `json:"muUrl"`
	Year               string     `json:"year"`
	PubStatus          string     `json:"pubStatus"`
	Type               SeriesType `json:"type"`
	TypeOverride       SeriesType `json:"typeOverride"`
	Completed          bool       `json:"completed"`
	CoverFile          string     `json:"coverFile"`
	CoverURL           string     `json:"coverUrl"`
	MURating           float64    `json:"muRating"`
	MUVotes            int        `json:"muVotes"`
	ALRating           float64    `json:"alRating"`
	LatestChapter      float64    `json:"latestChapter"`
	LatestRaw          string     `json:"latestRaw"`
	LatestGroup        string     `json:"latestGroup"`
	LatestDate         string     `json:"latestDate"`
	LatestManual       *float64   `json:"latestManual"`
	DetailsRefreshedAt time.Time  `json:"detailsRefreshedAt"`
	ChaptersCheckedAt  time.Time  `json:"chaptersCheckedAt"`
	AltTitles          []string   `json:"altTitles"`
}

// Entry is the owner's own tracking for a series.
type Entry struct {
	SeriesID   int64     `json:"seriesId"`
	Status     Status    `json:"status"`
	MyChapter  float64   `json:"myChapter"`
	MyRating   int       `json:"myRating"`
	Notes      string    `json:"notes"`
	ReadURL    string    `json:"readUrl"`
	LastReadAt time.Time `json:"lastReadAt"`
	AddedAt    time.Time `json:"addedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Item struct {
	Series Series `json:"series"`
	Entry  Entry  `json:"entry"`
}

// Latest is the newest release found for a series.
type Latest struct {
	Chapter float64
	Raw     string
	Group   string
	Date    string
}

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

// Open opens (creating if needed) the database at path. Call Migrate before use.
func Open(path string) (*Store, error) {
	// The path goes into a SQLite URI, where '#' and '?' end the file name and
	// '%' starts an escape; escape them so folders like "Manga #1" still work.
	uriPath := strings.NewReplacer("%", "%25", "?", "%3F", "#", "%23").Replace(path)
	dsn := "file:" + uriPath + "?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB  { return s.db }
func (s *Store) Close() error { return s.db.Close() }

var migrations = []string{
	`CREATE TABLE series (
		id INTEGER PRIMARY KEY,
		mu_id INTEGER NOT NULL UNIQUE,
		md_id TEXT NOT NULL DEFAULT '',
		al_id INTEGER NOT NULL DEFAULT 0,
		mal_id INTEGER NOT NULL DEFAULT 0,
		title TEXT NOT NULL,
		mu_url TEXT NOT NULL DEFAULT '',
		year TEXT NOT NULL DEFAULT '',
		pub_status TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL DEFAULT 'other',
		type_override TEXT NOT NULL DEFAULT '',
		completed INTEGER NOT NULL DEFAULT 0,
		cover_file TEXT NOT NULL DEFAULT '',
		cover_url TEXT NOT NULL DEFAULT '',
		mu_rating REAL NOT NULL DEFAULT 0,
		mu_votes INTEGER NOT NULL DEFAULT 0,
		al_rating REAL NOT NULL DEFAULT 0,
		latest_chapter REAL NOT NULL DEFAULT 0,
		latest_raw TEXT NOT NULL DEFAULT '',
		latest_group TEXT NOT NULL DEFAULT '',
		latest_date TEXT NOT NULL DEFAULT '',
		latest_manual REAL,
		details_refreshed_at TEXT NOT NULL DEFAULT '',
		chapters_checked_at TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE alt_titles (
		series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
		pos INTEGER NOT NULL,
		title TEXT NOT NULL,
		normalized TEXT NOT NULL,
		PRIMARY KEY (series_id, title)
	);
	CREATE INDEX alt_titles_normalized ON alt_titles(normalized);
	CREATE TABLE entry (
		series_id INTEGER PRIMARY KEY REFERENCES series(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		my_chapter REAL NOT NULL DEFAULT 0,
		my_rating INTEGER NOT NULL DEFAULT 0,
		notes TEXT NOT NULL DEFAULT '',
		read_url TEXT NOT NULL DEFAULT '',
		last_read_at TEXT NOT NULL DEFAULT '',
		added_at TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL DEFAULT ''
	);
	CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);`,
}

// SchemaVersion is the version Migrate brings the database to.
func SchemaVersion() int { return len(migrations) }

// Version returns the database's current schema version.
func (s *Store) Version(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&v)
	return v, err
}

// Migrate applies any pending schema migrations, each in its own transaction.
func (s *Store) Migrate(ctx context.Context) error {
	v, err := s.Version(ctx)
	if err != nil {
		return err
	}
	for i := v; i < len(migrations); i++ {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GetMeta(ctx context.Context, key string) (string, bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, "SELECT value FROM meta WHERE key = ?", key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return v, err == nil, err
}

func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO meta(key, value) VALUES(?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
}
