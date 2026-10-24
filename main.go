package main

import (
	"embed"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/jm5579/chaptermark/internal/covers"
	"github.com/jm5579/chaptermark/internal/paths"
)

//go:embed all:frontend/dist
var assets embed.FS

// version is set at release time with -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	dataDir, err := paths.ExeDataDir()
	app := NewApp(dataDir, err)

	var coverMiddleware assetserver.Middleware
	webviewDir := "" // empty = WebView2's default (under %APPDATA%)
	if err == nil {
		coverMiddleware = covers.Middleware(filepath.Join(dataDir, "covers"))
		// Keep the embedded browser's cache inside the app folder too, so the
		// app leaves nothing behind on the PC. If the folder isn't writable,
		// stay on the default so the "can't save here" message can still show.
		if paths.Ensure(dataDir) == nil {
			webviewDir = filepath.Join(dataDir, "webview")
		}
	}

	err = wails.Run(&options.App{
		Title:     "ChapterMark",
		Width:     1280,
		Height:    820,
		MinWidth:  1000,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets:     assets,
			Middleware: coverMiddleware,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 18, B: 24, A: 1},
		// Shown by domReady once the saved window size is applied (no resize flash).
		StartHidden:   true,
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme:               windows.Dark,
			WebviewUserDataPath: webviewDir,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
