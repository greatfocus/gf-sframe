package cache

import (
	"time"

	gfcache "github.com/greatfocus/gf-cache/cache"
)

// Cache -
type Cache struct {
	Cache *gfcache.Cache
}

// New cache instance
func New(defaultExpiration, cleanupInterval int64) *Cache {
	// Create a cache with a default expiration time of 5 minutes, and which
	// pu rges expired items every 10 minutes
	return &Cache{
		Cache: gfcache.New(time.Duration(defaultExpiration), time.Duration(cleanupInterval)),
	}
}

// Set cache
func (c Cache) Set(key string, data interface{}, time time.Duration) {
	c.Cache.Set(key, data, time)
}

// Get cache
func (c Cache) Get(key string) (interface{}, bool) {
	return c.Cache.Get(key)
}

// Delete cache
func (c Cache) Delete(key string) {
	c.Cache.Delete(key)
}
