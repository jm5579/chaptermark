package main

import (
	"context"
)

// App is the Wails-bound backend. Its exported methods are callable from the UI.
type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

// startup is called by Wails when the window is created.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
