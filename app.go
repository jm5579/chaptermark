package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/jm5579/chaptermark/internal/backup"
	"github.com/jm5579/chaptermark/internal/checker"
	"github.com/jm5579/chaptermark/internal/covers"
	"github.com/jm5579/chaptermark/internal/exportimport"
	"github.com/jm5579/chaptermark/internal/httpx"
	"github.com/jm5579/chaptermark/internal/library"
	"github.com/jm5579/chaptermark/internal/paths"
	"github.com/jm5579/chaptermark/internal/sources/anilist"
	"github.com/jm5579/chaptermark/internal/sources/mangadex"
	"github.com/jm5579/chaptermark/internal/sources/mangaupdates"
	"github.com/jm5579/chaptermark/internal/store"
)

const (
	metaInterval = "check_interval_hours"
	metaBackups  = "backups_keep"
	metaWindow   = "window"
)

var errNotReady = errors.New("ChapterMark couldn't open its data folder")

type Settings struct {
	CheckIntervalHours int `json:"checkIntervalHours"`
	BackupsKeep        int `json:"backupsKeep"`
}

var defaultSettings = Settings{CheckIntervalHours: 6, BackupsKeep: 10}

func (s Settings) validate() error {
	if s.CheckIntervalHours < 1 || s.CheckIntervalHours > 168 {
		return errors.New("check interval must be between 1 and 168 hours")
	}
	if s.BackupsKeep < 1 || s.BackupsKeep > 50 {
		return errors.New("backups to keep must be between 1 and 50")
	}
	return nil
}

type InitResult struct {
	Ready   bool   `json:"ready"`
	Error   string `json:"error"`
	DataDir string `json:"dataDir"`
	Version string `json:"version"`
}

// App is the Wails-bound backend. Its exported methods are callable from the UI.
type App struct {
	ctx     context.Context
	dataDir string
	initErr error

	store   *store.Store
	lib     *library.Service
	checker *checker.Checker

	mu       sync.Mutex
	settings Settings
}

func NewApp(dataDir string, dataDirErr error) *App {
	return &App{dataDir: dataDir, initErr: dataDirErr, settings: defaultSettings}
}

// startup is called by Wails when the window is created.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.initErr == nil {
		a.initErr = a.setup()
	}
	if a.initErr != nil {
		log.Printf("startup: %v", a.initErr)
		return
	}
	if due, _ := a.checker.Due(ctx); due {
		go a.runCheck()
	}
	// While the window stays open, look again every 15 minutes whether the
	// configured interval has passed. ctx ends when the app shuts down.
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		defer ticker.Stop()
		a.checker.Watch(ctx, ticker.C, a.runCheck)
	}()
}

// domReady restores the saved window size once the window exists.
func (a *App) domReady(ctx context.Context) {
	if a.store == nil {
		return
	}
	v, ok, err := a.store.GetMeta(ctx, metaWindow)
	if err != nil || !ok {
		return
	}
	var w struct {
		Width, Height int
		Maximised     bool
	}
	if json.Unmarshal([]byte(v), &w) != nil {
		return
	}
	if w.Width >= 1000 && w.Height >= 680 {
		width, height := w.Width, w.Height
		if screens, err := runtime.ScreenGetAll(ctx); err == nil {
			for _, s := range screens {
				if s.IsCurrent {
					width, height = fitWindowSize(width, height, s.Size.Width, s.Size.Height)
				}
			}
		}
		runtime.WindowSetSize(ctx, width, height)
		runtime.WindowCenter(ctx)
	}
	if w.Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// fitWindowSize shrinks a saved window size to fit a screen of sw×sh (the
// folder may have been copied from a PC with a bigger monitor), leaving room
// for the taskbar and title bar. An unknown screen size (0) changes nothing.
func fitWindowSize(w, h, sw, sh int) (int, int) {
	if sw > 0 && w > sw-40 {
		w = sw - 40
	}
	if sh > 0 && h > sh-80 {
		h = sh - 80
	}
	return w, h
}

// beforeClose saves the window size; returning false lets the window close.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.store != nil {
		w, h := runtime.WindowGetSize(ctx)
		b, _ := json.Marshal(struct {
			Width, Height int
			Maximised     bool
		}{w, h, runtime.WindowIsMaximised(ctx)})
		a.store.SetMeta(ctx, metaWindow, string(b))
	}
	return false
}

func (a *App) shutdown(ctx context.Context) { a.close() }

func (a *App) close() {
	if a.store != nil {
		a.store.Close()
		a.store = nil
	}
}

// setup opens the data folder: checks it is writable, backs up yesterday's
// library, migrates the schema, and builds the services.
func (a *App) setup() error {
	if err := paths.Ensure(a.dataDir); err != nil {
		return err
	}
	st, err := store.Open(filepath.Join(a.dataDir, "library.db"))
	if err != nil {
		return err
	}
	ctx := context.Background()
	v, err := st.Version(ctx)
	if err != nil {
		st.Close()
		return err
	}
	a.store = st
	if v > 0 {
		a.loadSettings(ctx)
		if _, err := backup.Daily(st.DB(), filepath.Join(a.dataDir, "backups"), a.GetSettings().BackupsKeep, time.Now()); err != nil {
			log.Printf("backup: %v", err)
		}
	}
	if err := st.Migrate(ctx); err != nil {
		return fmt.Errorf("upgrading your library failed (a backup is in data/backups): %w", err)
	}
	a.loadSettings(ctx)

	coversDir := filepath.Join(a.dataDir, "covers")
	coverHTTP := httpx.New(250 * time.Millisecond)
	a.lib = &library.Service{
		Store: st,
		MU:    &mangaupdates.Client{HTTP: httpx.New(time.Second)},
		MD:    &mangadex.Client{HTTP: httpx.New(250 * time.Millisecond)},
		AL:    &anilist.Client{HTTP: httpx.New(2 * time.Second)},
		FetchCover: func(ctx context.Context, url string, id int64) (string, error) {
			return covers.Download(ctx, coverHTTP, url, coversDir, id)
		},
		CoversDir: coversDir,
	}
	a.checker = &checker.Checker{
		Lib:   a.lib,
		Store: st,
		Interval: func() time.Duration {
			return time.Duration(a.GetSettings().CheckIntervalHours) * time.Hour
		},
	}
	return nil
}

func (a *App) loadSettings(ctx context.Context) {
	s := defaultSettings
	if v, ok, _ := a.store.GetMeta(ctx, metaInterval); ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.CheckIntervalHours = n
		}
	}
	if v, ok, _ := a.store.GetMeta(ctx, metaBackups); ok {
		if n, err := strconv.Atoi(v); err == nil {
			s.BackupsKeep = n
		}
	}
	if s.validate() != nil {
		s = defaultSettings
	}
	a.mu.Lock()
	a.settings = s
	a.mu.Unlock()
}

func (a *App) ready() error {
	if a.initErr != nil || a.lib == nil {
		return errNotReady
	}
	return nil
}

func (a *App) emit(name string, data any) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, name, data)
	}
}

func (a *App) runCheck() {
	err := a.checker.Run(context.Background(),
		func(p checker.Progress) { a.emit("check:progress", p) },
		func(v library.ItemView) { a.emit("library:item", v) })
	if err != nil {
		log.Printf("check: %v", err)
	}
}

// ---- Methods called from the UI ----

func (a *App) Init() InitResult {
	res := InitResult{Ready: a.ready() == nil, DataDir: a.dataDir, Version: version}
	if a.initErr != nil {
		res.Error = a.initErr.Error()
	}
	return res
}

func (a *App) ListLibrary() ([]library.ItemView, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.lib.List(context.Background())
}

func (a *App) SearchLocal(q string) ([]library.ItemView, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.lib.SearchLocal(context.Background(), q)
}

func (a *App) SearchOnline(q string, page int, includeOther bool) ([]library.OnlineResult, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	res, err := a.lib.SearchOnline(context.Background(), q, page, includeOther)
	if errors.Is(err, httpx.ErrOffline) {
		return nil, errors.New("couldn't reach MangaUpdates — check your internet connection")
	}
	return res, err
}

func (a *App) AddSeries(muID int64, in library.AddInput) (library.ItemView, error) {
	if err := a.ready(); err != nil {
		return library.ItemView{}, err
	}
	v, err := a.lib.Add(context.Background(), muID, in)
	if errors.Is(err, httpx.ErrOffline) {
		return v, errors.New("couldn't reach MangaUpdates — check your internet connection")
	}
	return v, err
}

func (a *App) UpdateEntry(seriesID int64, in library.EntryUpdate) (library.ItemView, error) {
	if err := a.ready(); err != nil {
		return library.ItemView{}, err
	}
	return a.lib.UpdateEntry(context.Background(), seriesID, in)
}

func (a *App) RemoveSeries(seriesID int64) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.lib.Remove(context.Background(), seriesID)
}

// CheckNow starts a background check for new chapters (no-op if one is running).
func (a *App) CheckNow() {
	if a.ready() == nil {
		go a.runCheck()
	}
}

func (a *App) CheckStatus() checker.Progress {
	if a.ready() != nil {
		return checker.Progress{}
	}
	return a.checker.Status()
}

func (a *App) GetSettings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

func (a *App) SaveSettings(s Settings) error {
	if err := s.validate(); err != nil {
		return err
	}
	if a.store == nil {
		return errNotReady
	}
	ctx := context.Background()
	if err := a.store.SetMeta(ctx, metaInterval, strconv.Itoa(s.CheckIntervalHours)); err != nil {
		return err
	}
	if err := a.store.SetMeta(ctx, metaBackups, strconv.Itoa(s.BackupsKeep)); err != nil {
		return err
	}
	a.mu.Lock()
	a.settings = s
	a.mu.Unlock()
	return nil
}

var jsonFilter = []runtime.FileFilter{{DisplayName: "ChapterMark export (*.json)", Pattern: "*.json"}}

// ExportLibrary asks where to save and writes the export; "" means cancelled.
func (a *App) ExportLibrary() (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export your ChapterMark list",
		DefaultFilename: "chaptermark-export-" + time.Now().Format("2006-01-02") + ".json",
		Filters:         jsonFilter,
	})
	if err != nil || path == "" {
		return "", err
	}
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := exportimport.Export(context.Background(), a.store, f, time.Now()); err != nil {
		return "", err
	}
	return path, nil
}

// ImportLibrary asks for an export file and imports it ("merge" or "replace").
func (a *App) ImportLibrary(mode string) (exportimport.Result, error) {
	if err := a.ready(); err != nil {
		return exportimport.Result{}, err
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{Title: "Import a ChapterMark export", Filters: jsonFilter})
	if err != nil || path == "" {
		return exportimport.Result{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return exportimport.Result{}, err
	}
	defer f.Close()
	res, err := exportimport.Import(context.Background(), a.store, f, exportimport.Mode(mode))
	if err == nil {
		go a.runCheck() // fetch covers and latest chapters for imported series
	}
	return res, err
}

func (a *App) OpenDataFolder() error {
	return exec.Command("explorer", a.dataDir).Start()
}

// OpenURL opens an http(s) link in the default browser.
func (a *App) OpenURL(url string) {
	if a.ctx != nil && (strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://")) {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}
