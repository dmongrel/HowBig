// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestCountryService(t *testing.T) {
	t.Run("loaded", func(t *testing.T) {
		cc, err := NewCountryCollection("country_data.json")
		if err != nil {
			t.Fatal(err)
		}
		s := NewCountryService(cc, nil)
		list := s.List()
		if len(list) != 200 {
			t.Errorf("List() has %d countries, want 200", len(list))
		}
		if list[0] != cc.Countries[0] {
			t.Errorf("List()[0] = %+v, want file order %+v", list[0], cc.Countries[0])
		}
		list[0].Name = "changed"
		if cc.Countries[0].Name == "changed" {
			t.Error("List() returned the collection's own slice")
		}
		if got := s.Area("Fiji"); got != cc.Areas["Fiji"] || got == 0 {
			t.Errorf("Area(Fiji) = %v, want %v", got, cc.Areas["Fiji"])
		}
		if got := s.Area("Atlantis"); got != 0 {
			t.Errorf("Area(Atlantis) = %v, want 0", got)
		}
		if got := s.LoadError(); got != "" {
			t.Errorf("LoadError() = %q, want empty", got)
		}
	})

	t.Run("failed to load", func(t *testing.T) {
		cc, err := NewCountryCollection(filepath.Join(t.TempDir(), "missing.json"))
		s := NewCountryService(cc, err)
		if list := s.List(); list == nil || len(list) != 0 {
			t.Errorf("List() = %#v, want an empty non-nil slice", list)
		}
		if got := s.Area("Fiji"); got != 0 {
			t.Errorf("Area(Fiji) = %v, want 0", got)
		}
		if got := s.LoadError(); !strings.HasPrefix(got, "failed to load country data: ") {
			t.Errorf("LoadError() = %q, want a load failure message", got)
		}
	})
}

func TestSettingsService(t *testing.T) {
	in := &Settings{LeftColor: "#123456", SkipSmall: 7}
	s := NewSettingsService(in)
	in.LeftColor = "#FFFFFF"
	if got := s.Get(); got.LeftColor != "#123456" || got.SkipSmall != 7 {
		t.Errorf("Get() = %+v, want the settings as they were passed in", got)
	}
}

// fakeWindow records fullscreen calls for WindowService tests.
type fakeWindow struct {
	full bool
}

func (w *fakeWindow) IsFullscreen() bool             { return w.full }
func (w *fakeWindow) Fullscreen() application.Window { w.full = true; return nil }
func (w *fakeWindow) UnFullscreen()                  { w.full = false }

func TestWindowService(t *testing.T) {
	w := &fakeWindow{full: true}
	quits := 0
	s := &WindowService{window: w, quit: func() { quits++ }}

	if got := s.ToggleFullscreen(); got || w.full {
		t.Errorf("first toggle = %v (window full %v), want windowed", got, w.full)
	}
	if got := s.ToggleFullscreen(); !got || !w.full {
		t.Errorf("second toggle = %v (window full %v), want fullscreen", got, w.full)
	}
	s.Quit()
	if quits != 1 {
		t.Errorf("Quit called quit %d times, want 1", quits)
	}

	var empty WindowService
	if empty.ToggleFullscreen() {
		t.Error("ToggleFullscreen with no window = true, want false")
	}
	empty.Quit() // must not panic

	if got := windowTitle(1280, 768); got != "HowBig 1280 x 768" {
		t.Errorf("windowTitle = %q", got)
	}
}

func TestResolveDataPathFrom(t *testing.T) {
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
			if got := resolveDataPathFrom(tt.in, exeDir); got != tt.want {
				t.Errorf("resolveDataPathFrom(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	abs := filepath.Join(root, "parent.json")
	if got := resolveDataPath(abs); got != abs {
		t.Errorf("resolveDataPath(abs) = %q, want it unchanged", got)
	}
}

// rectGeoJSON returns a one-feature GeoJSON document whose polygon is the
// lon/lat rectangle from (lon0, lat0) to (lon1, lat1).
func rectGeoJSON(lon0, lat0, lon1, lat1 float64) string {
	return fmt.Sprintf(`{"features":[{"geometry":{"type":"Polygon","coordinates":[[[%[1]v,%[2]v],[%[3]v,%[2]v],[%[3]v,%[4]v],[%[1]v,%[4]v],[%[1]v,%[2]v]]]}}]}`,
		lon0, lat0, lon1, lat1)
}

// newTestMapService writes the given ISO code -> GeoJSON documents to a temp
// map data directory and returns a MapService over countries named after the
// ISO codes with the given areas.
func newTestMapService(t *testing.T, settings Settings, docs map[string]string, areas map[string]float64) *MapService {
	t.Helper()
	dir := t.TempDir()
	cc := &CountryCollection{Areas: map[string]float64{}, ISOCodes: map[string]string{}}
	for iso, doc := range docs {
		if err := os.WriteFile(filepath.Join(dir, iso+".geojson"), []byte(doc), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, area := range areas {
		cc.Countries = append(cc.Countries, CountryInfo{Name: name, ISOCode: name, Area: area})
		cc.Areas[name] = area
		cc.ISOCodes[name] = name
	}
	return NewMapService(&settings, cc, dir)
}

// pixelBounds returns the extent of all points in paths.
func pixelBounds(paths [][]float64) (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, p := range paths {
		for i := 0; i < len(p); i += 2 {
			minX, maxX = min(minX, p[i]), max(maxX, p[i])
			minY, maxY = min(minY, p[i+1]), max(maxY, p[i+1])
		}
	}
	return minX, minY, maxX, maxY
}

// nearPx reports whether two pixel values agree to within rounding.
func nearPx(a, b float64) bool { return math.Abs(a-b) <= 0.15 }

func TestMapServiceLayout(t *testing.T) {
	const w, h = 800.0, 600.0
	docs := map[string]string{
		"BIG":  rectGeoJSON(0, 0, 20, 20),
		"SML":  rectGeoJSON(0, 0, 5, 5),
		"WIDE": rectGeoJSON(0, 0, 60, 2), // smaller by area than BIG, wider in pixels
		"BAD":  `not json`,
	}
	areas := map[string]float64{"BIG": 1000, "SML": 10, "WIDE": 100, "BAD": 50, "GONE": 5}
	s := newTestMapService(t, Settings{}, docs, areas)

	t.Run("nothing selected", func(t *testing.T) {
		got := s.Layout("", "", w, h)
		if got.Countries == nil || len(got.Countries) != 0 || got.Scale != 1 {
			t.Errorf("Layout = %+v, want scale 1 and no countries", got)
		}
	})

	t.Run("one selected is centered and fitted", func(t *testing.T) {
		got := s.Layout("", "BIG", w, h)
		if len(got.Countries) != 1 {
			t.Fatalf("got %d countries, want 1", len(got.Countries))
		}
		c := got.Countries[0]
		if c.Name != "BIG" || c.Side != SideRight || c.Order != 0 || c.Error != "" || c.Box != nil {
			t.Errorf("country = %+v", c)
		}
		minX, minY, maxX, maxY := pixelBounds(c.Paths)
		if !nearPx(minX+maxX, w) || !nearPx(minY+maxY, h) {
			t.Errorf("not centered: x %v..%v, y %v..%v in %vx%v", minX, maxX, minY, maxY, w, h)
		}
		// BIG is taller than wide in Mercator, so height is the limiting axis.
		if !nearPx(maxY-minY, h-fitMargin) {
			t.Errorf("height %v, want %v", maxY-minY, h-fitMargin)
		}
	})

	t.Run("larger by area is drawn first", func(t *testing.T) {
		for _, tc := range []struct{ left, right string }{{"SML", "BIG"}, {"BIG", "SML"}} {
			got := s.Layout(tc.left, tc.right, w, h)
			if len(got.Countries) != 2 {
				t.Fatalf("%v: got %d countries", tc, len(got.Countries))
			}
			first, second := got.Countries[0], got.Countries[1]
			if first.Name != "BIG" || first.Order != 0 || second.Name != "SML" || second.Order != 1 {
				t.Errorf("%v: order = %s(%d), %s(%d)", tc, first.Name, first.Order, second.Name, second.Order)
			}
			wantSide := SideLeft
			if tc.right == "BIG" {
				wantSide = SideRight
			}
			if first.Side != wantSide {
				t.Errorf("%v: BIG side = %q, want %q", tc, first.Side, wantSide)
			}
			// Shared scale: SML is a quarter of BIG's width.
			bMinX, _, bMaxX, _ := pixelBounds(first.Paths)
			sMinX, _, sMaxX, _ := pixelBounds(second.Paths)
			if ratio := (sMaxX - sMinX) / (bMaxX - bMinX); math.Abs(ratio-0.25) > 0.01 {
				t.Errorf("%v: width ratio %v, want 0.25", tc, ratio)
			}
		}
	})

	// R9: the smaller-by-area country that would overflow becomes the larger,
	// whichever list it came from.
	t.Run("overflowing smaller country swaps on either side", func(t *testing.T) {
		for _, tc := range []struct{ left, right, wantSide string }{
			{"WIDE", "BIG", SideLeft},
			{"BIG", "WIDE", SideRight},
		} {
			got := s.Layout(tc.left, tc.right, w, h)
			first := got.Countries[0]
			if first.Name != "WIDE" || first.Side != tc.wantSide {
				t.Errorf("%v: first = %s on %s, want WIDE on %s", tc, first.Name, first.Side, tc.wantSide)
			}
			for _, c := range got.Countries {
				minX, minY, maxX, maxY := pixelBounds(c.Paths)
				if minX < 0 || minY < 0 || maxX > w || maxY > h {
					t.Errorf("%v: %s overflows: x %v..%v, y %v..%v", tc, c.Name, minX, maxX, minY, maxY)
				}
			}
		}
	})

	t.Run("load failures are reported per country", func(t *testing.T) {
		got := s.Layout("GONE", "BAD", w, h)
		if len(got.Countries) != 2 {
			t.Fatalf("got %d countries, want 2", len(got.Countries))
		}
		for _, c := range got.Countries {
			if !strings.HasPrefix(c.Error, "Error loading "+c.Name+": ") || c.Paths == nil || len(c.Paths) != 0 {
				t.Errorf("%s: error %q, paths %v", c.Name, c.Error, c.Paths)
			}
		}
		got = s.Layout("GONE", "SML", w, h)
		if got.Countries[0].Name != "SML" || got.Countries[0].Error != "" || len(got.Countries[0].Paths) == 0 {
			t.Errorf("the country that loaded should still be laid out: %+v", got.Countries[0])
		}
		if got.Countries[1].Name != "GONE" || got.Countries[1].Error == "" {
			t.Errorf("the missing country should carry an error: %+v", got.Countries[1])
		}
	})

	t.Run("same country in both lists", func(t *testing.T) {
		got := s.Layout("SML", "SML", w, h)
		if len(got.Countries) != 2 || got.Countries[0].Side != SideLeft || got.Countries[1].Side != SideRight {
			t.Errorf("got %+v", got.Countries)
		}
	})
}

func TestMapServiceLayoutPaths(t *testing.T) {
	// A ring of two points (dropped, fewer than 3) and a square whose first two
	// corners collapse to one pixel after rounding.
	doc := `{"features":[
		{"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,1]]]}},
		{"geometry":{"type":"Polygon","coordinates":[[[0,0],[0.0000001,0],[10,0],[10,10],[0,10],[0,0]]]}}]}`
	s := newTestMapService(t, Settings{DebugShowBoundary: true}, map[string]string{"SQ": doc}, map[string]float64{"SQ": 1})

	got := s.Layout("SQ", "", 200, 200)
	c := got.Countries[0]
	if len(c.Paths) != 1 {
		t.Fatalf("got %d paths, want 1 (the 2-point ring is dropped)", len(c.Paths))
	}
	p := c.Paths[0]
	if len(p) != 10 {
		t.Errorf("path has %d numbers, want 10 (5 points after dropping the repeat): %v", len(p), p)
	}
	for _, v := range p {
		if v != round1(v) {
			t.Errorf("coordinate %v is not rounded to 0.1", v)
		}
	}
	if c.Box == nil {
		t.Fatal("Box is nil with debug_show_boundary on")
	}
	minX, minY, maxX, maxY := pixelBounds(c.Paths)
	if !nearPx(c.Box.X, minX) || !nearPx(c.Box.Y, minY) || !nearPx(c.Box.X+c.Box.Width, maxX) || !nearPx(c.Box.Y+c.Box.Height, maxY) {
		t.Errorf("Box %+v does not match path bounds x %v..%v, y %v..%v", *c.Box, minX, maxX, minY, maxY)
	}
}

func TestMapServiceLayoutRealData(t *testing.T) {
	cc, err := NewCountryCollection("country_data.json")
	if err != nil {
		t.Fatal(err)
	}
	settings := loadSettings("settings.json")
	s := NewMapService(settings, cc, "mapdata")
	const w, h = 1000.0, 700.0

	got := s.Layout("Fiji", "United States", w, h)
	if len(got.Countries) != 2 {
		t.Fatalf("got %d countries, want 2", len(got.Countries))
	}
	if got.Countries[0].Name != "United States" || got.Countries[0].Side != SideRight {
		t.Errorf("first = %s on %s, want United States on right", got.Countries[0].Name, got.Countries[0].Side)
	}
	for _, c := range got.Countries {
		if c.Error != "" || len(c.Paths) == 0 {
			t.Fatalf("%s: error %q, %d paths", c.Name, c.Error, len(c.Paths))
		}
		minX, minY, maxX, maxY := pixelBounds(c.Paths)
		if minX < 0 || minY < 0 || maxX > w || maxY > h {
			t.Errorf("%s outside the map box: x %v..%v, y %v..%v", c.Name, minX, maxX, minY, maxY)
		}
	}
}

func TestMapServiceMissingMapData(t *testing.T) {
	s := NewMapService(&Settings{}, nil, filepath.Join(t.TempDir(), "nowhere"))
	got := s.Layout("Fiji", "", 800, 600)
	if len(got.Countries) != 1 || got.Countries[0].Error == "" {
		t.Fatalf("got %+v, want one country with an error", got.Countries)
	}
	if !strings.Contains(got.Countries[0].Error, "Fiji") {
		t.Errorf("error %q does not name the country", got.Countries[0].Error)
	}
}
