package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
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

// GeoCache implements a thread-safe LRU cache for GeoData.
type GeoCache struct {
	items map[string]*list.Element
	order *list.List
	limit int
	mu    sync.Mutex
}

// NewGeoCache creates a new GeoCache with the specified item limit.
func NewGeoCache(limit int) *GeoCache {
	return &GeoCache{
		items: make(map[string]*list.Element),
		order: list.New(),
		limit: limit,
	}
}

// Get retrieves an item from the cache.
func (c *GeoCache) Get(key string) (*GeoData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*cacheEntry).value, true
	}
	return nil, false
}

// Put adds an item to the cache, evicting the oldest if the limit is reached.
func (c *GeoCache) Put(key string, value *GeoData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*cacheEntry).value = value
		return
	}

	if c.order.Len() >= c.limit {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*cacheEntry).key)
		}
	}

	newEntry := &cacheEntry{key: key, value: value}
	el := c.order.PushFront(newEntry)
	c.items[key] = el
}

// FetchAndCacheGeoJSON loads and caches parsed GeoJSON paths and their overall geographic bounding box from the mapdata directory.
func FetchAndCacheGeoJSON(country string, singlePolyline bool, skipSmall int, enablePacificCenter bool, mapDataPath string, cache *GeoCache, countryCollection *CountryCollection) (*GeoData, error) {

	fileName := getFileName(country, countryCollection)
	filePath := filepath.Join(mapDataPath, fileName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		filePath = filepath.Join("..", mapDataPath, fileName)
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

	if cache != nil {
		if data, ok := cache.Get(cacheKey); ok {
			return data, nil
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	geoData, err := convertGeoJSONToDisplayFormat(data, singlePolyline, skipSmall, enablePacificCenter)
	if err != nil {
		return nil, err
	}

	if cache != nil {
		cache.Put(cacheKey, geoData)
	}

	return geoData, nil
}

// getFileName resolves the correct GeoJSON filename for a given country name.
func getFileName(name string, cc *CountryCollection) string {
	if cc != nil {
		return cc.CompactNames[name]
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

				path := slices.Collect(func(yield func(Point) bool) {
					for _, pt := range ring {
						mx, my := LatLonToMercator(pt[0], pt[1])
						if !yield(Point{X: mx, Y: my}) {
							return
						}
					}
				})
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
