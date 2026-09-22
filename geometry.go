// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import "math"

// Point represents a 2D point with float64 precision.
type Point struct {
	X float64 // X is the horizontal coordinate.
	Y float64 // Y is the vertical coordinate.
}

// BoundingBox represents a geographic or pixel-space bounding box.
type BoundingBox struct {
	MinX, MaxX    float64 // MinX and MaxX are the horizontal boundaries.
	MinY, MaxY    float64 // MinY and MaxY are the vertical boundaries.
	Width, Height float64 // Width and Height are the dimensions of the bounding box.
}

// Geometry represents a GeoJSON geometry object.
type Geometry struct {
	Type        string          // Type is the GeoJSON geometry type (e.g., "Polygon", "MultiPolygon").
	Coordinates [][][][]float64 // Coordinates holds the geometry's coordinate data.
}

// GeoData holds the parsed geographic paths and the overall bounding box for a country.
type GeoData struct {
	Paths       [][]Point   // Paths is a collection of path points for rendering.
	BoundingBox BoundingBox // BoundingBox is the calculated boundary of all paths.
}

// UpdateBoundingBox recalculates the bounding box based on the current Paths.
// This ensures the bounding box exactly matches the Mercator coordinates.
func (gd *GeoData) UpdateBoundingBox() {
	if len(gd.Paths) == 0 {
		return
	}

	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	found := false

	for _, path := range gd.Paths {
		for _, p := range path {
			x, y := p.X, p.Y
			minX = min(minX, x)
			maxX = max(maxX, x)
			minY = min(minY, y)
			maxY = max(maxY, y)
			found = true
		}
	}

	if !found {
		gd.BoundingBox = BoundingBox{}
		return
	}

	gd.BoundingBox = BoundingBox{
		MinX:   minX,
		MaxX:   maxX,
		MinY:   minY,
		MaxY:   maxY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}
}

// NeedsPacificCentering checks if a MultiPolygon spans across the anti-meridian
func NeedsPacificCentering(g Geometry) bool {
	var hasFarEast, hasFarWest bool

	for _, polygon := range g.Coordinates {
		for _, ring := range polygon {
			for _, coord := range ring {
				lng := coord[0]

				// Check if the coordinates exist significantly deep in both hemispheres
				if lng > 90.0 {
					hasFarEast = true
				}
				if lng < -90.0 {
					hasFarWest = true
				}

				// Early exit condition met
				if hasFarEast && hasFarWest {
					return true
				}
			}
		}
	}
	return false
}

// ApplyPacificCentering shifts negative longitudes to create a seamless 0 to 360 map
func ApplyPacificCentering(g Geometry) Geometry {
	if !NeedsPacificCentering(g) {
		return g
	}

	// Deep copy and transform coordinates
	newCoords := make([][][][]float64, len(g.Coordinates))
	for i, polygon := range g.Coordinates {
		newCoords[i] = make([][][]float64, len(polygon))
		for j, ring := range polygon {
			newCoords[i][j] = make([][]float64, len(ring))
			for k, coord := range ring {
				lng := coord[0]
				lat := coord[1]

				// Shift negative longitudes to the 180-360 range
				if lng < 0 {
					lng += 360.0
				}
				newCoords[i][j][k] = []float64{lng, lat}
			}
		}
	}

	return Geometry{
		Type:        g.Type,
		Coordinates: newCoords,
	}
}

// LatLonToMercator converts geographic (longitude, latitude) coordinates into Mercator projection coordinates.
// It returns (x, y) coordinates normalized in the range [0.0, 1.0].
func LatLonToMercator(lon, lat float64) (x, y float64) {
	// 1. Project to Mercator meters
	mx := EarthRadius * (lon * math.Pi / 180.0)
	my := EarthRadius * math.Log(math.Tan((math.Pi/4.0)+(lat*math.Pi/360.0)))

	// 2. Normalize Mercator coordinates to [0, 1]
	nx := (mx + MaxMercator) / (2.0 * MaxMercator)
	ny := (MaxMercator - my) / (2.0 * MaxMercator) // Invert Y for screen space

	return nx, ny
}

// minDrawable is the smallest drawable width or height, in pixels, that the fit
// math will scale into. fitMargin is subtracted from each dimension before fitting.
const (
	minDrawable = 4.0
	fitMargin   = 4.0
)

// fitScale returns the scale factor, in pixels per Mercator unit, that fits bb
// inside a drawable area of width by height pixels less fitMargin on each axis.
// width and height are the map area with the header already subtracted.
// It returns 1.0 when bb has zero width or height, or when either drawable
// dimension is below minDrawable.
func fitScale(bb BoundingBox, width, height float64) float64 {
	if bb.Width == 0 || bb.Height == 0 {
		return 1.0
	}
	if width < minDrawable || height < minDrawable {
		return 1.0
	}
	scaleX := (width - fitMargin) / bb.Width
	scaleY := (height - fitMargin) / bb.Height
	return min(scaleX, scaleY)
}

// mapCountry is one selected country as seen by scaleAndOrder.
type mapCountry struct {
	Name string       // Name is the country name; "" means nothing is selected.
	Area float64      // Area is the surface area in square miles.
	BB   *BoundingBox // BB is the Mercator bounding box; nil means its geo data failed to load.
}

// scaleAndOrder returns the shared scale for drawing active and other together
// and the draw order: larger is drawn first, smaller on top of it.
// width and height are the drawable map size as passed to fitScale.
//
// The larger country is chosen by area (ties go to active) and fitted. If the
// smaller-by-area country's bounding box then reaches the drawable edge at that
// scale, the two swap: it becomes the larger and the scale is refitted to it.
func scaleAndOrder(active, other mapCountry, width, height float64) (scale float64, larger, smaller string) {
	fit := func(c mapCountry) float64 {
		if c.BB == nil {
			return 1.0
		}
		return fitScale(*c.BB, width, height)
	}

	if active.Name == "" && other.Name == "" {
		return 1.0, "", ""
	}
	if active.Name == "" {
		return fit(other), other.Name, ""
	}
	if other.Name == "" {
		return fit(active), active.Name, ""
	}

	if active.BB == nil && other.BB == nil {
		return 1.0, active.Name, other.Name
	}
	if active.BB == nil {
		return fit(other), other.Name, active.Name
	}
	if other.BB == nil {
		return fit(active), active.Name, other.Name
	}

	if height < minDrawable || width < minDrawable {
		return 1.0, active.Name, other.Name
	}

	large, small := active, other
	if other.Area > active.Area {
		large, small = other, active
	}
	scale = fitScale(*large.BB, width, height)

	if small.BB.Width*scale >= width-fitMargin || small.BB.Height*scale >= height-fitMargin {
		large, small = small, large
		scale = fitScale(*large.BB, width, height)
	}

	return scale, large.Name, small.Name
}
