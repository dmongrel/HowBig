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

// GeoData holds the parsed geographic paths and the overall bounding box for a country.
type GeoData struct {
	Paths       [][]fyne.Position
	BoundingBox BoundingBox
}

var (
	geoCache = make(map[string]*GeoData)
	cacheMu  sync.RWMutex
)

// FetchAndCacheGeoJSON loads and caches parsed GeoJSON paths and their overall geographic bounding box from the mapdata directory.
// The country parameter specifies the country name, and the singlePolyline parameter indicates if only the outer ring should be kept.
// The skipSmall parameter indicates the maximum number of coordinates a polygon ring can have to be skipped.
// The enablePacificCenter parameter indicates if the coordinates should be normalized to Pacific-centering if IDL crossing is detected.
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

	var allPaths [][]fyne.Position
	overallMinX, overallMaxX := math.MaxFloat64, -math.MaxFloat64
	overallMinY, overallMaxY := math.MaxFloat64, -math.MaxFloat64
	bboxInitialized := false

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

		// Calculate bounding box for the current geometry in Mercator space
		minX, maxX := math.MaxFloat64, -math.MaxFloat64
		minY, maxY := math.MaxFloat64, -math.MaxFloat64
		geometryHasCoords := false

		if len(g.Coordinates) > 0 {
			for _, polygon := range g.Coordinates {
				for _, ring := range polygon {
					for _, coord := range ring {
						lon := coord[0]
						lat := coord[1]

						mx, my := LatLonToMercator(lon, lat)

						minX = min(minX, mx)
						maxX = max(maxX, mx)
						minY = min(minY, my)
						maxY = max(maxY, my)
						geometryHasCoords = true
					}
				}
			}
			if geometryHasCoords {
				// Update overall bounding box
				if !bboxInitialized {
					overallMinX, overallMaxX = minX, maxX
					overallMinY, overallMaxY = minY, maxY
					bboxInitialized = true
				} else {
					overallMinX = min(overallMinX, minX)
					overallMaxX = max(overallMaxX, maxX)
					overallMinY = min(overallMinY, minY)
					overallMaxY = max(overallMaxY, maxY)
				}
			}
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
					mx, my := LatLonToMercator(pt[0], pt[1])
					path = append(path, fyne.NewPos(float32(mx), float32(my)))
				}
				allPaths = append(allPaths, path)
			}
		}
	}

	// Calculation of overall bounding box:
	overallBBox := BoundingBox{}
	if bboxInitialized {
		overallBBox = BoundingBox{
			MinX: overallMinX,
			MaxX: overallMaxX,
			MinY: overallMinY,
			MaxY: overallMaxY,
		}
	}
	log.Printf("Overall BBox set: %+v", overallBBox)

	return &GeoData{
		Paths:       allPaths,
		BoundingBox: overallBBox,
	}, nil
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
func LogSouthernmostPixels(country string, paths [][]fyne.Position, scale float32, offsetX, offsetY float32, transformedBBox BoundingBox) {
	pixelMinY := float64(offsetY) + (transformedBBox.MinY-transformedBBox.MinY)*float64(scale) // This is just offsetY
	pixelMaxY := float64(offsetY) + (transformedBBox.MaxY-transformedBBox.MinY)*float64(scale)
	pixelMinX := float64(offsetX) + (transformedBBox.MinX-transformedBBox.MinX)*float64(scale) // This is just offsetX
	pixelMaxX := float64(offsetX) + (transformedBBox.MaxX-transformedBBox.MinX)*float64(scale)

	width := pixelMaxX - pixelMinX
	height := pixelMaxY - pixelMinY

	maxDrawnY := -math.MaxFloat64
	maxDrawnX := -math.MaxFloat64
	for _, path := range paths {
		for _, pos := range path {
			drawnY := (float64(pos.Y)-transformedBBox.MinY)*float64(scale) + float64(offsetY)
			if drawnY > maxDrawnY {
				maxDrawnY = drawnY
			}
			drawnX := (float64(pos.X)-transformedBBox.MinX)*float64(scale) + float64(offsetX)
			if drawnX > maxDrawnX {
				maxDrawnX = drawnX
			}
		}
	}

	// Only log if country changed
	if country != lastLoggedCountry {
		log.Printf("Country: %s, BBox Pixel Size: W=%.2f, H=%.2f", country, width, height)
		log.Printf("Southernmost Drawn Pixel: %f, BBox: %f", maxDrawnY, pixelMaxY)
		log.Printf("Easternmost Drawn Pixel: %f, BBox: %f", maxDrawnX, pixelMaxX)
		lastLoggedCountry = country
	}
}

var (
	lastLoggedCountry string
)
