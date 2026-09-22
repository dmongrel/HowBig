// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package country loads country_data.json: each country's name, ISO code and
// area, with lookup maps by name.
package country

import (
	"encoding/json"
	"maps"
	"os"
)

// Info holds basic information about a country.
type Info struct {
	Name    string  `json:"Name"`    // Name is the official name of the country.
	ISOCode string  `json:"ISOCode"` // ISOCode is the ISO 3166-1 alpha-3 code of the country.
	Area    float64 `json:"Area"`    // Area is the surface area of the country in square miles.
}

// Collection holds a collection of Info objects and lookup maps for quick access.
type Collection struct {
	Countries []Info             // Countries is a slice of all country information objects.
	Areas     map[string]float64 // Areas maps country names to their surface area.
	ISOCodes  map[string]string  // ISOCodes maps country names to their ISO 3166-1 alpha-3 code.
}

// Load creates and initializes a new Collection from the specified JSON file.
func Load(path string) (*Collection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cc := &Collection{}
	if err := json.Unmarshal(data, cc); err != nil {
		return nil, err
	}

	// Populate the lookup maps for O(1) access using maps.Collect and custom iterators.
	cc.Areas = maps.Collect(func(yield func(string, float64) bool) {
		for _, c := range cc.Countries {
			if !yield(c.Name, c.Area) {
				return
			}
		}
	})

	cc.ISOCodes = maps.Collect(func(yield func(string, string) bool) {
		for _, c := range cc.Countries {
			if !yield(c.Name, c.ISOCode) {
				return
			}
		}
	})

	return cc, nil
}
