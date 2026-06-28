// SPDX-FileCopyrightText: 2026 Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
)

type CountryInfo struct {
	Name        string  `json:"Name"`
	CompactName string  `json:"CompactName"`
	ISOCode     string  `json:"ISOCode"`
	Area        float64 `json:"Area"`
}

type CountryCollection struct {
	Countries []CountryInfo `json:"Countries"`
}

type GeoJSON struct {
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

func truncate(val float64) float64 {
	return math.Trunc(val*10000) / 10000
}

func processPoint(p []float64) []float64 {
	return []float64{truncate(p[0]), truncate(p[1])}
}

func removeDuplicates(ring [][]float64) [][]float64 {
	if len(ring) <= 1 {
		return ring
	}
	var newRing [][]float64
	newRing = append(newRing, ring[0])
	for i := 1; i < len(ring); i++ {
		if ring[i][0] != ring[i-1][0] || ring[i][1] != ring[i-1][1] {
			newRing = append(newRing, ring[i])
		}
	}
	return newRing
}

func processPolygon(polygon [][][]float64) [][][]float64 {
	var newPolygon [][][]float64
	for _, ring := range polygon {
		var newRing [][]float64
		for _, pt := range ring {
			newRing = append(newRing, processPoint(pt))
		}
		newPolygon = append(newPolygon, newRing)
	}
	return newPolygon
}

func removeDuplicatesFromPolygon(polygon [][][]float64) [][][]float64 {
	var newPolygon [][][]float64
	for _, ring := range polygon {
		newPolygon = append(newPolygon, removeDuplicates(ring))
	}
	return newPolygon
}

func optimizeGeoJSON(data []byte) ([]byte, error) {
	var fc GeoJSON
	if err := json.Unmarshal(data, &fc); err != nil {
		return nil, err
	}

	for i := range fc.Features {
		geomType := fc.Features[i].Geometry.Type
		if geomType == "Polygon" {
			var coords [][][]float64
			if err := json.Unmarshal(fc.Features[i].Geometry.Coordinates, &coords); err != nil {
				continue
			}
			newCoords := processPolygon(coords)
			fc.Features[i].Geometry.Coordinates, _ = json.Marshal(newCoords)
		} else if geomType == "MultiPolygon" {
			var coords [][][][]float64
			if err := json.Unmarshal(fc.Features[i].Geometry.Coordinates, &coords); err != nil {
				continue
			}
			var newCoords [][][][]float64
			for _, poly := range coords {
				newCoords = append(newCoords, processPolygon(poly))
			}
			fc.Features[i].Geometry.Coordinates, _ = json.Marshal(newCoords)
		}
	}

	fmt.Print("...Simplification")

	for i := range fc.Features {
		geomType := fc.Features[i].Geometry.Type
		if geomType == "Polygon" {
			var coords [][][]float64
			if err := json.Unmarshal(fc.Features[i].Geometry.Coordinates, &coords); err != nil {
				continue
			}
			newCoords := removeDuplicatesFromPolygon(coords)
			fc.Features[i].Geometry.Coordinates, _ = json.Marshal(newCoords)
		} else if geomType == "MultiPolygon" {
			var coords [][][][]float64
			if err := json.Unmarshal(fc.Features[i].Geometry.Coordinates, &coords); err != nil {
				continue
			}
			var newCoords [][][][]float64
			for _, poly := range coords {
				newCoords = append(newCoords, removeDuplicatesFromPolygon(poly))
			}
			fc.Features[i].Geometry.Coordinates, _ = json.Marshal(newCoords)
		}
	}

	fmt.Print("...Duplicates Removed")

	return json.MarshalIndent(fc, "", "  ")
}

func downloadAndOptimize(target CountryInfo) error {
	fmt.Printf("Downloading %s", target.Name)

	apiURL := fmt.Sprintf("https://www.geoboundaries.org/api/current/gbOpen/%s/ADM0/", target.ISOCode)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error fetching API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println()
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result struct {
		SimplifiedGeometryGeoJSON string `json:"simplifiedGeometryGeoJSON"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Println()
		return fmt.Errorf("error decoding API response: %w", err)
	}

	geoURL := result.SimplifiedGeometryGeoJSON
	if geoURL == "" {
		fmt.Println()
		return fmt.Errorf("no GeoJSON download URL found")
	}

	geoResp, err := http.Get(geoURL)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error fetching GeoJSON: %w", err)
	}
	defer geoResp.Body.Close()

	geoData, err := io.ReadAll(geoResp.Body)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error reading GeoJSON data: %w", err)
	}

	optimizedData, err := optimizeGeoJSON(geoData)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error optimizing GeoJSON: %w", err)
	}

	destPath := filepath.Join("mapdata", target.CompactName)
	err = os.WriteFile(destPath, optimizedData, 0644)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("error saving %s: %w", destPath, err)
	}

	fmt.Println("...Done")
	return nil
}

func main() {
	// Load country_data.json
	dataJSON, err := os.ReadFile("country_data.json")
	if err != nil {
		fmt.Println("Error reading country_data.json:", err)
		return
	}

	var cc CountryCollection
	if err := json.Unmarshal(dataJSON, &cc); err != nil {
		fmt.Println("Error decoding country_data.json:", err)
		return
	}

	_ = os.MkdirAll("mapdata", 0755)

	for _, country := range cc.Countries {
		if err := downloadAndOptimize(country); err != nil {
			fmt.Printf("Failed to process %s: %v\n", country.Name, err)
		}
	}
}
