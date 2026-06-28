// SPDX-FileCopyrightText: 2026 Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

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

// getFitScale calculates the scale factor required to fit the bounding box of a country within the available display area.
func (a *App) getFitScale(country string) float64 {
	data, err := FetchAndCacheGeoJSON(country, true, a.Settings.SkipSmall, a.Settings.EnablePacificCenter, a.Settings.MapDataPath, a.GeoCache, a.CountryData)
	if err != nil {
		return 1.0
	}

	// Ensure bounding box is updated from paths (already done in FetchAndCacheGeoJSON, but good for safety)
	data.UpdateBoundingBox()

	mercWidth := data.BoundingBox.Width
	mercHeight := data.BoundingBox.Height
	if mercWidth == 0 || mercHeight == 0 {
		return 1.0
	}

	size := a.cMap.Container.Size()
	availableHeight := float64(size.Height)

	if a.headerContainer != nil {
		availableHeight -= float64(a.headerContainer.MinSize().Height)
	}

	// Subtract footer height
	if a.cCenter != nil && len(a.cCenter.Objects) > 0 {
		for _, obj := range a.cCenter.Objects {
			if footer, ok := obj.(*fyne.Container); ok {
				// The footer is likely the Bottom in Border layout.
				// Check if it contains our buttons.
				isFooter := false
				for _, child := range footer.Objects {
					if btn, ok := child.(*widget.Button); ok && (strings.Contains(btn.Text, "Exit") || strings.Contains(btn.Text, "About")) {
						isFooter = true
						break
					}
				}
				if isFooter {
					availableHeight -= float64(footer.MinSize().Height)
					break
				}
			}
		}
	}

	if availableHeight < 0 {
		availableHeight = 0
	}

	if float64(size.Width) < 4 || availableHeight < 4 {
		return 1.0
	}

	scaleX := (float64(size.Width) - 4) / mercWidth
	scaleY := (availableHeight - 4) / mercHeight
	return min(scaleX, scaleY)
}

// getScaleAndOrder determines the appropriate scale and drawing order for selected countries.
// It uses square mileage as the primary factor to determine the larger country and scales it to fit.
func (a *App) getScaleAndOrder(active, other string) (float64, string, string) {
	if active == "" && other == "" {
		return 1.0, "", ""
	}
	if active == "" {
		return a.getFitScale(other), other, ""
	}
	if other == "" {
		return a.getFitScale(active), active, ""
	}

	dataActive, errActive := FetchAndCacheGeoJSON(active, true, a.Settings.SkipSmall, a.Settings.EnablePacificCenter, a.Settings.MapDataPath, a.GeoCache, a.CountryData)
	dataOther, errOther := FetchAndCacheGeoJSON(other, true, a.Settings.SkipSmall, a.Settings.EnablePacificCenter, a.Settings.MapDataPath, a.GeoCache, a.CountryData)

	if errActive != nil && errOther != nil {
		return 1.0, active, other
	}
	if errActive != nil {
		return a.getFitScale(other), other, active
	}
	if errOther != nil {
		return a.getFitScale(active), active, other
	}

	dataActive.UpdateBoundingBox()
	dataOther.UpdateBoundingBox()

	if a.cMap == nil {
		return 1.0, active, other
	}

	size := a.cMap.Container.Size()
	availableHeight := float64(size.Height)
	if a.headerContainer != nil {
		availableHeight -= float64(a.headerContainer.MinSize().Height)
	}
	// Subtract footer height
	if a.cCenter != nil && len(a.cCenter.Objects) > 0 {
		for _, obj := range a.cCenter.Objects {
			if footer, ok := obj.(*fyne.Container); ok {
				isFooter := false
				for _, child := range footer.Objects {
					if btn, ok := child.(*widget.Button); ok && (strings.Contains(btn.Text, "Exit") || strings.Contains(btn.Text, "About")) {
						isFooter = true
						break
					}
				}
				if isFooter {
					availableHeight -= float64(footer.MinSize().Height)
					break
				}
			}
		}
	}
	if availableHeight < 4 || float64(size.Width) < 4 {
		return 1.0, active, other
	}

	areaActive := a.getArea(active)
	areaOther := a.getArea(other)

	// Step 1: Start with square mileage to determine larger country
	larger, smaller := active, other
	if areaOther > areaActive {
		larger, smaller = other, active
	}

	// Step 2: See if either country's pixel delta is larger than the current drawing area size
	// We need a baseline scale to check pixel deltas.
	// We'll use a temporary fit scale for the larger country as the baseline.
	scale := a.getFitScale(larger)

	// Check if the other country (smaller by area) is actually larger in pixel delta at this scale.
	// This can happen with countries that have extreme aspect ratios or spans.
	if dataOther.BoundingBox.Width*scale >= float64(size.Width)-4 || dataOther.BoundingBox.Height*scale >= availableHeight-4 {
		// If the smaller country (by sq mileage) doesn't fit at the larger country's scale,
		// it must be the one that dictates the fit scale.
		larger, smaller = other, active
		scale = a.getFitScale(larger)
	}

	return scale, larger, smaller
}
