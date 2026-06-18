package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type CountryInfo struct {
	Name        string
	CompactName string
	ISOCode     string
	Area        float64
}

type CountryCollection struct {
	Countries []CountryInfo
}

func (cc *CountryCollection) SaveToJSON(filename string) error {
	data, err := json.MarshalIndent(cc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

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
