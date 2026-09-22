//go:build !ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// modifyOptionsForIOS is a no-op on non-iOS platforms
//
//lint:ignore U1000 Wails iOS scaffold; the non-iOS stub of app_options_ios.go, which nothing calls on Windows
func modifyOptionsForIOS(opts *application.Options) {
	// No modifications needed for non-iOS platforms
}