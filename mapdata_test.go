// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertGeoJSONRealFiles(t *testing.T) {
	// Golden values captured from the unmodified code. The first config matches
	// the shipped settings.json (skip_small 25, Pacific centering on).
	tests := []struct {
		file        string
		single      bool
		skipSmall   int
		pacific     bool
		wantPaths   int
		wantPoints  int
		wantBB      BoundingBox
		wantFirstPt Point
	}{
		{
			"FJI", true, 25, true, 48, 6606,
			BoundingBox{MinX: 0.991683611111111, MaxX: 1.0044647222222223, MinY: 0.5349426635798549, MaxY: 0.554340766640616, Width: 0.012781111111111243, Height: 0.019398103060761174},
			Point{X: 0.999341111111111, Y: 0.5541234556025452},
		},
		{
			// Without centering Fiji straddles the antimeridian and spans the whole map width.
			"FJI", true, 25, false, 48, 6606,
			BoundingBox{MinX: 0, MaxX: 1, MinY: 0.5349426635798549, MaxY: 0.554340766640616, Width: 1, Height: 0.019398103060761174},
			Point{X: 0.999341111111111, Y: 0.5541234556025452},
		},
		{
			"FJI", false, 0, true, 99, 7416,
			BoundingBox{MinX: 0.991416111111111, MaxX: 1.0049194444444445, MinY: 0.5349426635798549, MaxY: 0.5587086792236728, Width: 0.013503333333333423, Height: 0.023766015643817973},
			Point{X: 1.0035194444444444, Y: 0.558600326750366},
		},
		{
			"USA", true, 25, true, 81, 24400,
			BoundingBox{MinX: 0.9017175000000001, MaxX: 1.3206486111111109, MinY: 0.2124420202455855, MaxY: 0.5403524566737985, Width: 0.4189311111111108, Height: 0.327910436428213},
			Point{X: 0.9961819444444443, Y: 0.3319449616881121},
		},
		{
			"USA", true, 25, false, 81, 24400,
			BoundingBox{MinX: 0.004922777777777865, MaxX: 0.9985208333333333, MinY: 0.2124420202455855, MaxY: 0.5403524566737985, Width: 0.9935980555555555, Height: 0.327910436428213},
			Point{X: 0.9961819444444443, Y: 0.3319449616881121},
		},
		{
			"USA", false, 0, false, 290, 26810,
			BoundingBox{MinX: 0.0023686111111111416, MaxX: 0.9993844444444443, MinY: 0.2124420202455855, MaxY: 0.5408653787405515, Width: 0.9970158333333331, Height: 0.328423358494966},
			Point{X: 0.9985622222222224, Y: 0.33039268269279265},
		},
	}
	for _, tt := range tests {
		data, err := os.ReadFile(filepath.Join("mapdata", tt.file+".geojson"))
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("%s single=%v skip=%d pacific=%v", tt.file, tt.single, tt.skipSmall, tt.pacific)
		t.Run(name, func(t *testing.T) {
			gd, err := convertGeoJSONToDisplayFormat(data, tt.single, tt.skipSmall, tt.pacific)
			if err != nil {
				t.Fatal(err)
			}
			points := 0
			for _, p := range gd.Paths {
				points += len(p)
			}
			if len(gd.Paths) != tt.wantPaths || points != tt.wantPoints {
				t.Errorf("got %d paths / %d points, want %d / %d", len(gd.Paths), points, tt.wantPaths, tt.wantPoints)
			}
			bb, w := gd.BoundingBox, tt.wantBB
			if !near(bb.MinX, w.MinX) || !near(bb.MaxX, w.MaxX) || !near(bb.MinY, w.MinY) || !near(bb.MaxY, w.MaxY) ||
				!near(bb.Width, w.Width) || !near(bb.Height, w.Height) {
				t.Errorf("BoundingBox = %+v, want %+v", bb, w)
			}
			if first := gd.Paths[0][0]; !near(first.X, tt.wantFirstPt.X) || !near(first.Y, tt.wantFirstPt.Y) {
				t.Errorf("first point = %+v, want %+v", first, tt.wantFirstPt)
			}
		})
	}
}

func TestConvertGeoJSONSynthetic(t *testing.T) {
	const doc = `{"features": [
		{"geometry": {"type": "Polygon", "coordinates": [
			[[0,0],[10,0],[10,10],[0,10],[0,0]],
			[[2,2],[3,2],[3,3],[2,2]]
		]}},
		{"geometry": {"type": "LineString", "coordinates": [[0,0],[1,1]]}},
		{"geometry": {"type": "Polygon", "coordinates": "not coordinates"}},
		{"geometry": {"type": "MultiPolygon", "coordinates": [
			[[[20,0],[21,0],[21,1]]]
		]}}
	]}`

	t.Run("holes kept, other types and bad coordinates skipped", func(t *testing.T) {
		gd, err := convertGeoJSONToDisplayFormat([]byte(doc), false, 0, false)
		if err != nil {
			t.Fatal(err)
		}
		lens := []int{}
		for _, p := range gd.Paths {
			lens = append(lens, len(p))
		}
		if len(lens) != 3 || lens[0] != 5 || lens[1] != 4 || lens[2] != 3 {
			t.Errorf("path lengths = %v, want [5 4 3]", lens)
		}
		x0, _ := LatLonToMercator(0, 0)
		x21, _ := LatLonToMercator(21, 0)
		if !near(gd.BoundingBox.MinX, x0) || !near(gd.BoundingBox.MaxX, x21) {
			t.Errorf("BoundingBox X = [%v, %v], want [%v, %v]", gd.BoundingBox.MinX, gd.BoundingBox.MaxX, x0, x21)
		}
	})

	t.Run("single polyline drops holes", func(t *testing.T) {
		gd, err := convertGeoJSONToDisplayFormat([]byte(doc), true, 0, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(gd.Paths) != 2 || len(gd.Paths[0]) != 5 || len(gd.Paths[1]) != 3 {
			t.Errorf("got %d paths, want outer ring (5) and triangle (3)", len(gd.Paths))
		}
	})

	t.Run("skipSmall drops rings with that many points or fewer", func(t *testing.T) {
		gd, err := convertGeoJSONToDisplayFormat([]byte(doc), false, 4, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(gd.Paths) != 1 || len(gd.Paths[0]) != 5 {
			t.Errorf("got %d paths, want only the 5-point ring", len(gd.Paths))
		}
	})

	t.Run("no usable features", func(t *testing.T) {
		gd, err := convertGeoJSONToDisplayFormat([]byte(`{"features": []}`), false, 0, false)
		if err != nil {
			t.Fatal(err)
		}
		if gd.Paths != nil || gd.BoundingBox != (BoundingBox{}) {
			t.Errorf("got %+v, want no paths and a zero box", gd)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := convertGeoJSONToDisplayFormat([]byte(`{`), false, 0, false); err == nil {
			t.Error("expected an error")
		}
	})
}

func TestGeoCacheEviction(t *testing.T) {
	a, b, c, d := &GeoData{}, &GeoData{}, &GeoData{}, &GeoData{}

	t.Run("evicts least recently used", func(t *testing.T) {
		cache := NewGeoCache(2)
		cache.Put("a", a)
		cache.Put("b", b)
		if _, ok := cache.Get("a"); !ok { // a becomes most recent
			t.Fatal("a missing")
		}
		cache.Put("c", c) // evicts b
		if _, ok := cache.Get("b"); ok {
			t.Error("b should have been evicted")
		}
		for _, k := range []string{"a", "c"} {
			if _, ok := cache.Get(k); !ok {
				t.Errorf("%s should still be cached", k)
			}
		}
	})

	t.Run("re-put replaces value and refreshes recency", func(t *testing.T) {
		cache := NewGeoCache(2)
		cache.Put("a", a)
		cache.Put("b", b)
		cache.Put("a", d) // a is now most recent and holds d
		cache.Put("c", c) // evicts b
		if got, ok := cache.Get("a"); !ok || got != d {
			t.Errorf("Get(a) = %p, %v; want %p, true", got, ok, d)
		}
		if _, ok := cache.Get("b"); ok {
			t.Error("b should have been evicted")
		}
	})

	t.Run("limit zero still holds one entry", func(t *testing.T) {
		cache := NewGeoCache(0)
		cache.Put("a", a)
		if _, ok := cache.Get("a"); !ok {
			t.Error("a should be cached")
		}
		cache.Put("b", b)
		if _, ok := cache.Get("a"); ok {
			t.Error("a should have been evicted")
		}
		if _, ok := cache.Get("b"); !ok {
			t.Error("b should be cached")
		}
	})

	t.Run("miss", func(t *testing.T) {
		if got, ok := NewGeoCache(1).Get("x"); ok || got != nil {
			t.Errorf("Get on empty cache = %v, %v", got, ok)
		}
	})
}

func TestFetchGeoJSONPathFallback(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(filepath.Join(root, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	geo := `{"features":[{"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}}]}`
	if err := os.WriteFile(filepath.Join(root, "maps", "X.geojson"), []byte(geo), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)

	t.Run("parent fallback is used when the file exists there", func(t *testing.T) {
		if _, err := FetchAndCacheGeoJSON("X", true, 0, false, "maps", nil, nil); err != nil {
			t.Fatalf("fallback not used: %v", err)
		}
	})
	t.Run("a missing file's error names the primary path", func(t *testing.T) {
		_, err := FetchAndCacheGeoJSON("Y", true, 0, false, "maps", nil, nil)
		if err == nil {
			t.Fatal("want an error")
		}
		want := filepath.Join("maps", "Y.geojson")
		if msg := err.Error(); !strings.Contains(msg, want) || strings.Contains(msg, "..") {
			t.Errorf("error %q should name %q and not the .. fallback", msg, want)
		}
	})
}
