package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	EarthRadius = 6378137.0
	MaxMercator = EarthRadius * math.Pi
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

// UpdateBoundingBox recalculates the bounding box based on the current Paths and a scale factor.
// This ensures the bounding box exactly matches the pixel coordinates used for drawing at the given scale.
func (gd *GeoData) UpdateBoundingBox(scale float64) {
	if len(gd.Paths) == 0 {
		return
	}

	minX, maxX := math.MaxFloat64, -math.MaxFloat64
	minY, maxY := math.MaxFloat64, -math.MaxFloat64
	found := false

	for _, path := range gd.Paths {
		for _, p := range path {
			x, y := p.X*scale, p.Y*scale
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

var (
	geoCache = make(map[string]*GeoData)
	cacheMu  sync.RWMutex
)

// FetchAndCacheGeoJSON loads and caches parsed GeoJSON paths and their overall geographic bounding box from the mapdata directory.
// The country parameter specifies the country name, and the singlePolyline parameter indicates if only the outer ring should be kept.
// The skipSmall parameter indicates the maximum number of coordinates a polygon ring can have to be skipped.
// The enablePacificCenter parameter indicates if the coordinates should be normalized to Pacific centering if IDL crossing is detected.
// It returns a GeoData struct containing the paths and the overall geographic bounding box, or an error if the file cannot be read or parsed.
func FetchAndCacheGeoJSON(country string, singlePolyline bool, skipSmall int, enablePacificCenter bool) (*GeoData, error) {

	fileName := getFileName(country)
	filePath := filepath.Join("mapdata", fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		filePath = filepath.Join("..", "mapdata", fileName)
	}

	cacheKey := country
	if singlePolyline {
		cacheKey += "_single"
	}
	if skipSmall > 0 {
		cacheKey += "_skip" + fmt.Sprint(skipSmall)
	}
	if enablePacificCenter {
		cacheKey += "_pacific"
	}

	cacheMu.RLock()
	cachedData, ok := geoCache[cacheKey]
	cacheMu.RUnlock()
	if ok {
		return cachedData, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	geoData, err := convertGeoJSONToDisplayFormat(data, singlePolyline, skipSmall, enablePacificCenter)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	geoCache[cacheKey] = geoData
	cacheMu.Unlock()
	return geoData, nil
}

// getFileName resolves the correct GeoJSON filename for a given country name.
// It uses CompactName from CountryData if available, otherwise it falls back to a formatted name.
func getFileName(name string) string {
	if CountryData != nil {
		for _, c := range CountryData.Countries {
			if c.Name == name {
				return c.CompactName
			}
		}
	}
	return strings.ReplaceAll(name, " ", "") + ".geojson"
}

// convertGeoJSONToDisplayFormat parses raw GeoJSON bytes into a format suitable for drawing on the Fyne canvas.
// It also calculates the overall Mercator bounding box of all features.
// The singlePolyline flag indicates if polygon simplification (keeping only the outer ring) should be applied.
// The skipSmall parameter indicates if polygons with skipSmall or fewer coordinates should be skipped.
// It returns a GeoData struct containing the paths and the overall Mercator bounding box, or an error if the file cannot be read or parsed.
func convertGeoJSONToDisplayFormat(data []byte, singlePolyline bool, skipSmall int, enablePacificCenter bool) (*GeoData, error) {
	var fc struct {
		Features []struct {
			Geometry struct {
				Type        string          `json:"type"`
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}

	var allPaths [][]Point

	for _, f := range fc.Features {
		var g Geometry
		g.Type = f.Geometry.Type

		if g.Type == "Polygon" {
			var coords [][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			// Wrap Polygon coordinates into MultiPolygon-like structure for unified processing
			g.Coordinates = [][][][]float64{coords}
		} else if g.Type == "MultiPolygon" {
			if err := json.Unmarshal(f.Geometry.Coordinates, &g.Coordinates); err != nil {
				continue
			}
		} else {
			continue
		}

		needsCentering := NeedsPacificCentering(g)
		log.Printf("Geometry (%s) needs Pacific centering: %v", g.Type, needsCentering)

		if enablePacificCenter && needsCentering {
			g = ApplyPacificCentering(g)
		}

		for _, polygon := range g.Coordinates {
			polyToProcess := polygon
			if singlePolyline {
				polyToProcess = singlePolyLine(polygon)
			}
			for _, ring := range polyToProcess {
				if len(ring) <= skipSmall {
					continue
				}

				var path []Point
				for _, pt := range ring {
					mx, my := LatLonToMercator(pt[0], pt[1])
					path = append(path, Point{X: mx, Y: my})
				}
				allPaths = append(allPaths, path)
			}
		}
	}

	geoData := &GeoData{
		Paths: allPaths,
	}
	geoData.UpdateBoundingBox(1.0)

	return geoData, nil
}

// singlePolyLine processes polygon rings and keeps only the outer ring, discarding holes or interior polygons.
// It returns a slice containing only the first ring of the provided polygon structure.
func singlePolyLine(rings [][][]float64) [][][]float64 {
	if len(rings) > 0 {
		return rings[0:1]
	}
	return rings
}
