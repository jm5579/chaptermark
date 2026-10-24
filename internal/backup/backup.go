// Package backup keeps one copy of the library per day in data/backups.
package backup

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Daily writes library-YYYY-MM-DD.db into dir (using SQLite's VACUUM INTO, which
// is safe while the database is open) unless today's backup already exists,
// then deletes the oldest backups beyond keep.
func Daily(db *sql.DB, dir string, keep int, now time.Time) (bool, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	path := filepath.Join(dir, "library-"+now.Format("2006-01-02")+".db")
	created := false
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if _, err := db.Exec("VACUUM INTO ?", path); err != nil {
			return false, err
		}
		created = true
	} else if err != nil {
		return false, err
	}
	return created, prune(dir, keep)
}

func prune(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasPrefix(n, "library-") && strings.HasSuffix(n, ".db") {
			names = append(names, n)
		}
	}
	sort.Strings(names) // dates sort chronologically
	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}
