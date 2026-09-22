// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package mapdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderMissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := NewLoader(dir, nil, Options{}).Load("Y")
	if err == nil {
		t.Fatal("want an error")
	}
	if want := filepath.Join(dir, "Y.geojson"); !strings.Contains(err.Error(), want) {
		t.Errorf("error %q should name %q", err, want)
	}
}

func TestLoaderCache(t *testing.T) {
	dir := t.TempDir()
	geo := `{"features":[{"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}}]}`
	if err := os.WriteFile(filepath.Join(dir, "X.geojson"), []byte(geo), 0o644); err != nil {
		t.Fatal(err)
	}
	l := NewLoader(dir, nil, Options{SkipSmall: 25, PacificCenter: true})
	if got, want := l.cacheKey("X"), "X_single_skip25_pacific"; got != want {
		t.Errorf("cacheKey = %q, want %q", got, want)
	}
	if got, want := NewLoader(dir, nil, Options{}).cacheKey("X"), "X_single"; got != want {
		t.Errorf("cacheKey with no options = %q, want %q", got, want)
	}

	l = NewLoader(dir, nil, Options{})
	first, err := l.Load("X")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "X.geojson")); err != nil {
		t.Fatal(err)
	}
	second, err := l.Load("X")
	if err != nil || second != first {
		t.Errorf("second Load = %p, %v; want the cached %p", second, err, first)
	}
}
