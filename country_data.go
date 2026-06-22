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

// CountryCollection holds a collection of CountryInfo objects.
type CountryCollection struct {
	Countries []CountryInfo
	Areas     map[string]float64
	ISOCodes  map[string]string
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
