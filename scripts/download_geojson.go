// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

//go:build ignore

// download_geojson fetches each country's simplified ADM0 boundary from the
// geoBoundaries API, truncates coordinates to 4 decimal places, drops the
// consecutive duplicate points that leaves, and writes <ISO code>.geojson.
//
// Run it from the repository root:
//
//	go run scripts/download_geojson.go                      # every country, into mapdata/
//	go run scripts/download_geojson.go -country LIE -out /tmp/geo
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// countryInfo is the part of a country_data.json entry this script needs.
type countryInfo struct {
	Name    string `json:"Name"`
	ISOCode string `json:"ISOCode"`
}

// geoJSON is a FeatureCollection as geoBoundaries serves it. Geometry
// coordinates stay raw so only Polygon and MultiPolygon get rewritten.
type geoJSON struct {
	Type     string `json:"type"`
	Features []struct {
		Type       string         `json:"type"`
		Properties map[string]any `json:"properties"`
		Geometry   struct {
			Type        string          `json:"type"`
			Coordinates json.RawMessage `json:"coordinates"`
		} `json:"geometry"`
	} `json:"features"`
}

// client bounds each request so a stalled download doesn't hang the run.
var client = &http.Client{Timeout: 2 * time.Minute}

// truncate cuts val to 4 decimal places (about 11 m of latitude).
func truncate(val float64) float64 {
	return math.Trunc(val*10000) / 10000
}

// simplifyRing truncates each point and drops points equal to the one before.
func simplifyRing(ring [][]float64) [][]float64 {
	var out [][]float64
	for _, pt := range ring {
		p := []float64{truncate(pt[0]), truncate(pt[1])}
		if n := len(out); n > 0 && out[n-1][0] == p[0] && out[n-1][1] == p[1] {
			continue
		}
		out = append(out, p)
	}
	return out
}

// simplifyPolygon applies simplifyRing to every ring of a polygon.
func simplifyPolygon(polygon [][][]float64) [][][]float64 {
	var out [][][]float64
	for _, ring := range polygon {
		out = append(out, simplifyRing(ring))
	}
	return out
}

// optimizeGeoJSON simplifies every Polygon and MultiPolygon in data and
// returns it re-encoded with two-space indentation. Features of other types,
// or with coordinates that don't parse, are kept as they are.
func optimizeGeoJSON(data []byte) ([]byte, error) {
	var fc geoJSON
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}

	for i := range fc.Features {
		g := &fc.Features[i].Geometry
		var simplified any
		switch g.Type {
		case "Polygon":
			var coords [][][]float64
			if err := json.Unmarshal(g.Coordinates, &coords); err != nil {
				continue
			}
			simplified = simplifyPolygon(coords)
		case "MultiPolygon":
			var coords [][][][]float64
			if err := json.Unmarshal(g.Coordinates, &coords); err != nil {
				continue
			}
			var polygons [][][][]float64
			for _, poly := range coords {
				polygons = append(polygons, simplifyPolygon(poly))
			}
			simplified = polygons
		default:
			continue
		}
		raw, err := json.Marshal(simplified)
		if err != nil {
			return nil, err
		}
		g.Coordinates = raw
	}

	return json.MarshalIndent(fc, "", "  ")
}

// get fetches url and fails on any status other than 200 OK.
func get(url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// downloadAndOptimize fetches one country's boundary and writes it to
// <outDir>/<ISO code>.geojson.
func downloadAndOptimize(target countryInfo, outDir string) error {
	apiURL := fmt.Sprintf("https://www.geoboundaries.org/api/current/gbOpen/%s/ADM0/", target.ISOCode)
	meta, err := get(apiURL)
	if err != nil {
		return fmt.Errorf("fetching API: %w", err)
	}
	var result struct {
		SimplifiedGeometryGeoJSON string `json:"simplifiedGeometryGeoJSON"`
	}
	if err := json.Unmarshal(meta, &result); err != nil {
		return fmt.Errorf("decoding API response: %w", err)
	}
	if result.SimplifiedGeometryGeoJSON == "" {
		return errors.New("API response has no simplifiedGeometryGeoJSON URL")
	}

	geoData, err := get(result.SimplifiedGeometryGeoJSON)
	if err != nil {
		return fmt.Errorf("fetching GeoJSON: %w", err)
	}
	optimized, err := optimizeGeoJSON(geoData)
	if err != nil {
		return fmt.Errorf("optimizing GeoJSON: %w", err)
	}

	destPath := filepath.Join(outDir, target.ISOCode+".geojson")
	if err := os.WriteFile(destPath, optimized, 0o644); err != nil {
		return fmt.Errorf("saving %s: %w", destPath, err)
	}
	return nil
}

func main() {
	only := flag.String("country", "", "download only the country with this ISO 3166-1 alpha-3 code")
	outDir := flag.String("out", "mapdata", "directory to write <ISO code>.geojson files to")
	dataPath := flag.String("data", "country_data.json", "path to country_data.json")
	flag.Parse()

	dataJSON, err := os.ReadFile(*dataPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading country data:", err)
		os.Exit(1)
	}
	var cc struct {
		Countries []countryInfo `json:"Countries"`
	}
	if err := json.Unmarshal(dataJSON, &cc); err != nil {
		fmt.Fprintln(os.Stderr, "Error decoding country data:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "Error creating output directory:", err)
		os.Exit(1)
	}

	matched, failed := 0, 0
	for _, c := range cc.Countries {
		if *only != "" && !strings.EqualFold(c.ISOCode, *only) {
			continue
		}
		matched++
		fmt.Printf("Downloading %s (%s)...", c.Name, c.ISOCode)
		if err := downloadAndOptimize(c, *outDir); err != nil {
			fmt.Println()
			fmt.Fprintf(os.Stderr, "Failed to process %s: %v\n", c.Name, err)
			failed++
			continue
		}
		fmt.Println("done")
	}

	if matched == 0 {
		fmt.Fprintf(os.Stderr, "No country with ISO code %q in %s\n", *only, *dataPath)
		os.Exit(1)
	}
	if failed > 0 {
		os.Exit(1)
	}
}
