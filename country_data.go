package main

import (
	"encoding/json"
	"os"
)

// CountryInfo holds basic information about a country.
type CountryInfo struct {
	Name        string  `json:"Name"`        // Name is the official name of the country.
	CompactName string  `json:"CompactName"` // CompactName is the filename for the country's GeoJSON.
	ISOCode     string  `json:"ISOCode"`     // ISOCode is the ISO 3166-1 alpha-3 code of the country.
	Area        float64 `json:"Area"`        // Area is the surface area of the country in square miles.
}

// CountryCollection holds a collection of CountryInfo objects.
type CountryCollection struct {
	Countries []CountryInfo
}

// SaveToJSON saves the CountryCollection to a JSON file.
func (cc *CountryCollection) SaveToJSON(filename string) error {
	data, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// NewCountryCollection creates and initializes a new CountryCollection from country_data.json.
func NewCountryCollection() (*CountryCollection, error) {
	path := "country_data.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cc := &CountryCollection{}
	if err := json.Unmarshal(data, cc); err != nil {
		return nil, err
	}
	return cc, nil
}
