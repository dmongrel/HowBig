// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"math"
	"testing"
)

// near reports whether a and b agree to within 1e-12.
func near(a, b float64) bool {
	return math.Abs(a-b) <= 1e-12
}

func TestLatLonToMercator(t *testing.T) {
	tests := []struct {
		name         string
		lon, lat     float64
		wantX, wantY float64
	}{
		{"origin", 0, 0, 0.5, 0.5},
		{"antimeridian east", 180, 0, 1, 0.5},
		{"antimeridian west", -180, 0, 0, 0.5},
		{"beyond 180 is not wrapped", 270, 0, 1.25, 0.5},
		{"web mercator top edge", 0, 85.0511287798066, 0.5, 0},
		{"new york", -74.006, 40.7128, 0.2944277777777778, 0.3759807982580088},
		{"north pole is not clamped", 0, 90, 0.5, -5.441549447954536},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := LatLonToMercator(tt.lon, tt.lat)
			if !near(x, tt.wantX) || !near(y, tt.wantY) {
				t.Errorf("LatLonToMercator(%v, %v) = (%v, %v), want (%v, %v)", tt.lon, tt.lat, x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

// multiPolygon builds a one-polygon, one-ring Geometry from the given longitudes (latitude 0).
func multiPolygon(lons ...float64) Geometry {
	ring := make([][]float64, len(lons))
	for i, lon := range lons {
		ring[i] = []float64{lon, 0}
	}
	return Geometry{Type: "MultiPolygon", Coordinates: [][][][]float64{{ring}}}
}

func TestNeedsPacificCentering(t *testing.T) {
	tests := []struct {
		name string
		g    Geometry
		want bool
	}{
		{"empty", Geometry{}, false},
		{"only far east", multiPolygon(100, 170, 179), false},
		{"only far west", multiPolygon(-100, -170), false},
		{"both far hemispheres", multiPolygon(170, -170), true},
		{"exactly 90 and -90 do not count", multiPolygon(90, -90), false},
		{"near zero both signs", multiPolygon(-10, 10), false},
		{
			"far east and far west in different polygons",
			Geometry{Coordinates: [][][][]float64{{{{179, 0}}}, {{{-179, 0}}}}},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NeedsPacificCentering(tt.g); got != tt.want {
				t.Errorf("NeedsPacificCentering() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyPacificCentering(t *testing.T) {
	t.Run("no centering needed returns input unchanged", func(t *testing.T) {
		g := multiPolygon(-10, 10)
		got := ApplyPacificCentering(g)
		if &got.Coordinates[0][0][0][0] != &g.Coordinates[0][0][0][0] {
			t.Error("expected the same backing coordinates when no centering is needed")
		}
	})

	t.Run("negative longitudes shift by 360", func(t *testing.T) {
		g := multiPolygon(170, -170, 0, -0.5)
		g.Type = "Polygon"
		g.Coordinates[0][0][1] = []float64{-170, 12, 99} // carries an extra ordinate
		got := ApplyPacificCentering(g)

		if got.Type != "Polygon" {
			t.Errorf("Type = %q, want Polygon", got.Type)
		}
		want := [][]float64{{170, 0}, {190, 12}, {0, 0}, {359.5, 0}}
		ring := got.Coordinates[0][0]
		if len(ring) != len(want) {
			t.Fatalf("ring has %d coords, want %d", len(ring), len(want))
		}
		for i := range want {
			// Characterization: coordinates are rebuilt as [lng, lat], so a third ordinate is dropped.
			if len(ring[i]) != 2 || ring[i][0] != want[i][0] || ring[i][1] != want[i][1] {
				t.Errorf("coord %d = %v, want %v", i, ring[i], want[i])
			}
		}
		if g.Coordinates[0][0][1][0] != -170 || len(g.Coordinates[0][0][1]) != 3 {
			t.Errorf("input was mutated: %v", g.Coordinates[0][0][1])
		}
	})
}

func TestUpdateBoundingBox(t *testing.T) {
	t.Run("computes extents", func(t *testing.T) {
		gd := GeoData{Paths: [][]Point{
			{{X: 1, Y: 2}, {X: 3, Y: -1}},
			{{X: -2, Y: 5}},
		}}
		gd.UpdateBoundingBox()
		want := BoundingBox{MinX: -2, MaxX: 3, MinY: -1, MaxY: 5, Width: 5, Height: 6}
		if gd.BoundingBox != want {
			t.Errorf("BoundingBox = %+v, want %+v", gd.BoundingBox, want)
		}
	})

	t.Run("no paths leaves the old box in place", func(t *testing.T) {
		old := BoundingBox{MinX: 1, MaxX: 2, Width: 1}
		gd := GeoData{BoundingBox: old}
		gd.UpdateBoundingBox()
		if gd.BoundingBox != old {
			t.Errorf("BoundingBox = %+v, want unchanged %+v", gd.BoundingBox, old)
		}
	})

	t.Run("only empty paths zeroes the box", func(t *testing.T) {
		gd := GeoData{Paths: [][]Point{{}, {}}, BoundingBox: BoundingBox{MinX: 1, Width: 1}}
		gd.UpdateBoundingBox()
		if gd.BoundingBox != (BoundingBox{}) {
			t.Errorf("BoundingBox = %+v, want zero", gd.BoundingBox)
		}
	})
}
