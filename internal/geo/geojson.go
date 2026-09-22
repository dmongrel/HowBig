// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package geo

import "encoding/json"

// ParseGeoJSON parses raw GeoJSON bytes into Mercator paths ready to scale
// and draw, with the overall Mercator bounding box of all features.
// Only Polygon and MultiPolygon features are read, and only each polygon's
// outer ring is kept; holes and interior rings are dropped. Rings with
// skipSmall or fewer coordinates are skipped. With pacificCenter set, a
// feature that spans the antimeridian is shifted onto 0 to 360 degrees first.
// It returns an error only if data is not valid JSON.
func ParseGeoJSON(data []byte, skipSmall int, pacificCenter bool) (*GeoData, error) {
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
		var g geometry
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

		if pacificCenter && needsPacificCentering(g) {
			g = applyPacificCentering(g)
		}

		for _, polygon := range g.Coordinates {
			for _, ring := range outerRing(polygon) {
				if len(ring) <= skipSmall {
					continue
				}

				path := make([]Point, 0, len(ring))
				for _, pt := range ring {
					if len(pt) < 2 {
						continue // a malformed position with no latitude
					}
					mx, my := latLonToMercator(pt[0], pt[1])
					path = append(path, Point{X: mx, Y: my})
				}
				if len(path) > 0 {
					allPaths = append(allPaths, path)
				}
			}
		}
	}

	geoData := &GeoData{
		Paths: allPaths,
	}
	geoData.updateBoundingBox()

	return geoData, nil
}

// outerRing keeps only a polygon's first (outer) ring, discarding holes and
// interior rings. A polygon with no rings is returned as is.
func outerRing(rings [][][]float64) [][][]float64 {
	if len(rings) > 0 {
		return rings[0:1]
	}
	return rings
}
