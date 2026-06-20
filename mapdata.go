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

	"fyne.io/fyne/v2"
)

const (
	EarthRadius = 6378137.0
	MaxMercator = EarthRadius * math.Pi
)

// Coordinate represents a standard lat/lon point.
type Coordinate struct {
	Lon float64
	Lat float64
}

// BoundingBox represents a geographic or pixel-space bounding box.
type BoundingBox struct {
	MinX, MaxX float64
	MinY, MaxY float64
}

// Geometry represents a GeoJSON geometry object.
type Geometry struct {
	Type        string
	Coordinates [][][][]float64
	BoundingBox BoundingBox
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
		BoundingBox: g.BoundingBox,
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

/*
// GetMercatorBounds calculates the bounding box of a country in normalized Mercator coordinates [0, 1].
func GetMercatorBounds(country string, skipSmall int, enablePacificCenter bool) (minX, minY, maxX, maxY float64, err error) {
	paths, err := FetchAndCacheGeoJSON(country, true, skipSmall, enablePacificCenter)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64
	for _, path := range paths {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			minX = min(minX, mx)
			maxX = max(maxX, mx)
			minY = min(minY, my)
			maxY = max(maxY, my)
		}
	}
	return minX, minY, maxX, maxY, nil
}
*/

/*
// GetBoundingBox calculates the bounding box of a country in pixel coordinates.
// It returns the min/max X and Y values in screen space.
func GetBoundingBox(country string, scale float32, offsetX, offsetY float32, skipSmall int, enablePacificCenter bool) (minX, minY, maxX, maxY float64, err error) {
	mercMinX, mercMinY, mercMaxX, mercMaxY, err := GetMercatorBounds(country, skipSmall, enablePacificCenter)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	minX = float64(offsetX)
	minY = float64(offsetY)
	maxX = minX + (mercMaxX-mercMinX)*float64(scale)
	maxY = minY + (mercMaxY-mercMinY)*float64(scale)

	return minX, minY, maxX, maxY, nil
}
*/

// GetBoundingBox calculates the bounding box of a country in pixel coordinates.
// It returns the min/max X and Y values in screen space.
func GetBoundingBox(country string, scale float32, offsetX, offsetY float32, skipSmall int, enablePacificCenter bool) (minX, minY, maxX, maxY float64, err error) {
	paths, err := FetchAndCacheGeoJSON(country, true, skipSmall, enablePacificCenter)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64

	for _, path := range paths {
		for _, pos := range path {
			// 1. Project Lon/Lat to normalized Mercator [0, 1]
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))

			// 2. Transform to pixel space
			px := float64(offsetX) + mx*float64(scale)
			py := float64(offsetY) + my*float64(scale)

			// 3. Update bounds
			minX = min(minX, px)
			maxX = max(maxX, px)
			minY = min(minY, py)
			maxY = max(maxY, py)
		}
	}

	return minX, minY, maxX, maxY, nil
}

// geoCache stores parsed geoJSON paths for efficient retrieval.
// cacheMu ensures thread-safe access to geoCache.
var (
	geoCache = make(map[string][][]fyne.Position)
	cacheMu  sync.RWMutex
)

// FetchAndCacheGeoJSON loads and caches parsed GeoJSON paths from the mapdata directory.
// The country parameter specifies the country name, and the singlePolyline parameter indicates if only the outer ring should be kept.
// The skipSmall parameter indicates the maximum number of coordinates a polygon ring can have to be skipped.
// The enablePacificCenter parameter indicates if the coordinates should be normalized to Pacific-centering if IDL crossing is detected.
// It returns a slice of paths for each polygon, or an error if the file cannot be read or parsed.
func FetchAndCacheGeoJSON(country string, singlePolyline bool, skipSmall int, enablePacificCenter bool) ([][]fyne.Position, error) {

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
	paths, ok := geoCache[cacheKey]
	cacheMu.RUnlock()
	if ok {
		return paths, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	paths, err = convertGeoJSONToDisplayFormat(data, singlePolyline, skipSmall, enablePacificCenter)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	geoCache[cacheKey] = paths
	cacheMu.Unlock()
	return paths, nil
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
	return strings.ReplaceAll(name, " ", "") + ".gjson"
}

// convertGeoJSONToDisplayFormat parses raw GeoJSON bytes into a format suitable for drawing on the Fyne canvas.
// The singlePolyline flag indicates if polygon simplification (keeping only the outer ring) should be applied.
// The skipSmall parameter indicates if polygons with skipSmall or fewer coordinates should be skipped.
// It returns a slice of paths for each polygon, or an error if the file cannot be read or parsed.
func convertGeoJSONToDisplayFormat(data []byte, singlePolyline bool, skipSmall int, enablePacificCenter bool) ([][]fyne.Position, error) {
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

	var allPaths [][]fyne.Position
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

				var path []fyne.Position
				for _, pt := range ring {
					path = append(path, fyne.NewPos(float32(pt[0]), float32(pt[1])))
				}
				allPaths = append(allPaths, path)
			}
		}
	}
	return allPaths, nil
}

// singlePolyLine processes polygon rings and keeps only the outer ring, discarding holes or interior polygons.
// It returns a slice containing only the first ring of the provided polygon structure.
func singlePolyLine(rings [][][]float64) [][][]float64 {
	if len(rings) > 0 {
		return rings[0:1]
	}
	return rings
}

// LogSouthernmostPixels calculates and logs the southernmost and easternmost drawn country coordinate and
// the southernmost and easternmost bounding box coordinate in pixel values.
func LogSouthernmostPixels(country string, paths [][]fyne.Position, scale float32, offsetX, offsetY float32, transformedMinX, transformedMaxX, transformedMinY, transformedMaxY float64) {
	pixelMaxY := float64(offsetY) + (transformedMaxY-transformedMinY)*float64(scale)
	pixelMaxX := float64(offsetX) + (transformedMaxX-transformedMinX)*float64(scale)

	maxDrawnY := -math.MaxFloat64
	maxDrawnX := -math.MaxFloat64
	for _, path := range paths {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			drawnY := (my-transformedMinY)*float64(scale) + float64(offsetY)
			if drawnY > maxDrawnY {
				maxDrawnY = drawnY
			}
			drawnX := (mx-transformedMinX)*float64(scale) + float64(offsetX)
			if drawnX > maxDrawnX {
				maxDrawnX = drawnX
			}
		}
	}

	// Only log if country changed
	if country != lastLoggedCountry {
		log.Printf("Southernmost Drawn Pixel: %f, BBox: %f", maxDrawnY, pixelMaxY)
		log.Printf("Easternmost Drawn Pixel: %f, BBox: %f", maxDrawnX, pixelMaxX)
		lastLoggedCountry = country
	}
}

var (
	lastLoggedCountry string
)
