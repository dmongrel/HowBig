package main

import (
	"container/list"
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
	CacheLimit  = 5
)

// cacheEntry is a helper struct for the LRU cache.
type cacheEntry struct {
	key   string
	value *GeoData
}

var (
	geoCache   = make(map[string]*list.Element)
	cacheOrder = list.New()
	cacheMu    sync.Mutex
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

	cacheMu.Lock()
	defer cacheMu.Unlock()

	if el, ok := geoCache[cacheKey]; ok {
		cacheOrder.MoveToFront(el)
		return el.Value.(*cacheEntry).value, nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	geoData, err := convertGeoJSONToDisplayFormat(data, singlePolyline, skipSmall, enablePacificCenter)
	if err != nil {
		return nil, err
	}

	if cacheOrder.Len() >= CacheLimit {
		oldest := cacheOrder.Back()
		if oldest != nil {
			cacheOrder.Remove(oldest)
			delete(geoCache, oldest.Value.(*cacheEntry).key)
		}
	}

	newEntry := &cacheEntry{key: cacheKey, value: geoData}
	el := cacheOrder.PushFront(newEntry)
	geoCache[cacheKey] = el

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
	geoData.UpdateBoundingBox()

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
