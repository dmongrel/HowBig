// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package country loads country_data.json: each country's name, ISO code and
// area, with lookup maps by name.
package country

import (
	"encoding/json"
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

	cc.Areas = make(map[string]float64, len(cc.Countries))
	cc.ISOCodes = make(map[string]string, len(cc.Countries))
	for _, c := range cc.Countries {
		cc.Areas[c.Name] = c.Area
		cc.ISOCodes[c.Name] = c.ISOCode
	}

	return cc, nil
}
