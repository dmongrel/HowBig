package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

// LatLonToMercator converts geographic (longitude, latitude) coordinates into Mercator projection coordinates.
// It returns (x, y) coordinates normalized in the range [0.0, 1.0].
func LatLonToMercator(lon, lat float64) (x, y float64) {
	x = (lon + 180.0) / 360.0
	latRad := lat * math.Pi / 180.0
	mercN := math.Log(math.Tan((math.Pi / 4.0) + (latRad / 2.0)))
	y = 0.5 - (mercN / (2.0 * math.Pi)) // Y down for screen coordinates
	return x, y
}

// GetMercatorBoundingBox calculates the bounding box of a country in Mercator coordinates.
// It returns the min/max X and Y values, or an error if the data cannot be retrieved.
func GetMercatorBoundingBox(country string) (minX, minY, maxX, maxY float64, err error) {
	paths, err := FetchAndCacheGeoJSON(country, true)
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

// geoCache stores parsed geoJSON paths for efficient retrieval.
// cacheMu ensures thread-safe access to geoCache.
var (
	geoCache = make(map[string][][]fyne.Position)
	cacheMu  sync.RWMutex
)

// FetchAndCacheGeoJSON loads and caches parsed GeoJSON paths from the mapdata directory.
// The country parameter specifies the country name, and the singlePolyline parameter indicates if only the outer ring should be kept.
// It returns a slice of paths for each polygon, or an error if the file cannot be read or parsed.
func FetchAndCacheGeoJSON(country string, singlePolyline bool) ([][]fyne.Position, error) {

	fileName := getFileName(country)
	filePath := filepath.Join("mapdata", fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		filePath = filepath.Join("..", "mapdata", fileName)
	}

	cacheKey := country
	if singlePolyline {
		cacheKey += "_single"
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

	paths, err = convertGeoJSONToDisplayFormat(data, singlePolyline)
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
// It returns a slice of paths for each polygon, or an error if the parsing fails.
func convertGeoJSONToDisplayFormat(data []byte, singlePolyline bool) ([][]fyne.Position, error) {
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
		if f.Geometry.Type == "Polygon" {
			var coords [][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			if singlePolyline {
				coords = singlePolyLine(coords)
			}
			for _, ring := range coords {
				var path []fyne.Position
				for _, point := range ring {
					x := point[0]
					path = append(path, fyne.NewPos(float32(x), float32(point[1])))
				}
				allPaths = append(allPaths, path)
			}
		} else if f.Geometry.Type == "MultiPolygon" {
			var coords [][][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}

			for i, polygon := range coords {
				if singlePolyline {
					coords[i] = singlePolyLine(polygon)
				}
				for _, ring := range coords[i] {
					var path []fyne.Position
					for _, point := range ring {
						x := point[0]
						path = append(path, fyne.NewPos(float32(x), float32(point[1])))
					}
					allPaths = append(allPaths, path)
				}
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
