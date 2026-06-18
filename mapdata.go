// Attribution: geoBoundaries data is used under CC-BY 4.0 license.
// See https://www.geoboundaries.org/ for details.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

// geoCache stores parsed geoJSON paths for efficient retrieval.
// cacheMu ensures thread-safe access to geoCache.
var (
	geoCache = make(map[string][][]fyne.Position)
	cacheMu  sync.RWMutex
)

// getCachedGeoJSON loads and caches parsed geoJSON from the mapdata directory.
func getCachedGeoJSON(country string, singlePolyline bool) ([][]fyne.Position, error) {
	if country == "Barbados" {
		singlePolyline = false
	}
	fileName := getFileName(country)
	filePath := filepath.Join("mapdata", fileName)

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
	return strings.ReplaceAll(name, " ", "") + ".gjson"
}

// fetchGeoJSON reads geoJSON from the mapdata directory.
func fetchGeoJSON(country string) ([]byte, error) {
	fileName := getFileName(country)
	filePath := filepath.Join("mapdata", fileName)
	return os.ReadFile(filePath)
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
			for i, ring := range coords {
				if singlePolyline && i > 0 {
					continue
				}
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

			for _, polygon := range coords {
				for i, ring := range polygon {
					if singlePolyline && i > 0 {
						continue
					}
					var path []fyne.Position
					for _, point := range ring {
						path = append(path, fyne.NewPos(float32(point[1]), float32(point[0])))
					}
					allPaths = append(allPaths, path)
				}
			}
		}
	}
	return allPaths, nil
}

type BoundingBox struct {
	MinX, MinY, MaxX, MaxY float32
}

// GetBoundingBox calculates the bounding box. It attempts to read pre-calculated values
// from GeoJSON properties, falling back to recalculation if necessary.
func GetBoundingBox(country string) (BoundingBox, error) {
	data, err := fetchGeoJSON(country)
	if err == nil {
		var fc struct {
			Features []struct {
				Properties struct {
					BoundingBox []float32 `json:"boundingBox"`
				} `json:"properties"`
			} `json:"features"`
		}
		if err := json.Unmarshal(data, &fc); err == nil && len(fc.Features) > 0 && len(fc.Features[0].Properties.BoundingBox) == 4 {
			bbox := fc.Features[0].Properties.BoundingBox
			return BoundingBox{
				MinX: bbox[0],
				MinY: bbox[1],
				MaxX: bbox[2],
				MaxY: bbox[3],
			}, nil
		}
	}

	paths, err := getCachedGeoJSON(country, true)
	if err != nil {
		return BoundingBox{}, err
	}

	var bb BoundingBox
	first := true
	for _, path := range paths {
		for _, pos := range path {
			if first {
				bb.MinX = pos.X
				bb.MinY = pos.Y
				bb.MaxX = pos.X
				bb.MaxY = pos.Y
				first = false
				continue
			}
			if pos.X < bb.MinX {
				bb.MinX = pos.X
			}
			if pos.Y < bb.MinY {
				bb.MinY = pos.Y
			}
			if pos.X > bb.MaxX {
				bb.MaxX = pos.X
			}
			if pos.Y > bb.MaxY {
				bb.MaxY = pos.Y
			}
		}
	}
	return bb, nil
}
