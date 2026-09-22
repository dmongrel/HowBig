// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"image/color"
	"testing"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{7, "7"},
		{999, "999"},
		{1000, "1,000"},
		{1234567, "1,234,567"},
		{3677647, "3,677,647"},
		{999.5, "1,000"},
		{1234.5, "1,234"}, // %.0f rounds half to even
		{2.5, "2"},
		{-1234, "-1,234"},
		// Characterization of a bug: the minus sign counts as a digit, so a
		// negative number whose digit count is a multiple of three gets "-,".
		{-123, "-,123"},
		{-123456, "-,123,456"},
		{-0.4, "-0"},
	}
	for _, tt := range tests {
		if got := formatNumber(tt.in); got != tt.want {
			t.Errorf("formatNumber(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseHexColor(t *testing.T) {
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	tests := []struct {
		in   string
		want color.NRGBA
	}{
		{"#000000", color.NRGBA{A: 255}},
		{"#FF0000", color.NRGBA{R: 255, A: 255}},
		{"#00ff00", color.NRGBA{G: 255, A: 255}},
		{"#FFCC00", color.NRGBA{R: 255, G: 204, A: 255}},
		{"#12aB9f", color.NRGBA{R: 0x12, G: 0xab, B: 0x9f, A: 255}},
		// Everything that isn't exactly #RRGGBB falls back to opaque white.
		{"", white},
		{"FF0000", white},
		{"#FFF", white},
		{"#FF000080", white},
		{"#GGGGGG", white},
		{"#12GG56", white},
		{" #FF000", white},
	}
	for _, tt := range tests {
		if got := ParseHexColor(tt.in); got != tt.want {
			t.Errorf("ParseHexColor(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
