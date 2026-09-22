// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package settings

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// writeSettings writes content to a settings file in a temp dir and returns its path.
func writeSettings(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDefault(t *testing.T) {
	want := Settings{
		LeftColor:           "#00FF00",
		RightColor:          "#FF0000",
		LeftBorderColor:     "#00FFFF",
		RightBorderColor:    "#FFCC00",
		BackgroundColor:     "#000000",
		EnablePacificCenter: true,
		MapDataPath:         "mapdata",
		CountryDataPath:     "country_data.json",
		ButtonFontSize:      14,
		SearchFontSize:      14,
		CountryListFontSize: 18,
		HeaderFontSize:      36,
	}
	if got := Default(); got != want {
		t.Errorf("Default() = %+v, want %+v", got, want)
	}
}

func TestLoadMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "settings.json"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want not-exist", err)
	}
	if got != Default() {
		t.Errorf("Load = %+v, want the defaults", got)
	}
}

func TestLoadMalformedJSON(t *testing.T) {
	// The valid prefix must not leak into the result.
	got, err := Load(writeSettings(t, `{"left_color": "#123456", "skip_small": `))
	if err == nil {
		t.Error("want an error")
	}
	if got != Default() {
		t.Errorf("Load = %+v, want the defaults", got)
	}
}

func TestLoadPartialJSON(t *testing.T) {
	got, err := Load(writeSettings(t, `{
		"left_color": "#123456",
		"skip_small": 7,
		"debug_show_boundary": true,
		"right_color": "",
		"header_font_size": 0
	}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.LeftColor = "#123456"
	want.SkipSmall = 7
	want.DebugShowBoundary = true
	// right_color "" and header_font_size 0 fall back to their defaults.
	if got != want {
		t.Errorf("Load = %+v, want %+v", got, want)
	}

	// An explicit false is kept; only a missing enable_pacific_center defaults to true.
	got, err = Load(writeSettings(t, `{"enable_pacific_center": false}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.EnablePacificCenter {
		t.Error("enable_pacific_center false was overridden")
	}
}

func TestLoadShippedFile(t *testing.T) {
	got, err := Load(filepath.Join("..", "..", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := Default()
	want.SkipSmall = 25
	want.HeaderFontSize = 18
	if got != want {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestResolvePathFrom(t *testing.T) {
	root := t.TempDir()
	exeDir := filepath.Join(root, "bin")
	cwd := t.TempDir()
	for _, dir := range []string{exeDir, cwd} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(exeDir, "both.json"))
	write(filepath.Join(cwd, "both.json"))
	write(filepath.Join(cwd, "cwd.json"))
	write(filepath.Join(root, "parent.json"))
	t.Chdir(cwd)

	tests := []struct {
		name, in, want string
	}{
		{"exe dir wins over cwd", "both.json", filepath.Join(exeDir, "both.json")},
		{"cwd", "cwd.json", "cwd.json"},
		{"exe dir's parent", "parent.json", filepath.Join(root, "parent.json")},
		{"not found anywhere", "none.json", "none.json"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvePathFrom(tt.in, exeDir); got != tt.want {
				t.Errorf("resolvePathFrom(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	abs := filepath.Join(root, "parent.json")
	if got := ResolvePath(abs); got != abs {
		t.Errorf("ResolvePath(abs) = %q, want it unchanged", got)
	}
}
