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
	for _, bad := range []Settings{
		{CheckIntervalHours: 0, BackupsKeep: 10}, {CheckIntervalHours: 169, BackupsKeep: 10},
		{CheckIntervalHours: 6, BackupsKeep: 0}, {CheckIntervalHours: 6, BackupsKeep: 200},
	} {
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

func TestShowAdultIsOffUntilAllowed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	a := readyApp(t, dir)
	if a.GetSettings().ShowAdult {
		t.Fatal("18+ titles must be hidden by default")
	}
	if err := a.SaveSettings(Settings{CheckIntervalHours: 6, BackupsKeep: 10, ShowAdult: true}); err != nil {
		t.Fatal(err)
	}
	a.close()

	b := readyApp(t, dir)
	if !b.GetSettings().ShowAdult {
		t.Fatal("allowing 18+ titles should be remembered")
	}
	if err := b.SaveSettings(Settings{CheckIntervalHours: 6, BackupsKeep: 10}); err != nil {
		t.Fatal(err)
	}
	b.close()
	if readyApp(t, dir).GetSettings().ShowAdult {
		t.Fatal("turning 18+ titles off again should be remembered")
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

func TestViewPrefsPersist(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	a := readyApp(t, dir)
	if got := a.GetViewPrefs(); got != (ViewPrefs{Sort: "smart"}) {
		t.Fatalf("defaults = %+v", got)
	}
	want := ViewPrefs{Sort: "rating", StaleOnly: true, Tag: "Favourites"}
	if err := a.SaveViewPrefs(want); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []ViewPrefs{{Sort: "nonsense"}, {Sort: "title", Tag: strings.Repeat("x", 41)}} {
		if err := a.SaveViewPrefs(bad); err == nil {
			t.Errorf("SaveViewPrefs(%+v) should fail", bad)
		}
	}
	a.close()
	if got := readyApp(t, dir).GetViewPrefs(); got != want {
		t.Fatalf("after reopening = %+v, want %+v", got, want)
	}
}

func TestFitWindowSize(t *testing.T) {
	cases := []struct{ w, h, sw, sh, wantW, wantH int }{
		{2400, 1350, 1366, 768, 1326, 688}, // saved on a big monitor, reopened on a laptop
		{1280, 820, 1920, 1080, 1280, 820}, // fits: unchanged
		{1280, 820, 0, 0, 1280, 820},       // screen size unknown: unchanged
	}
	for _, c := range cases {
		if w, h := fitWindowSize(c.w, c.h, c.sw, c.sh); w != c.wantW || h != c.wantH {
			t.Errorf("fitWindowSize(%d,%d on %dx%d) = %d,%d want %d,%d", c.w, c.h, c.sw, c.sh, w, h, c.wantW, c.wantH)
		}
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
