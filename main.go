// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"embed"
	"log/slog"
	"os"

	"HowBig/internal/country"
	"HowBig/internal/settings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// assets holds the built frontend. Run `npm run build` in frontend (or
// `wails3 build`) before building or testing so frontend/dist exists.
//
//go:embed all:frontend/dist
var assets embed.FS

// main loads settings and country data, binds the services and opens the window.
func main() {
	cfg, err := settings.Load(settings.ResolvePath("settings.json"))
	if err != nil {
		slog.Warn("could not load settings.json, using defaults", "err", err)
	}

	cc, err := country.Load(settings.ResolvePath(cfg.CountryDataPath))
	if err != nil {
		slog.Error("failed to load country data", "err", err)
	}
	windowService := &WindowService{}

	app := application.New(application.Options{
		Name:        "HowBig",
		Description: "Compares the sizes of two countries",
		Services: []application.Service{
			application.NewService(NewCountryService(cc, err)),
			application.NewService(NewMapService(&cfg, cc, settings.ResolvePath(cfg.MapDataPath))),
			application.NewService(NewSettingsService(&cfg)),
			application.NewService(windowService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "HowBig",
		Width:            1280,
		Height:           768,
		MinWidth:         1280,
		MinHeight:        768,
		StartState:       application.WindowStateFullscreen,
		BackgroundColour: application.NewRGB(0, 0, 0),
		URL:              "/",
	})
	windowService.window = window
	windowService.quit = app.Quit

	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		window.SetTitle(windowTitle(window.Size()))
	})

	if err := app.Run(); err != nil {
		slog.Error("application stopped with an error", "err", err)
		os.Exit(1)
	}
}
