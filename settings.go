// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// Settings define the UI configuration for the application.
type Settings struct {
	DebugShowBoundary   bool    `json:"debug_show_boundary"`    // DebugShowBoundary determines if the bounding box is rendered.
	LeftColor           string  `json:"left_color"`             // LeftColor is the hex color for the left map.
	RightColor          string  `json:"right_color"`            // RightColor is the hex color for the right map.
	LeftBorderColor     string  `json:"left_border_color"`      // LeftBorderColor is the hex color for the left country border.
	RightBorderColor    string  `json:"right_border_color"`     // RightBorderColor is the hex color for the right country border.
	BackgroundColor     string  `json:"background_color"`       // BackgroundColor is the background color for the application.
	EnablePacificCenter bool    `json:"enable_pacific_center"`  // EnablePacificCenter determines if the map is centered on the Pacific Ocean for countries spanning the 180-degree meridian.
	SkipSmall           int     `json:"skip_small"`             // SkipSmall determines if polygons with few points are skipped.
	MapDataPath         string  `json:"map_data_path"`          // MapDataPath is the path to the directory containing GeoJSON files.
	CountryDataPath     string  `json:"country_data_path"`      // CountryDataPath is the path to the country_data.json file.
	ButtonFontSize      float32 `json:"button_font_size"`       // ButtonFontSize is the font size for buttons.
	SearchFontSize      float32 `json:"search_font_size"`       // SearchFontSize is the font size for search bars.
	CountryListFontSize float32 `json:"country_list_font_size"` // CountryListFontSize is the font size for country lists.
	HeaderFontSize      float32 `json:"header_font_size"`       // HeaderFontSize is the font size for the header.
}

// loadSettings reads application configuration from settings.json, applying default values if the file is missing or invalid.
func loadSettings(path string) *Settings {
	var s Settings
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println("Error reading settings.json, using defaults:", err)
		s = Settings{
			DebugShowBoundary:   false,
			LeftColor:           "#00FF00",
			RightColor:          "#FF0000",
			LeftBorderColor:     "#00FFFF",
			RightBorderColor:    "#FFCC00",
			BackgroundColor:     "#000000",
			EnablePacificCenter: true,
			SkipSmall:           0,
			MapDataPath:         "mapdata",
			CountryDataPath:     "country_data.json",
			ButtonFontSize:      14,
			SearchFontSize:      14,
			CountryListFontSize: 18,
			HeaderFontSize:      36,
		}
	} else if err := json.Unmarshal(data, &s); err != nil {
		log.Println("Error unmarshaling settings.json:", err)
		s = Settings{
			DebugShowBoundary:   false,
			LeftColor:           "#00FF00",
			RightColor:          "#FF0000",
			LeftBorderColor:     "#00FFFF",
			RightBorderColor:    "#FFCC00",
			BackgroundColor:     "#000000",
			EnablePacificCenter: true,
			SkipSmall:           0,
			MapDataPath:         "mapdata",
			CountryDataPath:     "country_data.json",
			ButtonFontSize:      14,
			SearchFontSize:      14,
			CountryListFontSize: 18,
			HeaderFontSize:      36,
		}
	}
	if s.LeftColor == "" {
		s.LeftColor = "#00FF00"
	}
	if s.RightColor == "" {
		s.RightColor = "#FF0000"
	}
	if s.LeftBorderColor == "" {
		s.LeftBorderColor = "#00FFFF"
	}
	if s.RightBorderColor == "" {
		s.RightBorderColor = "#FFCC00"
	}
	if s.BackgroundColor == "" {
		s.BackgroundColor = "#000000"
	}
	if s.MapDataPath == "" {
		s.MapDataPath = "mapdata"
	}
	if s.CountryDataPath == "" {
		s.CountryDataPath = "country_data.json"
	}
	if s.ButtonFontSize == 0 {
		s.ButtonFontSize = 14
	}
	if s.SearchFontSize == 0 {
		s.SearchFontSize = 14
	}
	if s.CountryListFontSize == 0 {
		s.CountryListFontSize = 18
	}
	if s.HeaderFontSize == 0 {
		s.HeaderFontSize = 36
	}
	return &s
}

// resolveDataPath locates a relative data path (settings.json, map_data_path,
// country_data_path). It tries the executable's directory, then the working
// directory, then the executable directory's parent, and returns the first
// candidate that exists. An absolute path, or one found nowhere, is returned
// unchanged so the caller's error names the configured path.
func resolveDataPath(p string) string {
	exe, err := os.Executable()
	if err != nil {
		return resolveDataPathFrom(p, "")
	}
	return resolveDataPathFrom(p, filepath.Dir(exe))
}

// resolveDataPathFrom is resolveDataPath with the executable directory given.
// An empty exeDir skips the two executable-relative candidates.
func resolveDataPathFrom(p, exeDir string) string {
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	var candidates []string
	if exeDir != "" {
		candidates = append(candidates, filepath.Join(exeDir, p))
	}
	candidates = append(candidates, p)
	if exeDir != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(exeDir), p))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return p
}
