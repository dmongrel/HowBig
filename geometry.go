package main

import (
	"math"
)

// Point represents a 2D point with float64 precision.
type Point struct {
	X, Y float64
}

// BoundingBox represents a geographic or pixel-space bounding box.
type BoundingBox struct {
	MinX, MaxX    float64
	MinY, MaxY    float64
	Width, Height float64
}

// Geometry represents a GeoJSON geometry object.
type Geometry struct {
	Type        string
	Coordinates [][][][]float64
}

// GeoData holds the parsed geographic paths and the overall bounding box for a country.
type GeoData struct {
	Paths       [][]Point
	BoundingBox BoundingBox
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
func getFitScale(country string) float64 {
	data, err := FetchAndCacheGeoJSON(country, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
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

	size := cMap.Container.Size()
	var h float64
	if headerContainer != nil {
		h = float64(headerContainer.MinSize().Height)
	}
	availableHeight := float64(size.Height)
	if availableHeight > h {
		availableHeight -= h
	} else {
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
// It uses the larger of the two bounding box areas to ensure both are visible and fit the screen.
func getScaleAndOrder(active, other string) (float64, string, string) {
	if active == "" && other == "" {
		return 1.0, "", ""
	}
	if active == "" {
		return getFitScale(other), other, ""
	}
	if other == "" {
		return getFitScale(active), active, ""
	}

	dataActive, errActive := FetchAndCacheGeoJSON(active, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
	dataOther, errOther := FetchAndCacheGeoJSON(other, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)

	if errActive != nil && errOther != nil {
		return 1.0, active, other
	}
	if errActive != nil {
		return getFitScale(other), other, active
	}
	if errOther != nil {
		return getFitScale(active), active, other
	}

	dataActive.UpdateBoundingBox()
	dataOther.UpdateBoundingBox()

	if cMap == nil {
		return 1.0, active, other
	}
	size := cMap.Container.Size()
	if size.Width < 4 || size.Height < 4 {
		return 1.0, active, other
	}

	areaActive := dataActive.BoundingBox.Width * dataActive.BoundingBox.Height
	areaOther := dataOther.BoundingBox.Width * dataOther.BoundingBox.Height

	if areaActive > areaOther {
		return getFitScale(active), active, other
	}
	return getFitScale(other), other, active
}
