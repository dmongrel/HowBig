// Attribution: geoBoundaries data is used under CC-BY 4.0 license.
// See https://www.geoboundaries.org/ for details.

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
)

var (
	geoCache = make(map[string][][]fyne.Position)
	cacheMu  sync.RWMutex
)

// getCachedGeoJSON loads and caches parsed geoJSON from the mapdata directory.
func getCachedGeoJSON(country string, allowDownsampling bool, skipSmall bool) ([][]fyne.Position, error) {
	fileName := getFileName(country)
	filePath := filepath.Join("mapdata", fileName)

	skipFactor := 1
	if allowDownsampling {
		fileInfo, err := os.Stat(filePath)
		if err == nil {
			sizeMB := fileInfo.Size() / (1024 * 1024)
			if sizeMB >= 100 {
				skipFactor = (2 + int((sizeMB-100)/50)) * 2
			}
		}
	}

	cacheKey := country
	if skipFactor > 1 {
		cacheKey += "_skip" + strconv.Itoa(skipFactor)
	}
	if skipSmall {
		cacheKey += "_small"
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

	paths, err = convertGeoJSONToDisplayFormat(data, skipFactor, skipSmall)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	geoCache[cacheKey] = paths
	cacheMu.Unlock()
	return paths, nil
}

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
func convertGeoJSONToDisplayFormat(data []byte, skipFactor int, skipSmall bool) ([][]fyne.Position, error) {
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
			for _, ring := range coords {
				var path []fyne.Position
				for i, point := range ring {
					if skipFactor > 1 && i%skipFactor != 0 {
						continue
					}
					path = append(path, fyne.NewPos(float32(point[1]), float32(point[0])))
				}
				allPaths = append(allPaths, path)
			}
		} else if f.Geometry.Type == "MultiPolygon" {
			var coords [][][][]float64
			if err := json.Unmarshal(f.Geometry.Coordinates, &coords); err != nil {
				continue
			}
			for _, polygon := range coords {
				totalPoints := 0
				for _, ring := range polygon {
					totalPoints += len(ring)
				}
				if skipSmall && totalPoints < 50 {
					continue
				}
				for _, ring := range polygon {
					var path []fyne.Position
					for i, point := range ring {
						if skipFactor > 1 && i%skipFactor != 0 {
							continue
						}
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
					X  float32 `json:"x"`
					Y  float32 `json:"y"`
					X1 float32 `json:"x1"`
					X2 float32 `json:"x2"`
				} `json:"properties"`
			} `json:"features"`
		}
		if err := json.Unmarshal(data, &fc); err == nil && len(fc.Features) > 0 && (fc.Features[0].Properties.X != 0 || fc.Features[0].Properties.X1 != 0) {
			return BoundingBox{
				MinX: fc.Features[0].Properties.X,
				MinY: fc.Features[0].Properties.Y,
				MaxX: fc.Features[0].Properties.X1,
				MaxY: fc.Features[0].Properties.X2,
			}, nil
		}
	}

	paths, err := getCachedGeoJSON(country, false, false)
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
