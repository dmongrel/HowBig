package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CountryInfo holds basic information about a country.
type CountryInfo struct {
	Name        string  `json:"Name"`
	CompactName string  `json:"CompactName"`
	ISOCode     string  `json:"ISOCode"`
	Area        float64 `json:"Area"`
	Flip_Y      bool    `json:"flip_y"`
	Rotate      int     `json:"rotate"`
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
func NewCountryCollection() *CountryCollection {
	path := filepath.Join("mapdata", "country_data.json")
	data, err := os.ReadFile(path)
	if err != nil {
		panic("country_data.json not found: " + err.Error())
	}
	cc := &CountryCollection{}
	if err := json.Unmarshal(data, cc); err != nil {
		panic("error unmarshaling country_data.json: " + err.Error())
	}
	return cc
}
