// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package settings loads settings.json and resolves the data paths it names.
package settings

import (
	"cmp"
	"encoding/json"
	"fmt"
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

// Default returns the settings used when settings.json is missing or invalid,
// and for any setting a valid file leaves out.
func Default() Settings {
	return Settings{
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

// Load reads settings from the JSON file at path. Settings the file leaves
// out take their Default values, and so do colors, paths and font sizes set
// to "" or 0. If the file can't be read or isn't valid JSON, Load returns
// Default() and the error; a missing file's error satisfies
// errors.Is(err, fs.ErrNotExist).
func Load(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Default(), err
	}
	s := Default()
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), fmt.Errorf("parsing %s: %w", path, err)
	}
	s.fillZeroValues()
	return s, nil
}

// fillZeroValues replaces empty colors and paths and zero font sizes with
// their defaults.
func (s *Settings) fillZeroValues() {
	d := Default()
	s.LeftColor = cmp.Or(s.LeftColor, d.LeftColor)
	s.RightColor = cmp.Or(s.RightColor, d.RightColor)
	s.LeftBorderColor = cmp.Or(s.LeftBorderColor, d.LeftBorderColor)
	s.RightBorderColor = cmp.Or(s.RightBorderColor, d.RightBorderColor)
	s.BackgroundColor = cmp.Or(s.BackgroundColor, d.BackgroundColor)
	s.MapDataPath = cmp.Or(s.MapDataPath, d.MapDataPath)
	s.CountryDataPath = cmp.Or(s.CountryDataPath, d.CountryDataPath)
	s.ButtonFontSize = cmp.Or(s.ButtonFontSize, d.ButtonFontSize)
	s.SearchFontSize = cmp.Or(s.SearchFontSize, d.SearchFontSize)
	s.CountryListFontSize = cmp.Or(s.CountryListFontSize, d.CountryListFontSize)
	s.HeaderFontSize = cmp.Or(s.HeaderFontSize, d.HeaderFontSize)
}

// ResolvePath locates a relative data path (settings.json, map_data_path,
// country_data_path). It tries the executable's directory, then the working
// directory, then the executable directory's parent, and returns the first
// candidate that exists. An absolute path, or one found nowhere, is returned
// unchanged so the caller's error names the configured path.
func ResolvePath(p string) string {
	exe, err := os.Executable()
	if err != nil {
		return resolvePathFrom(p, "")
	}
	return resolvePathFrom(p, filepath.Dir(exe))
}

// resolvePathFrom is ResolvePath with the executable directory given.
// An empty exeDir skips the two executable-relative candidates.
func resolvePathFrom(p, exeDir string) string {
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
