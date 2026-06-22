// Copyright © 2026 Joel L. Caesar
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://gnu.org>.

package main

import (
	"encoding/json"
	"maps"
	"os"
)

// CountryInfo holds basic information about a country.
type CountryInfo struct {
	Name    string  `json:"Name"`    // Name is the official name of the country.
	ISOCode string  `json:"ISOCode"` // ISOCode is the ISO 3166-1 alpha-3 code of the country.
	Area    float64 `json:"Area"`    // Area is the surface area of the country in square miles.
}

// CountryCollection holds a collection of CountryInfo objects and lookup maps for quick access.
type CountryCollection struct {
	Countries []CountryInfo      // Countries is a slice of all country information objects.
	Areas     map[string]float64 // Areas maps country names to their surface area.
	ISOCodes  map[string]string  // ISOCodes maps country names to their ISO 3166-1 alpha-3 code.
}

// SaveToJSON saves the CountryCollection to a JSON file.
func (cc *CountryCollection) SaveToJSON(filename string) error {
	data, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// NewCountryCollection creates and initializes a new CountryCollection from the specified JSON file.
func NewCountryCollection(path string) (*CountryCollection, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cc := &CountryCollection{}
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
