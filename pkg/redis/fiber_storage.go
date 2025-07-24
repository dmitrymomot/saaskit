package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Storage is a Redis-based storage implementation that provides a key-value
// store compatible with various caching and session management interfaces,
// including Fiber's session storage interface.
// It wraps the go-redis client to provide a simplified storage API.
type Storage struct {
	db redis.UniversalClient
}

// NewStorage creates a new Redis storage instance using the provided Redis client.
// It accepts any implementation of redis.UniversalClient (such as redis.Client,
// redis.ClusterClient, etc.) to allow flexibility in Redis deployment configurations.
//
// Parameters:
//   - redisClient: A configured Redis client that will handle the actual Redis operations.
//
// Returns:
//   - *Storage: A new Storage instance ready to use for data operations.
func NewStorage(redisClient redis.UniversalClient) *Storage {
	return &Storage{
		db: redisClient,
	}
}

// Get retrieves a value from Redis by its key.
//
// Parameters:
//   - key: The key to retrieve the value for.
//
// Returns:
//   - []byte: The value as a byte slice if found.
//   - error: An error if the retrieval operation failed, or nil if the key doesn't exist.
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

// Set stores a key-value pair in Redis with an optional expiration time.
//
// Parameters:
//   - key: The key under which to store the value.
//   - val: The value to store as a byte slice.
//   - exp: The expiration duration after which the key will be automatically deleted.
//     Use zero duration for no expiration.
//
// Returns:
//   - error: An error if the operation failed, nil otherwise.
func (s *Storage) Set(key string, val []byte, exp time.Duration) error {
	if len(key) <= 0 || len(val) <= 0 {
		return nil
	}
	return s.db.Set(context.Background(), key, val, exp).Err()
}

// Delete removes a key and its value from Redis.
//
// Parameters:
//   - key: The key to delete.
//
// Returns:
//   - error: An error if the delete operation failed, nil otherwise.
func (s *Storage) Delete(key string) error {
	if len(key) <= 0 {
		return nil
	}
	return s.db.Del(context.Background(), key).Err()
}

// Reset removes all keys from the Redis database, effectively clearing all stored data.
// This is equivalent to the FLUSHDB Redis command.
//
// Returns:
//   - error: An error if the flush operation failed, nil otherwise.
func (s *Storage) Reset() error {
	return s.db.FlushDB(context.Background()).Err()
}

// Close terminates the connection to the Redis server.
// This should be called when the storage is no longer needed to free up resources.
//
// Returns:
//   - error: An error if closing the connection failed, nil otherwise.
func (s *Storage) Close() error {
	return s.db.Close()
}

// Conn returns the underlying Redis client.
// This allows direct access to the Redis client for operations
// not covered by the Storage interface.
//
// Returns:
//   - redis.UniversalClient: The underlying Redis client instance.
func (s *Storage) Conn() redis.UniversalClient {
	return s.db
}

// Keys returns all keys in the Redis database.
// This method uses the SCAN command with a wildcard pattern to retrieve all keys.
//
// Returns:
//   - [][]byte: A slice of all keys as byte slices.
//   - error: An error if the key retrieval operation failed, nil otherwise.
func (s *Storage) Keys() ([][]byte, error) {
	var keys [][]byte
	var cursor uint64
	var err error

	for {
		var batch []string

		if batch, cursor, err = s.db.Scan(context.Background(), cursor, "*", 10).Result(); err != nil {
			return nil, err
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
