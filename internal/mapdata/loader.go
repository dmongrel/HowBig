// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package mapdata loads a country's GeoJSON file from the map data directory,
// parses it with package geo and caches the result.
package mapdata

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"HowBig/internal/country"
	"HowBig/internal/geo"
	"HowBig/internal/mapcache"
)

// Options controls how GeoJSON is turned into paths. They match the
// skip_small and enable_pacific_center settings.
type Options struct {
	SkipSmall     int  // SkipSmall drops rings with this many points or fewer.
	PacificCenter bool // PacificCenter shifts countries that span the antimeridian onto 0 to 360 degrees.
}

// Loader reads, parses and caches the GeoJSON for one country at a time.
// It is safe for concurrent use.
type Loader struct {
	dir       string              // dir is the map data directory, already resolved (R4).
	opts      Options             // opts applies to every load.
	countries *country.Collection // countries maps names to ISO codes; it may be nil.
	cache     *mapcache.Cache     // cache holds the most recently loaded countries.
}

// NewLoader creates a Loader that reads <ISO code>.geojson files from dir.
// countries may be nil, in which case file names come from the country name
// with its spaces removed.
func NewLoader(dir string, countries *country.Collection, opts Options) *Loader {
	return &Loader{
		dir:       dir,
		opts:      opts,
		countries: countries,
		cache:     mapcache.New(mapcache.DefaultLimit),
	}
}

// Load returns the parsed paths and Mercator bounding box for the named
// country, from the cache when it was loaded recently. A relative dir that
// lacks the file falls back to ../dir when the file exists there.
func (l *Loader) Load(name string) (*geo.GeoData, error) {
	key := l.cacheKey(name)
	if data, ok := l.cache.Get(key); ok {
		return data, nil
	}

	fileName := l.fileName(name)
	filePath := filepath.Join(l.dir, fileName)
	// Fall back to ../dir only when that file exists, so a missing
	// file's error names the primary path rather than the fallback.
	if _, err := os.Stat(filePath); errors.Is(err, fs.ErrNotExist) && !filepath.IsAbs(l.dir) {
		fallback := filepath.Join("..", l.dir, fileName)
		if _, err := os.Stat(fallback); err == nil {
			filePath = fallback
		}
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	data, err := geo.ParseGeoJSON(raw, l.opts.SkipSmall, l.opts.PacificCenter)
	if err != nil {
		return nil, err
	}
	l.cache.Put(key, data)
	return data, nil
}

// cacheKey is the country name tagged with the options it was parsed with.
func (l *Loader) cacheKey(name string) string {
	key := name + "_single"
	if l.opts.SkipSmall > 0 {
		key += "_skip" + strconv.Itoa(l.opts.SkipSmall)
	}
	if l.opts.PacificCenter {
		key += "_pacific"
	}
	return key
}

// fileName resolves the GeoJSON file name for a country: its ISO code when
// the collection knows it, otherwise the name with spaces removed.
func (l *Loader) fileName(name string) string {
	if l.countries != nil {
		if iso, ok := l.countries.ISOCodes[name]; ok {
			return iso + ".geojson"
		}
	}
	return strings.ReplaceAll(name, " ", "") + ".geojson"
}
