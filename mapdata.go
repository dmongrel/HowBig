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

// LatLonToMercator projection mapping: (lon, lat) -> (0.0 to 1.0) space
func LatLonToMercator(lon, lat float64) (x, y float64) {
	x = (lon + 180.0) / 360.0
	latRad := lat * math.Pi / 180.0
	mercN := math.Log(math.Tan((math.Pi / 4.0) + (latRad / 2.0)))
	y = 0.5 - (mercN / (2.0 * math.Pi)) // Y down for screen coordinates
	return x, y
}

// GetMercatorBoundingBox calculates the bounding box in Mercator coordinates.
func GetMercatorBoundingBox(country string) (minX, minY, maxX, maxY float64, err error) {
	paths, err := GetCachedGeoJSON(country, true)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	minX, minY = math.MaxFloat64, math.MaxFloat64
	maxX, maxY = -math.MaxFloat64, -math.MaxFloat64
	for _, path := range paths {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			if mx < minX {
				minX = mx
			}
			if mx > maxX {
				maxX = mx
			}
			if my < minY {
				minY = my
			}
			if my > maxY {
				maxY = my
			}
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

// GetCachedGeoJSON loads and caches parsed geoJSON from the mapdata directory.
func GetCachedGeoJSON(country string, singlePolyline bool) ([][]fyne.Position, error) {

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

// getFileName returns the file name for the given country name.
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

// convertGeoJSONToDisplayFormat converts raw JSON into a format suitable for displaying.
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
					path = append(path, fyne.NewPos(float32(point[0]), float32(point[1])))
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
						path = append(path, fyne.NewPos(float32(point[0]), float32(point[1])))
					}
					allPaths = append(allPaths, path)
				}
			}
		}
	}
	return allPaths, nil
}

func singlePolyLine(rings [][][]float64) [][][]float64 {
	if len(rings) > 0 {
		return rings[0:1]
	}
	return rings
}
