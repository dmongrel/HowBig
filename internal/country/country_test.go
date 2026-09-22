// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package country

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRealFile(t *testing.T) {
	cc, err := Load(filepath.Join("..", "..", "country_data.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cc.Countries) != 200 || len(cc.Areas) != 200 || len(cc.ISOCodes) != 200 {
		t.Errorf("got %d countries, %d areas, %d ISO codes; want 200 each", len(cc.Countries), len(cc.Areas), len(cc.ISOCodes))
	}
	if first := cc.Countries[0]; first != (Info{Name: "Afghanistan", ISOCode: "AFG", Area: 252072}) {
		t.Errorf("first country = %+v", first)
	}
	checks := []struct {
		name string
		iso  string
		area float64
	}{
		{"Fiji", "FJI", 7055},
		{"United States", "USA", 3677647},
	}
	for _, c := range checks {
		if cc.ISOCodes[c.name] != c.iso || cc.Areas[c.name] != c.area {
			t.Errorf("%s = %q / %v, want %q / %v", c.name, cc.ISOCodes[c.name], cc.Areas[c.name], c.iso, c.area)
		}
	}
}

func TestLoadEdgeCases(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "countries.json")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("duplicate names: last one wins in the lookup maps", func(t *testing.T) {
		cc, err := Load(write(t, `{"Countries": [
			{"Name": "X", "ISOCode": "AAA", "Area": 1},
			{"Name": "X", "ISOCode": "BBB", "Area": 2}
		]}`))
		if err != nil {
			t.Fatal(err)
		}
		if len(cc.Countries) != 2 || cc.ISOCodes["X"] != "BBB" || cc.Areas["X"] != 2 {
			t.Errorf("got %+v", cc)
		}
	})

	t.Run("no countries gives empty non-nil maps", func(t *testing.T) {
		cc, err := Load(write(t, `{}`))
		if err != nil {
			t.Fatal(err)
		}
		if cc.Areas == nil || cc.ISOCodes == nil || len(cc.Areas) != 0 {
			t.Errorf("got %+v", cc)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		if _, err := Load(write(t, `{"Countries": [`)); err == nil {
			t.Error("expected an error")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("err = %v, want not-exist", err)
		}
	})
}
