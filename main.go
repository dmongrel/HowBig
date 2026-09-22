// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

//go:build !fyne_legacy

package main

import (
	"embed"
	"fmt"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// assets holds the built frontend. Run `npm run build` in frontend (or
// `wails3 build`) before building or testing so frontend/dist exists.
//
//go:embed all:frontend/dist
var assets embed.FS

// main opens the application window.
func main() {
	app := application.New(application.Options{
		Name:        "HowBig",
		Description: "Compares the sizes of two countries",
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

	window.OnWindowEvent(events.Common.WindowDidResize, func(*application.WindowEvent) {
		w, h := window.Size()
		window.SetTitle(fmt.Sprintf("HowBig %d x %d", w, h))
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
