// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package geo

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseGeoJSONRealFiles(t *testing.T) {
	// Golden values captured from the unmodified code. The first config matches
	// the shipped settings.json (skip_small 25, Pacific centering on).
	tests := []struct {
		file        string
		skipSmall   int
		pacific     bool
		wantPaths   int
		wantPoints  int
		wantBB      BoundingBox
		wantFirstPt Point
	}{
		{
			"FJI", 25, true, 48, 6606,
			BoundingBox{MinX: 0.991683611111111, MaxX: 1.0044647222222223, MinY: 0.5349426635798549, MaxY: 0.554340766640616, Width: 0.012781111111111243, Height: 0.019398103060761174},
			Point{X: 0.999341111111111, Y: 0.5541234556025452},
		},
		{
			// Without centering Fiji straddles the antimeridian and spans the whole map width.
			"FJI", 25, false, 48, 6606,
			BoundingBox{MinX: 0, MaxX: 1, MinY: 0.5349426635798549, MaxY: 0.554340766640616, Width: 1, Height: 0.019398103060761174},
			Point{X: 0.999341111111111, Y: 0.5541234556025452},
		},
		{
			"USA", 25, true, 81, 24400,
			BoundingBox{MinX: 0.9017175000000001, MaxX: 1.3206486111111109, MinY: 0.2124420202455855, MaxY: 0.5403524566737985, Width: 0.4189311111111108, Height: 0.327910436428213},
			Point{X: 0.9961819444444443, Y: 0.3319449616881121},
		},
		{
			"USA", 25, false, 81, 24400,
			BoundingBox{MinX: 0.004922777777777865, MaxX: 0.9985208333333333, MinY: 0.2124420202455855, MaxY: 0.5403524566737985, Width: 0.9935980555555555, Height: 0.327910436428213},
			Point{X: 0.9961819444444443, Y: 0.3319449616881121},
		},
	}
	for _, tt := range tests {
		data, err := os.ReadFile(filepath.Join("..", "..", "mapdata", tt.file+".geojson"))
		if err != nil {
			t.Fatal(err)
		}
		name := fmt.Sprintf("%s skip=%d pacific=%v", tt.file, tt.skipSmall, tt.pacific)
		t.Run(name, func(t *testing.T) {
			gd, err := ParseGeoJSON(data, tt.skipSmall, tt.pacific)
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

func TestParseGeoJSONSynthetic(t *testing.T) {
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

	t.Run("other types and bad coordinates skipped", func(t *testing.T) {
		gd, err := ParseGeoJSON([]byte(doc), 0, false)
		if err != nil {
			t.Fatal(err)
		}
		x0, _ := latLonToMercator(0, 0)
		x21, _ := latLonToMercator(21, 0)
		if !near(gd.BoundingBox.MinX, x0) || !near(gd.BoundingBox.MaxX, x21) {
			t.Errorf("BoundingBox X = [%v, %v], want [%v, %v]", gd.BoundingBox.MinX, gd.BoundingBox.MaxX, x0, x21)
		}
	})

	t.Run("holes dropped", func(t *testing.T) {
		gd, err := ParseGeoJSON([]byte(doc), 0, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(gd.Paths) != 2 || len(gd.Paths[0]) != 5 || len(gd.Paths[1]) != 3 {
			t.Errorf("got %d paths, want outer ring (5) and triangle (3)", len(gd.Paths))
		}
	})

	t.Run("skipSmall drops rings with that many points or fewer", func(t *testing.T) {
		gd, err := ParseGeoJSON([]byte(doc), 4, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(gd.Paths) != 1 || len(gd.Paths[0]) != 5 {
			t.Errorf("got %d paths, want only the 5-point ring", len(gd.Paths))
		}
	})

	t.Run("no usable features", func(t *testing.T) {
		gd, err := ParseGeoJSON([]byte(`{"features": []}`), 0, false)
		if err != nil {
			t.Fatal(err)
		}
		if gd.Paths != nil || gd.BoundingBox != (BoundingBox{}) {
			t.Errorf("got %+v, want no paths and a zero box", gd)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := ParseGeoJSON([]byte(`{`), 0, false); err == nil {
			t.Error("expected an error")
		}
	})
}
