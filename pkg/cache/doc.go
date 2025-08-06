// Package cache provides a generic, thread-safe LRU (Least Recently Used) cache
// implementation for efficiently managing limited resources in memory.
//
// The cache automatically evicts the least recently used items when it reaches
// its configured capacity, making it ideal for scenarios where you need to
// cache data but want to prevent unbounded memory growth.
//
// # Key Features
//
//   - Generic implementation supporting any comparable key type and any value type
//   - Thread-safe operations with mutex-based synchronization
//   - Automatic LRU eviction when capacity is exceeded
//   - Optional eviction callbacks for resource cleanup (e.g., closing files, connections)
//   - Zero dependencies - uses only Go standard library
//   - O(1) operations for Get, Put, and Remove
//
// # Usage
//
// Create a cache with a specified capacity:
//
//	cache := cache.NewLRUCache[string, *sql.DB](100)
//
// Basic operations:
//
//	// Add items to cache
//	cache.Put("user:123", userData)
//	cache.Put("session:abc", sessionData)
//
//	// Retrieve items (marks as recently used)
//	data, found := cache.Get("user:123")
//	if found {
//		// Use data
//	}
//
//	// Remove specific items
//	removed, existed := cache.Remove("user:123")
//
//	// Clear all items
//	cache.Clear()
//
// # Resource Cleanup
//
// For resources that need cleanup when evicted (like database connections,
// file handles, or network connections), use eviction callbacks:
//
//	cache := cache.NewLRUCache[string, *sql.DB](10)
//	cache.SetEvictCallback(func(key string, db *sql.DB) {
//		db.Close() // Cleanup database connection
//	})
//
//	// Cache will automatically close connections when they're evicted
//	cache.Put("db1", db1)
//	cache.Put("db2", db2)
//
// # Thread Safety
//
// All operations are thread-safe and can be called concurrently from multiple
// goroutines:
//
//	// Safe to call from multiple goroutines
//	go cache.Put("key1", value1)
//	go cache.Put("key2", value2)
//	go cache.Get("key1")
//
// # Performance Characteristics
//
//   - Get: O(1) average case
//   - Put: O(1) average case
//   - Remove: O(1) average case
//   - Memory overhead: Approximately 3x the size of stored values due to
//     internal bookkeeping structures
//
// # Use Cases
//
// The cache is particularly useful for:
//
//   - Database connection pooling with automatic cleanup
//   - Caching expensive computation results
//   - Storing frequently accessed configuration data
//   - Managing limited resources like file handles or API clients
//   - Session data caching in web applications
//   - Template or compiled expression caching
//
// # Capacity Management
//
// When the cache reaches its capacity and a new item is added:
//
//  1. The least recently used item is identified
//  2. If an eviction callback is set, it's called with the item's key and value
//  3. The item is removed from the cache
//  4. The new item is added
//
// Items are considered "recently used" when they are:
//   - Retrieved with Get()
//   - Added or updated with Put()
//
// # Example: Database Connection Cache
//
//	type ConnectionCache struct {
//		cache *cache.LRUCache[string, *sql.DB]
//	}
//
//	func NewConnectionCache(maxConnections int) *ConnectionCache {
//		c := &ConnectionCache{
//			cache: cache.NewLRUCache[string, *sql.DB](maxConnections),
//		}
//
//		// Setup cleanup for evicted connections
//		c.cache.SetEvictCallback(func(dsn string, db *sql.DB) {
//			db.Close()
//		})
//
//		return c
//	}
//
//	func (c *ConnectionCache) GetConnection(dsn string) (*sql.DB, error) {
//		if db, found := c.cache.Get(dsn); found {
//			return db, nil
//		}
//
//		db, err := sql.Open("postgres", dsn)
//		if err != nil {
//			return nil, err
//		}
//
//		c.cache.Put(dsn, db)
//		return db, nil
//	}
package cache
