package main

import (
	"embed"
	"net/http"
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

	var coverHandler http.Handler = http.NotFoundHandler()
	if err == nil {
		coverHandler = covers.Handler(filepath.Join(dataDir, "covers"))
	}

	err = wails.Run(&options.App{
		Title:     "ChapterMark",
		Width:     1280,
		Height:    820,
		MinWidth:  1000,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: coverHandler,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 18, B: 24, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			Theme: windows.Dark,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
