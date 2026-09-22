// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package mapcache

import (
	"strconv"
	"sync"
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

	t.Run("hit in the middle keeps the rest in order", func(t *testing.T) {
		cache := New(3)
		cache.Put("a", a)
		cache.Put("b", b)
		cache.Put("c", c)
		cache.Get("b")    // order is now b, c, a
		cache.Put("d", d) // evicts a
		cache.Put("e", a) // evicts c
		for k, want := range map[string]bool{"a": false, "b": true, "c": false, "d": true, "e": true} {
			if _, ok := cache.Get(k); ok != want {
				t.Errorf("Get(%s) ok = %v, want %v", k, ok, want)
			}
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

// TestCacheConcurrent gives the race detector concurrent Gets and Puts that
// hit, miss, update and evict.
func TestCacheConcurrent(t *testing.T) {
	cache := New(DefaultLimit)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 200 {
				key := strconv.Itoa((g + i) % (2 * DefaultLimit))
				if _, ok := cache.Get(key); !ok {
					cache.Put(key, &geo.GeoData{})
				}
			}
		})
	}
	wg.Wait()

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.keys) != len(cache.items) || len(cache.keys) > DefaultLimit {
		t.Errorf("len(keys) = %d, len(items) = %d; want equal and at most %d", len(cache.keys), len(cache.items), DefaultLimit)
	}
	for _, k := range cache.keys {
		if _, ok := cache.items[k]; !ok {
			t.Errorf("key %q is in keys but not in items", k)
		}
	}
}
