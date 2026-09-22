// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package mapdata

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderPathFallback(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "work")
	if err := os.MkdirAll(filepath.Join(root, "maps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	geo := `{"features":[{"geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,0]]]}}]}`
	if err := os.WriteFile(filepath.Join(root, "maps", "X.geojson"), []byte(geo), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(work)
	l := NewLoader("maps", nil, Options{})

	t.Run("parent fallback is used when the file exists there", func(t *testing.T) {
		if _, err := l.Load("X"); err != nil {
			t.Fatalf("fallback not used: %v", err)
		}
	})
	t.Run("a missing file's error names the primary path", func(t *testing.T) {
		_, err := l.Load("Y")
		if err == nil {
			t.Fatal("want an error")
		}
		want := filepath.Join("maps", "Y.geojson")
		if msg := err.Error(); !strings.Contains(msg, want) || strings.Contains(msg, "..") {
			t.Errorf("error %q should name %q and not the .. fallback", msg, want)
		}
	})
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
