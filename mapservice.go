// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"fmt"
	"math"

	"HowBig/internal/country"
	"HowBig/internal/geo"
	"HowBig/internal/mapdata"
	"HowBig/internal/settings"
)

// Side values for CountryLayout.Side.
const (
	SideLeft  = "left"
	SideRight = "right"
)

// Rect is an axis-aligned rectangle in map-box pixels.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// CountryLayout is one selected country, ready to draw.
//
// Colors are not included: the frontend takes them from SettingsService by
// Side (left_color/left_border_color or right_color/right_border_color).
type CountryLayout struct {
	Name  string `json:"name"`  // Name is the country name.
	Side  string `json:"side"`  // Side is "left" or "right": which list the country was picked from.
	Order int    `json:"order"` // Order is the draw order: 0 is the larger, filled first; 1 is filled on top.
	// Paths holds one closed polygon per entry as flat x,y pairs
	// ([x0, y0, x1, y1, ...]) in map-box pixels, scaled, centered and rounded
	// to 0.1 px. It is empty, never null, when the country failed to load.
	Paths [][]float64 `json:"paths"`
	Box   *Rect       `json:"box,omitempty"`   // Box is the pixel bounding box, set only when debug_show_boundary is on.
	Error string      `json:"error,omitempty"` // Error is why the country's map data failed to load, or "".
}

// MapLayout is everything the frontend needs to draw the map box.
type MapLayout struct {
	Scale     float64         `json:"scale"`     // Scale is the shared scale in pixels per Mercator unit.
	Countries []CountryLayout `json:"countries"` // Countries holds the selected countries in draw order, never null.
}

// MapService lays out the selected countries for drawing.
type MapService struct {
	settings  *settings.Settings
	countries *country.Collection // countries may be nil if country data failed to load.
	loader    *mapdata.Loader     // loader reads and caches each country's GeoJSON.
}

// NewMapService creates a MapService that reads GeoJSON from mapDataPath,
// which R4 has already resolved, with the skip_small and
// enable_pacific_center options from cfg.
func NewMapService(cfg *settings.Settings, countries *country.Collection, mapDataPath string) *MapService {
	return &MapService{
		settings:  cfg,
		countries: countries,
		loader: mapdata.NewLoader(mapDataPath, countries, mapdata.Options{
			SkipSmall:     cfg.SkipSmall,
			PacificCenter: cfg.EnablePacificCenter,
		}),
	}
}

// loadedCountry is a selected country with its map data, if it loaded.
type loadedCountry struct {
	side string
	mc   geo.MapCountry
	data *geo.GeoData
	err  error
}

// Layout scales the countries picked in the left and right lists ("" for none)
// to a shared scale that fits the map box of width by height pixels, and
// returns them in draw order with pixel-space paths. Each country is centered
// in the box on its own bounding box. A country whose map data fails to load
// is still listed, with Error set and no paths.
func (s *MapService) Layout(left, right string, width, height float64) MapLayout {
	l := s.load(left, SideLeft)
	r := s.load(right, SideRight)
	scale, larger, _ := geo.ScaleAndOrder(l.mc, r.mc, width, height)

	order := []loadedCountry{r, l}
	if larger == left {
		order = []loadedCountry{l, r}
	}

	out := MapLayout{Scale: scale, Countries: []CountryLayout{}}
	for _, c := range order {
		if c.mc.Name == "" {
			continue
		}
		out.Countries = append(out.Countries, s.project(c, len(out.Countries), scale, width, height))
	}
	return out
}

// load fetches a selected country's map data; name "" means nothing is selected.
func (s *MapService) load(name, side string) loadedCountry {
	c := loadedCountry{side: side, mc: geo.MapCountry{Name: name, Area: areaOf(s.countries, name)}}
	if name == "" {
		return c
	}
	data, err := s.loader.Load(name)
	if err != nil {
		c.err = err
		return c
	}
	bb := data.BoundingBox
	c.mc.BB = &bb
	c.data = data
	return c
}

// project turns a loaded country into pixel-space paths at scale, centered in
// the width by height map box.
func (s *MapService) project(c loadedCountry, order int, scale, width, height float64) CountryLayout {
	cl := CountryLayout{Name: c.mc.Name, Side: c.side, Order: order, Paths: [][]float64{}}
	if c.err != nil {
		cl.Error = fmt.Sprintf("Error loading %s: %v", c.mc.Name, c.err)
		return cl
	}
	if len(c.data.Paths) == 0 {
		return cl
	}

	bb := c.data.BoundingBox
	pixelWidth, pixelHeight := bb.Width*scale, bb.Height*scale
	offsetX, offsetY := (width-pixelWidth)/2, (height-pixelHeight)/2
	if s.settings.DebugShowBoundary {
		cl.Box = &Rect{X: round1(offsetX), Y: round1(offsetY), Width: round1(pixelWidth), Height: round1(pixelHeight)}
	}

	for _, path := range c.data.Paths {
		if len(path) < 3 {
			continue
		}
		flat := make([]float64, 0, 2*len(path))
		for _, p := range path {
			x := round1((p.X-bb.MinX)*scale + offsetX)
			y := round1((p.Y-bb.MinY)*scale + offsetY)
			// Rounding collapses runs of nearby points; drop the repeats.
			if n := len(flat); n >= 2 && flat[n-2] == x && flat[n-1] == y {
				continue
			}
			flat = append(flat, x, y)
		}
		cl.Paths = append(cl.Paths, flat)
	}
	return cl
}

// round1 rounds v to the nearest 0.1.
func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
