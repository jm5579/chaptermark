package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readyApp(t *testing.T, dataDir string) *App {
	t.Helper()
	a := NewApp(dataDir, nil)
	if err := a.setup(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	t.Cleanup(a.close)
	return a
}

func TestSaveSettingsValidates(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	a := readyApp(t, dir)
	if got := a.GetSettings(); got != (Settings{CheckIntervalHours: 6, BackupsKeep: 10}) {
		t.Fatalf("defaults = %+v", got)
	}
	for _, bad := range []Settings{{0, 10}, {169, 10}, {6, 0}, {6, 200}} {
		if err := a.SaveSettings(bad); err == nil {
			t.Errorf("SaveSettings(%+v) should fail", bad)
		}
	}
	if err := a.SaveSettings(Settings{CheckIntervalHours: 12, BackupsKeep: 5}); err != nil {
		t.Fatal(err)
	}
	a.close()

	b := readyApp(t, dir)
	if got := b.GetSettings(); got != (Settings{CheckIntervalHours: 12, BackupsKeep: 5}) {
		t.Fatalf("persisted settings = %+v", got)
	}
}

func TestInitReportsUnwritableFolder(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	os.WriteFile(blocker, []byte("x"), 0o644)
	a := NewApp(filepath.Join(blocker, "data"), nil)
	a.initErr = a.setup()
	res := a.Init()
	if res.Ready || !strings.Contains(res.Error, "Move the ChapterMark folder") {
		t.Fatalf("Init = %+v", res)
	}
	if _, err := a.ListLibrary(); err == nil {
		t.Fatal("ListLibrary should fail when not ready")
	}
}

func TestSetupBacksUpExistingLibrary(t *testing.T) {
	// A folder name with URI-special characters must still keep everything inside it.
	dir := filepath.Join(t.TempDir(), "Manga #1 100%25", "data")
	a := readyApp(t, dir)
	a.close()
	readyApp(t, dir)
	entries, _ := os.ReadDir(filepath.Join(dir, "backups"))
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "library-") {
		t.Fatalf("backups = %v", entries)
	}
}
