// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package mapcache is a small thread-safe LRU cache of parsed map data.
package mapcache

import (
	"slices"
	"sync"

	"HowBig/internal/geo"
)

// DefaultLimit is the number of countries the map loader keeps parsed.
const DefaultLimit = 5

// Cache implements a thread-safe LRU cache for GeoData. It is meant for a
// handful of entries: recency is a slice of keys, so each hit is O(limit).
type Cache struct {
	mu    sync.Mutex              // mu protects the fields below.
	items map[string]*geo.GeoData // items holds the cached values by key.
	keys  []string                // keys lists the cached keys, most recently used first.
	limit int                     // limit is the maximum number of items in the cache.
}

// New creates a Cache with the specified item limit. A limit below 1 still
// holds one entry.
func New(limit int) *Cache {
	return &Cache{
		items: make(map[string]*geo.GeoData),
		limit: limit,
	}
}

// Get retrieves an item from the cache.
func (c *Cache) Get(key string) (*geo.GeoData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.items[key]
	if ok {
		c.touch(key)
	}
	return value, ok
}

// Put adds an item to the cache, evicting the oldest if the limit is reached.
func (c *Cache) Put(key string, value *geo.GeoData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.items[key]; ok {
		c.items[key] = value
		c.touch(key)
		return
	}

	if n := len(c.keys); n > 0 && n >= c.limit {
		delete(c.items, c.keys[n-1])
		c.keys = c.keys[:n-1]
	}
	c.items[key] = value
	c.keys = slices.Insert(c.keys, 0, key)
}

// touch moves a cached key to the front of keys. c.mu must be held.
func (c *Cache) touch(key string) {
	i := slices.Index(c.keys, key)
	copy(c.keys[1:i+1], c.keys[:i])
	c.keys[0] = key
}
