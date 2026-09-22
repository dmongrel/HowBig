// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

//go:build ignore

// make_icon draws the HowBig app icon as a 1024x1024 SVG: two real country
// outlines from mapdata/ (by default the contiguous United States and
// Australia), projected and drawn at one shared scale the way the app draws
// them, on a dark rounded tile.
//
// Run it from the repository root:
//
//	go run scripts/make_icon.go > build/appicon.svg
//	go run scripts/make_icon.go -bare > build/appicon.icon/Assets/howbig.svg
//
// Then render build/appicon.svg to a 1024x1024 build/appicon.png (for example
// with headless Chrome) and run `wails3 task common:generate:icons` to rebuild
// build/windows/icon.ico and build/darwin/icons.icns. -bare leaves out the
// tile, for the macOS Icon Composer layer, which supplies its own.
package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"HowBig/internal/geo"
)

const (
	size    = 1024.0 // canvas edge
	tileIn  = 100.0  // tile inset from the canvas edge
	tileR   = 184.0  // tile corner radius
	fitW    = 700.0  // box the two countries together are fitted to
	fitH    = 660.0
	minRing = 4.0  // rings smaller than this, in canvas pixels, are dropped
	opacity = 0.6  // fill opacity; the app uses 0.5, a little brighter reads better small
	stroke  = 10.0 // outline width in canvas pixels
)

// Fill and outline colors are the app's defaults for the left and right side.
const (
	leftFill, leftStroke   = "#00FF00", "#00FFFF"
	rightFill, rightStroke = "#FF0000", "#FFCC00"
)

func main() {
	a := flag.String("a", "USA48", "ISO code of the left (green) country")
	b := flag.String("b", "AUS", "ISO code of the right (red) country")
	dir := flag.String("data", "mapdata", "directory holding <ISO>.geojson files")
	bare := flag.Bool("bare", false, "draw only the countries on a transparent canvas, without the tile")
	flag.Parse()

	art := countries(load(*dir, *a), load(*dir, *b))
	if *bare {
		fmt.Printf(`<svg xmlns="http://www.w3.org/2000/svg" width="%[1]g" height="%[1]g" viewBox="0 0 %[1]g %[1]g">`+"\n%s\n</svg>\n", size, art)
		return
	}
	fmt.Print(tile(art))
}

// tile wraps the artwork in the rounded background tile.
func tile(body string) string {
	edge := size - 2*tileIn
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%[1]g" height="%[1]g" viewBox="0 0 %[1]g %[1]g">`+"\n", size)
	fmt.Fprintf(&b, `<defs><linearGradient id="bg" x1="0" y1="0" x2="0" y2="1">`+
		`<stop offset="0" stop-color="#1B222C"/><stop offset="1" stop-color="#07090C"/></linearGradient>`+
		`<clipPath id="tile"><rect x="%[1]g" y="%[1]g" width="%[2]g" height="%[2]g" rx="%[3]g"/></clipPath></defs>`+"\n", tileIn, edge, tileR)
	fmt.Fprintf(&b, `<rect x="%[1]g" y="%[1]g" width="%[2]g" height="%[2]g" rx="%[3]g" fill="url(#bg)"/>`+"\n", tileIn, edge, tileR)
	b.WriteString(`<g clip-path="url(#tile)">` + body + "</g>\n")
	fmt.Fprintf(&b, `<rect x="%[1]g" y="%[1]g" width="%[2]g" height="%[2]g" rx="%[3]g" fill="none" stroke="#FFFFFF" stroke-opacity="0.14" stroke-width="4"/>`+"\n", tileIn+2, edge-4, tileR-2)
	b.WriteString("</svg>\n")
	return b.String()
}

// countries draws a and b at one shared scale. Like the app, each is centered
// on its own bounding box; the pair is then scaled and shifted so the two
// together fill the tile. a is filled first, then b, then both outlines,
// matching the app's draw order.
func countries(a, b *geo.GeoData) string {
	// Place both at scale 1 around the origin, then fit their union.
	ua, ub := centered(a.BoundingBox), centered(b.BoundingBox)
	minX, maxX := min(ua[0], ub[0]), max(ua[1], ub[1])
	minY, maxY := min(ua[2], ub[2]), max(ua[3], ub[3])
	scale := math.Min(fitW/(maxX-minX), fitH/(maxY-minY))
	cx := size/2 - (minX+maxX)/2*scale
	cy := size/2 - (minY+maxY)/2*scale

	pa, pb := paths(a, scale, cx, cy), paths(b, scale, cx, cy)
	var s strings.Builder
	fmt.Fprintf(&s, `<path d="%s" fill="%s" fill-opacity="%g" fill-rule="evenodd"/>`, pa, leftFill, opacity)
	fmt.Fprintf(&s, `<path d="%s" fill="%s" fill-opacity="%g" fill-rule="evenodd"/>`, pb, rightFill, opacity)
	fmt.Fprintf(&s, `<path d="%s" fill="none" stroke="%s" stroke-width="%g" stroke-linejoin="round"/>`, pa, leftStroke, stroke)
	fmt.Fprintf(&s, `<path d="%s" fill="none" stroke="%s" stroke-width="%g" stroke-linejoin="round"/>`, pb, rightStroke, stroke)
	return s.String()
}

// centered returns bb's extent [minX, maxX, minY, maxY] once bb is centered
// on the origin.
func centered(bb geo.BoundingBox) [4]float64 {
	return [4]float64{-bb.Width / 2, bb.Width / 2, -bb.Height / 2, bb.Height / 2}
}

func load(dir, iso string) *geo.GeoData {
	data, err := os.ReadFile(filepath.Join(dir, iso+".geojson"))
	if err != nil {
		log.Fatal(err)
	}
	g, err := geo.ParseGeoJSON(data, 25, true)
	if err != nil {
		log.Fatalf("%s: %v", iso, err)
	}
	return visible(g)
}

// visible keeps the rings at least 2% as wide or tall as the country's largest
// ring and recomputes the bounding box from them. Remote islands, such as
// Australia's Heard Island, would otherwise stretch the box that the pair is
// fitted and centered to, while being far too small to see in an icon.
func visible(g *geo.GeoData) *geo.GeoData {
	var largest float64
	for _, ring := range g.Paths {
		x0, x1, y0, y1 := extent(ring)
		largest = max(largest, x1-x0, y1-y0)
	}
	out := &geo.GeoData{}
	bb := geo.BoundingBox{MinX: math.Inf(1), MaxX: math.Inf(-1), MinY: math.Inf(1), MaxY: math.Inf(-1)}
	for _, ring := range g.Paths {
		x0, x1, y0, y1 := extent(ring)
		if max(x1-x0, y1-y0) < 0.02*largest {
			continue
		}
		out.Paths = append(out.Paths, ring)
		bb.MinX, bb.MaxX = min(bb.MinX, x0), max(bb.MaxX, x1)
		bb.MinY, bb.MaxY = min(bb.MinY, y0), max(bb.MaxY, y1)
	}
	bb.Width, bb.Height = bb.MaxX-bb.MinX, bb.MaxY-bb.MinY
	out.BoundingBox = bb
	return out
}

// paths turns g's projected rings into one SVG path string, centered on
// (cx, cy) at the given scale, dropping rings too small to see.
func paths(g *geo.GeoData, scale, cx, cy float64) string {
	bb := g.BoundingBox
	ox := cx - (bb.MinX+bb.Width/2)*scale
	oy := cy - (bb.MinY+bb.Height/2)*scale
	var d strings.Builder
	for _, ring := range g.Paths {
		minX, maxX, minY, maxY := extent(ring)
		if (maxX-minX)*scale < minRing && (maxY-minY)*scale < minRing {
			continue
		}
		// Whole pixels are plenty at 1024px; skip points that round onto the last one.
		var lastX, lastY float64
		for i, p := range ring {
			x, y := math.Round(p.X*scale+ox), math.Round(p.Y*scale+oy)
			if i > 0 && x == lastX && y == lastY {
				continue
			}
			cmd := "L"
			if i == 0 {
				cmd = "M"
			}
			fmt.Fprintf(&d, "%s%g %g", cmd, x, y)
			lastX, lastY = x, y
		}
		d.WriteString("Z")
	}
	return d.String()
}

// extent returns the bounds of one ring.
func extent(ring []geo.Point) (minX, maxX, minY, maxY float64) {
	minX, maxX, minY, maxY = math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range ring {
		minX, maxX = min(minX, p.X), max(maxX, p.X)
		minY, maxY = min(minY, p.Y), max(maxY, p.Y)
	}
	return minX, maxX, minY, maxY
}
