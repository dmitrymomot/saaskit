package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Storage implements Fiber's session storage interface using Redis.
// Wraps go-redis client for simplified key-value operations.
type Storage struct {
	db            redis.UniversalClient
	scanBatchSize int64
}

// NewStorage creates a Redis storage wrapper compatible with Fiber's session interface.
// Uses default scan batch size of 1000 for efficient key scanning.
func NewStorage(redisClient redis.UniversalClient) *Storage {
	return &Storage{
		db:            redisClient,
		scanBatchSize: 1000,
	}
}

// NewStorageWithConfig creates a Redis storage with custom configuration.
func NewStorageWithConfig(redisClient redis.UniversalClient, cfg Config) *Storage {
	return &Storage{
		db:            redisClient,
		scanBatchSize: int64(cfg.ScanBatchSize),
	}
}

// Get returns nil for empty keys and missing values (redis.Nil becomes nil).
func (s *Storage) Get(key string) ([]byte, error) {
	if len(key) <= 0 {
		return nil, nil
	}
	val, err := s.db.Get(context.Background(), key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

// Set stores key-value with expiration. Zero duration means no expiration.
func (s *Storage) Set(key string, val []byte, exp time.Duration) error {
	if len(key) <= 0 || len(val) <= 0 {
		return nil
	}
	return s.db.Set(context.Background(), key, val, exp).Err()
}

// Delete removes a key. Empty keys are ignored.
func (s *Storage) Delete(key string) error {
	if len(key) <= 0 {
		return nil
	}
	return s.db.Del(context.Background(), key).Err()
}

// Reset clears ALL keys using FLUSHDB. CAUTION: affects entire Redis database.
func (s *Storage) Reset() error {
	return s.db.FlushDB(context.Background()).Err()
}

// Close terminates the Redis connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// Conn returns the underlying Redis client for advanced operations.
func (s *Storage) Conn() redis.UniversalClient {
	return s.db
}

// Keys returns all database keys using SCAN to avoid blocking Redis.
func (s *Storage) Keys() ([][]byte, error) {
	// Pre-allocate with reasonable capacity to reduce allocations
	keys := make([][]byte, 0, 1000)
	var cursor uint64
	var err error

	for {
		var batch []string

		if batch, cursor, err = s.db.Scan(context.Background(), cursor, "*", s.scanBatchSize).Result(); err != nil {
			return nil, err
		}

		// Grow slice if needed to accommodate new batch
		if cap(keys) < len(keys)+len(batch) {
			newKeys := make([][]byte, len(keys), (cap(keys)+len(batch))*2)
			copy(newKeys, keys)
			keys = newKeys
		}

		for _, key := range batch {
			keys = append(keys, []byte(key))
		}

		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil, nil
	}

	return keys, nil
}
