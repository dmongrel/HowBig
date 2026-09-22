// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// fullscreenWindow is the part of *application.WebviewWindow that WindowService uses.
type fullscreenWindow interface {
	IsFullscreen() bool
	Fullscreen() application.Window
	UnFullscreen()
}

// WindowService lets the frontend toggle fullscreen and quit.
// Its window and quit fields are set in main once the window exists.
type WindowService struct {
	window fullscreenWindow
	quit   func()
}

// ToggleFullscreen switches the window between fullscreen and windowed and
// returns true if it is now fullscreen.
func (s *WindowService) ToggleFullscreen() bool {
	if s.window == nil {
		return false
	}
	if s.window.IsFullscreen() {
		s.window.UnFullscreen()
		return false
	}
	s.window.Fullscreen()
	return true
}

// Quit exits the application.
func (s *WindowService) Quit() {
	if s.quit != nil {
		s.quit()
	}
}

// windowTitle is the title shown after a resize, as in v1.0.0.
func windowTitle(width, height int) string {
	return fmt.Sprintf("HowBig %d x %d", width, height)
}
