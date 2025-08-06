# cache

Thread-safe generic LRU (Least Recently Used) cache implementation with configurable capacity and eviction callbacks.

## Features

- Generic implementation supporting any comparable key type and any value type
- Thread-safe concurrent access with proper locking
- Configurable capacity with automatic eviction of least recently used items
- Optional eviction callbacks for cleanup operations
- Zero external dependencies using standard library only

## Installation

```bash
go get github.com/dmitrymomot/saaskit
```

## Usage

```go
package main

import (
    "fmt"
    "github.com/dmitrymomot/saaskit/pkg/cache"
)

func main() {
    // Create cache with capacity of 3 items
    c := cache.NewLRUCache[string, int](3)
    
    // Add items
    c.Put("user:1", 100)
    c.Put("user:2", 200)
    c.Put("user:3", 300)
    
    // Get item (moves to front)
    if value, ok := c.Get("user:1"); ok {
        fmt.Printf("Found: %d\n", value) // Found: 100
    }
    
    // Add fourth item (evicts least recently used)
    c.Put("user:4", 400)
    
    // user:2 was evicted (least recently used)
    if _, ok := c.Get("user:2"); !ok {
        fmt.Println("user:2 was evicted")
    }
}
```

## Common Operations

### Basic Cache Operations

```go
c := cache.NewLRUCache[string, string](10)

// Put items
oldValue, existed := c.Put("key", "value")

// Get items
value, found := c.Get("key")

// Remove items
removedValue, existed := c.Remove("key")

// Check size
size := c.Len()

// Clear all items
c.Clear()
```

### Eviction Callbacks

```go
c := cache.NewLRUCache[string, *Resource](5)

// Set cleanup callback
c.SetEvictCallback(func(key string, resource *Resource) {
    fmt.Printf("Evicting %s\n", key)
    resource.Close() // cleanup resources
})

// Items will trigger callback when evicted
c.Put("resource1", &Resource{})
```

### Concurrent Usage

```go
c := cache.NewLRUCache[int, string](100)

// Safe for concurrent access
go func() {
    c.Put(1, "value1")
}()

go func() {
    if val, ok := c.Get(1); ok {
        fmt.Println(val)
    }
}()
```

## API Documentation

### Types

```go
type LRUCache[K comparable, V any] struct {
    // Contains filtered or unexported fields
}
```

### Functions

```go
// NewLRUCache creates a new LRU cache with specified capacity
func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V]

// SetEvictCallback sets callback for when items are evicted
func (c *LRUCache[K, V]) SetEvictCallback(fn func(key K, value V))

// Get retrieves value and marks as recently used
func (c *LRUCache[K, V]) Get(key K) (V, bool)

// Put adds/updates value, returns previous value if existed
func (c *LRUCache[K, V]) Put(key K, value V) (V, bool)

// Remove removes item and returns its value
func (c *LRUCache[K, V]) Remove(key K) (V, bool)

// Len returns current number of items
func (c *LRUCache[K, V]) Len() int

// Clear removes all items (calls eviction callback)
func (c *LRUCache[K, V]) Clear()
```

For detailed API documentation:

```bash
go doc -all ./pkg/cache
```

Or visit [pkg.go.dev](https://pkg.go.dev/github.com/dmitrymomot/saaskit/pkg/cache) for online documentation.

## Notes

- Cache capacity must be positive, otherwise NewLRUCache panics
- All operations are O(1) time complexity
- Memory usage is O(capacity) for the cache structure
- Eviction callbacks are called synchronously during eviction operations