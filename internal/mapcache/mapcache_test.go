// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package mapcache

import (
	"testing"

	"HowBig/internal/geo"
)

func TestCacheEviction(t *testing.T) {
	a, b, c, d := &geo.GeoData{}, &geo.GeoData{}, &geo.GeoData{}, &geo.GeoData{}

	t.Run("evicts least recently used", func(t *testing.T) {
		cache := New(2)
		cache.Put("a", a)
		cache.Put("b", b)
		if _, ok := cache.Get("a"); !ok { // a becomes most recent
			t.Fatal("a missing")
		}
		cache.Put("c", c) // evicts b
		if _, ok := cache.Get("b"); ok {
			t.Error("b should have been evicted")
		}
		for _, k := range []string{"a", "c"} {
			if _, ok := cache.Get(k); !ok {
				t.Errorf("%s should still be cached", k)
			}
		}
	})

	t.Run("re-put replaces value and refreshes recency", func(t *testing.T) {
		cache := New(2)
		cache.Put("a", a)
		cache.Put("b", b)
		cache.Put("a", d) // a is now most recent and holds d
		cache.Put("c", c) // evicts b
		if got, ok := cache.Get("a"); !ok || got != d {
			t.Errorf("Get(a) = %p, %v; want %p, true", got, ok, d)
		}
		if _, ok := cache.Get("b"); ok {
			t.Error("b should have been evicted")
		}
	})

	t.Run("limit zero still holds one entry", func(t *testing.T) {
		cache := New(0)
		cache.Put("a", a)
		if _, ok := cache.Get("a"); !ok {
			t.Error("a should be cached")
		}
		cache.Put("b", b)
		if _, ok := cache.Get("a"); ok {
			t.Error("a should have been evicted")
		}
		if _, ok := cache.Get("b"); !ok {
			t.Error("b should be cached")
		}
	})

	t.Run("miss", func(t *testing.T) {
		if got, ok := New(1).Get("x"); ok || got != nil {
			t.Errorf("Get on empty cache = %v, %v", got, ok)
		}
	})
}
