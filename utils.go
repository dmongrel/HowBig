package main

import (
	"fmt"
	"image/color"
)

// ParseHexColor parses a hexadecimal color string (e.g., "#RRGGBB") and returns its color.NRGBA representation.
func ParseHexColor(s string) color.NRGBA {
	var r, g, b uint8
	if len(s) == 7 && s[0] == '#' {
		if _, err := fmt.Sscanf(s[1:], "%02x%02x%02x", &r, &g, &b); err == nil {
			return color.NRGBA{R: r, G: g, B: b, A: 255}
		}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}
