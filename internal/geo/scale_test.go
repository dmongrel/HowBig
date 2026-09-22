// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package geo

import "testing"

func TestFitScale(t *testing.T) {
	tests := []struct {
		name          string
		bb            BoundingBox
		width, height float64
		want          float64
	}{
		{"width limited", BoundingBox{Width: 10, Height: 5}, 104, 104, 10},
		{"height limited", BoundingBox{Width: 5, Height: 10}, 204, 104, 10},
		{"margin of 4 comes off both axes", BoundingBox{Width: 1, Height: 1}, 54, 34, 30},
		{"zero width box", BoundingBox{Width: 0, Height: 10}, 104, 104, 1},
		{"zero height box", BoundingBox{Width: 10, Height: 0}, 104, 104, 1},
		{"width below 4", BoundingBox{Width: 1, Height: 1}, 3.9, 104, 1},
		{"height below 4", BoundingBox{Width: 1, Height: 1}, 104, 3.9, 1},
		{"exactly 4 fits to zero", BoundingBox{Width: 1, Height: 1}, 4, 4, 0},
		{"negative height", BoundingBox{Width: 1, Height: 1}, 104, -10, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fitScale(tt.bb, tt.width, tt.height); got != tt.want {
				t.Errorf("fitScale(%+v, %v, %v) = %v, want %v", tt.bb, tt.width, tt.height, got, tt.want)
			}
		})
	}
}

func TestScaleAndOrder(t *testing.T) {
	// The drawable area is 104x104 unless stated, so 100x100 after the fit margin.
	const w, h = 104.0, 104.0
	box := func(width, height float64) *BoundingBox {
		return &BoundingBox{Width: width, Height: height}
	}

	tests := []struct {
		name          string
		active, other MapCountry
		height        float64
		wantScale     float64
		wantLarger    string
		wantSmaller   string
	}{
		{
			name:      "nothing selected",
			wantScale: 1,
		},
		{
			name:      "only active selected",
			active:    MapCountry{Name: "A", Area: 10, BB: box(10, 20)},
			wantScale: 5, wantLarger: "A",
		},
		{
			name:      "only other selected",
			other:     MapCountry{Name: "B", Area: 10, BB: box(25, 5)},
			wantScale: 4, wantLarger: "B",
		},
		{
			name:      "only active selected and it failed to load",
			active:    MapCountry{Name: "A", Area: 10},
			wantScale: 1, wantLarger: "A",
		},
		{
			name:      "both selected, active larger by area",
			active:    MapCountry{Name: "A", Area: 1000, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 100, BB: box(5, 5)},
			wantScale: 10, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "both selected, other larger by area",
			active:    MapCountry{Name: "A", Area: 100, BB: box(5, 5)},
			other:     MapCountry{Name: "B", Area: 1000, BB: box(20, 10)},
			wantScale: 5, wantLarger: "B", wantSmaller: "A",
		},
		{
			name:      "both selected, equal area goes to active",
			active:    MapCountry{Name: "A", Area: 500, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 500, BB: box(5, 5)},
			wantScale: 10, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "swap: other is smaller by area but wider in pixels",
			active:    MapCountry{Name: "A", Area: 1000, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 100, BB: box(20, 2)},
			wantScale: 5, wantLarger: "B", wantSmaller: "A",
		},
		{
			name:      "swap: other is smaller by area but taller in pixels",
			active:    MapCountry{Name: "A", Area: 1000, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 100, BB: box(2, 40)},
			wantScale: 2.5, wantLarger: "B", wantSmaller: "A",
		},
		{
			// Characterization: the edge check is >=, so a smaller country that exactly
			// touches the edge still swaps. The scale is unchanged but the order flips.
			name:      "swap: other exactly touches the edge",
			active:    MapCountry{Name: "A", Area: 1000, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 100, BB: box(10, 5)},
			wantScale: 10, wantLarger: "B", wantSmaller: "A",
		},
		{
			name:      "swap: active is smaller by area but wider in pixels",
			active:    MapCountry{Name: "A", Area: 100, BB: box(20, 2)},
			other:     MapCountry{Name: "B", Area: 1000, BB: box(10, 10)},
			wantScale: 5, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "swap: active is smaller by area but taller in pixels",
			active:    MapCountry{Name: "A", Area: 100, BB: box(2, 40)},
			other:     MapCountry{Name: "B", Area: 1000, BB: box(10, 10)},
			wantScale: 2.5, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "swap: active exactly touches the edge",
			active:    MapCountry{Name: "A", Area: 100, BB: box(10, 5)},
			other:     MapCountry{Name: "B", Area: 1000, BB: box(10, 10)},
			wantScale: 10, wantLarger: "A", wantSmaller: "B",
		},
		{
			// The larger-by-area country is wider in pixels, but the smaller one
			// fits, so there is no swap.
			name:      "no swap: smaller by area fits at the larger's scale",
			active:    MapCountry{Name: "A", Area: 1000, BB: box(10, 10)},
			other:     MapCountry{Name: "B", Area: 100, BB: box(9, 9)},
			wantScale: 10, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "both selected, both failed to load",
			active:    MapCountry{Name: "A", Area: 100},
			other:     MapCountry{Name: "B", Area: 1000},
			wantScale: 1, wantLarger: "A", wantSmaller: "B",
		},
		{
			name:      "both selected, active failed to load",
			active:    MapCountry{Name: "A", Area: 1000},
			other:     MapCountry{Name: "B", Area: 100, BB: box(50, 10)},
			wantScale: 2, wantLarger: "B", wantSmaller: "A",
		},
		{
			name:      "both selected, other failed to load",
			active:    MapCountry{Name: "A", Area: 100, BB: box(50, 10)},
			other:     MapCountry{Name: "B", Area: 1000},
			wantScale: 2, wantLarger: "A", wantSmaller: "B",
		},
		{
			// Too small to draw: the area ordering is skipped and the input order is kept.
			name:      "both selected, drawable area too small",
			active:    MapCountry{Name: "A", Area: 100, BB: box(5, 5)},
			other:     MapCountry{Name: "B", Area: 1000, BB: box(10, 10)},
			height:    3,
			wantScale: 1, wantLarger: "A", wantSmaller: "B",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			height := h
			if tt.height != 0 {
				height = tt.height
			}
			scale, larger, smaller := ScaleAndOrder(tt.active, tt.other, w, height)
			if scale != tt.wantScale || larger != tt.wantLarger || smaller != tt.wantSmaller {
				t.Errorf("ScaleAndOrder() = (%v, %q, %q), want (%v, %q, %q)",
					scale, larger, smaller, tt.wantScale, tt.wantLarger, tt.wantSmaller)
			}
		})
	}
}
