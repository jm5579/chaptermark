// Package paths resolves where ChapterMark keeps its data: a "data" folder
// next to the exe, so copying the app folder carries the library with it.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNotWritable = errors.New("ChapterMark can't save here. Move the ChapterMark folder somewhere you can write to (e.g. Desktop or Documents) and run it again.")

// DataDirFor returns the data directory for an exe at exePath.
func DataDirFor(exePath string) string {
	return filepath.Join(filepath.Dir(exePath), "data")
}

// ExeDataDir returns the data directory next to the running executable.
func ExeDataDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return DataDirFor(exe), nil
}

// Ensure creates dataDir with its covers/ and backups/ subfolders and checks
// that files can be written there.
func Ensure(dataDir string) error {
	for _, dir := range []string{dataDir, filepath.Join(dataDir, "covers"), filepath.Join(dataDir, "backups")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("%w (%v)", ErrNotWritable, err)
		}
	}
	probe := filepath.Join(dataDir, ".write-test")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		return fmt.Errorf("%w (%v)", ErrNotWritable, err)
	}
	return os.Remove(probe)
}
