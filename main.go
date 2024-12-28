package main

import (
	"fmt"
	"sync"
	"time"
)

// CacheItem represents an individual item in the cache
type CacheItem struct {
	Value      interface{}
	Expiration int64 // Unix timestamp for expiration
}

// SieveCache represents the sieve cache
type SieveCache struct {
	store         sync.Map
	ttl           time.Duration
	sieveFilter   func(key string, value interface{}) bool // Custom filter logic
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewSieveCache creates a new instance of SieveCache
func NewSieveCache(ttl time.Duration, sieveFilter func(key string, value interface{}) bool) *SieveCache {
	cache := &SieveCache{
		ttl:         ttl,
		sieveFilter: sieveFilter,
		stopCleanup: make(chan struct{}),
	}
	cache.cleanupTicker = time.NewTicker(ttl / 2) // Cleanup runs at half the TTL interval
	go cache.cleanupExpiredItems()
	return cache
}

// Set adds a new item to the cache if it passes the sieve filter
func (c *SieveCache) Set(key string, value interface{}) {
	if c.sieveFilter(key, value) {
		c.store.Store(key, CacheItem{
			Value:      value,
			Expiration: time.Now().Add(c.ttl).Unix(),
		})
	}
}

// Get retrieves an item from the cache if it exists and is not expired
func (c *SieveCache) Get(key string) (interface{}, bool) {
	item, exists := c.store.Load(key)
	if !exists {
		return nil, false
	}

	cacheItem := item.(CacheItem)
	if time.Now().Unix() > cacheItem.Expiration {
		c.store.Delete(key)
		return nil, false
	}
	return cacheItem.Value, true
}

// Delete removes an item from the cache
func (c *SieveCache) Delete(key string) {
	c.store.Delete(key)
}

// cleanupExpiredItems removes expired items from the cache periodically
func (c *SieveCache) cleanupExpiredItems() {
	for {
		select {
		case <-c.cleanupTicker.C:
			c.store.Range(func(key, value interface{}) bool {
				cacheItem := value.(CacheItem)
				if time.Now().Unix() > cacheItem.Expiration {
					c.store.Delete(key)
				}
				return true
			})
		case <-c.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (c *SieveCache) Stop() {
	close(c.stopCleanup)
	c.cleanupTicker.Stop()
}

func main() {
	// Custom sieve filter: only cache strings with length > 3
	filter := func(key string, value interface{}) bool {
		str, ok := value.(string)
		return ok && len(str) > 3
	}

	// Create a sieve cache with a 10-second TTL
	cache := NewSieveCache(10*time.Second, filter)
	defer cache.Stop()

	// Add items to the cache
	cache.Set("key1", "sh")           // Won't be cached (length <= 3)
	cache.Set("key2", "longerString") // Will be cached

	// Retrieve items from the cache
	if value, ok := cache.Get("key2"); ok {
		fmt.Println("Found key2:", value)
	} else {
		fmt.Println("key2 not found")
	}

	if value, ok := cache.Get("key1"); ok {
		fmt.Println("Found key1:", value)
	} else {
		fmt.Println("key1 not found")
	}

	// Wait for 11 seconds to test expiration
	time.Sleep(11 * time.Second)
	if _, ok := cache.Get("key2"); !ok {
		fmt.Println("key2 expired")
	}
}
