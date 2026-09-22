// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

// Package mapcache is a small thread-safe LRU cache of parsed map data.
package mapcache

import (
	"container/list"
	"sync"

	"HowBig/internal/geo"
)

// DefaultLimit is the number of countries the map loader keeps parsed.
const DefaultLimit = 5

// entry is one cached item; the list holds these.
type entry struct {
	key   string       // key is the unique identifier for the cached item.
	value *geo.GeoData // value is the cached geographic data.
}

// Cache implements a thread-safe LRU cache for GeoData.
type Cache struct {
	items map[string]*list.Element // items maps keys to list elements for O(1) access.
	order *list.List               // order maintains the LRU order of elements.
	limit int                      // limit is the maximum number of items in the cache.
	mu    sync.Mutex               // mu protects the cache from concurrent access.
}

// New creates a Cache with the specified item limit. A limit below 1 still
// holds one entry.
func New(limit int) *Cache {
	return &Cache{
		items: make(map[string]*list.Element),
		order: list.New(),
		limit: limit,
	}
}

// Get retrieves an item from the cache.
func (c *Cache) Get(key string) (*geo.GeoData, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return el.Value.(*entry).value, true
	}
	return nil, false
}

// Put adds an item to the cache, evicting the oldest if the limit is reached.
func (c *Cache) Put(key string, value *geo.GeoData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		el.Value.(*entry).value = value
		return
	}

	if c.order.Len() >= c.limit {
		oldest := c.order.Back()
		if oldest != nil {
			c.order.Remove(oldest)
			delete(c.items, oldest.Value.(*entry).key)
		}
	}

	el := c.order.PushFront(&entry{key: key, value: value})
	c.items[key] = el
}
