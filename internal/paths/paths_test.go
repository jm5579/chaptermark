package paths

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirFor(t *testing.T) {
	got := DataDirFor(`C:\x\ChapterMark.exe`)
	if want := `C:\x\data`; got != want {
		t.Fatalf("DataDirFor = %q, want %q", got, want)
	}
}

func TestEnsureCreatesSubdirs(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := Ensure(dataDir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	for _, sub := range []string{"covers", "backups"} {
		info, err := os.Stat(filepath.Join(dataDir, sub))
		if err != nil || !info.IsDir() {
			t.Fatalf("%s not created: %v", sub, err)
		}
	}
	entries, _ := os.ReadDir(dataDir)
	if len(entries) != 2 {
		t.Fatalf("probe file left behind: %v", entries)
	}
}

func TestEnsureNotWritable(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Ensure(filepath.Join(blocker, "data"))
	if !errors.Is(err, ErrNotWritable) {
		t.Fatalf("Ensure error = %v, want ErrNotWritable", err)
	}
}
