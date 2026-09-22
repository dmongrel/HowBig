// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"fmt"
	"slices"
)

// CountryService exposes the country list and area lookups to the frontend.
type CountryService struct {
	countries *CountryCollection // countries is nil when country data failed to load.
	loadErr   string             // loadErr is the load failure message, or "".
}

// NewCountryService wraps a loaded collection. A non-nil loadErr is kept for
// LoadError instead of stopping startup; cc may then be nil.
func NewCountryService(cc *CountryCollection, loadErr error) *CountryService {
	s := &CountryService{countries: cc}
	if loadErr != nil {
		s.loadErr = fmt.Sprintf("failed to load country data: %v", loadErr)
	}
	return s
}

// List returns every country in file order. It is empty, never nil, when no
// country data is loaded.
func (s *CountryService) List() []CountryInfo {
	if s.countries == nil || len(s.countries.Countries) == 0 {
		return []CountryInfo{}
	}
	return slices.Clone(s.countries.Countries)
}

// Area returns a country's surface area in square miles, or 0 if it is unknown.
func (s *CountryService) Area(name string) float64 {
	return areaOf(s.countries, name)
}

// LoadError returns why country data failed to load, or "" if it loaded.
func (s *CountryService) LoadError() string {
	return s.loadErr
}

// areaOf looks up a country's area in cc, which may be nil.
func areaOf(cc *CountryCollection, name string) float64 {
	if cc == nil {
		return 0
	}
	return cc.Areas[name]
}
