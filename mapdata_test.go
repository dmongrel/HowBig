package main

import (
	"testing"
)

// TestGetBoundingBox validates that GetBoundingBox returns a valid bounding box.
func TestGetBoundingBox(t *testing.T) {
	// Use a country that we know exists
	bbox, err := GetBoundingBox("Afghanistan")
	if err != nil {
		t.Fatalf("GetBoundingBox failed: %v", err)
	}

	// Basic validation: Min should be <= Max
	if bbox.MinX > bbox.MaxX {
		t.Errorf("MinX (%f) > MaxX (%f)", bbox.MinX, bbox.MaxX)
	}
	if bbox.MinY > bbox.MaxY {
		t.Errorf("MinY (%f) > MaxY (%f)", bbox.MinY, bbox.MaxY)
	}
}
