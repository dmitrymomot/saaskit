package ratelimiter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/ratelimiter"
)

func TestBucket_ConcurrentSafety(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       1000,
		RefillRate:     100,
		RefillInterval: 10 * time.Second, // Long interval to prevent refills during test
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("concurrent requests same key", func(t *testing.T) {
		key := "concurrent-test"
		goroutines := 100
		requestsPerGoroutine := 20

		var wg sync.WaitGroup
		wg.Add(goroutines)

		var allowed atomic.Int64
		var denied atomic.Int64

		for range goroutines {
			go func() {
				defer wg.Done()
				for range requestsPerGoroutine {
					result, err := tb.Allow(ctx, key)
					if err == nil {
						if result.Allowed() {
							allowed.Add(1)
						} else {
							denied.Add(1)
						}
					}
				}
			}()
		}

		wg.Wait()

		totalRequests := int64(goroutines * requestsPerGoroutine)
		assert.Equal(t, totalRequests, allowed.Load()+denied.Load())
		assert.LessOrEqual(t, allowed.Load(), int64(config.Capacity))
	})

	t.Run("concurrent requests different keys", func(t *testing.T) {
		goroutines := 50
		requestsPerGoroutine := 10

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(id int) {
				defer wg.Done()
				key := "key-" + string(rune('a'+id))

				for range requestsPerGoroutine {
					_, err := tb.AllowN(ctx, key, 1+id%3)
					assert.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()
	})

	t.Run("concurrent allow and reset", func(t *testing.T) {
		key := "reset-test"
		goroutines := 20

		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for range goroutines {
			go func() {
				defer wg.Done()
				for range 50 {
					_, _ = tb.Allow(ctx, key)
					time.Sleep(time.Microsecond)
				}
			}()

			go func() {
				defer wg.Done()
				for range 10 {
					_ = tb.Reset(ctx, key)
					time.Sleep(5 * time.Microsecond)
				}
			}()
		}

		wg.Wait()
	})

	t.Run("concurrent status checks", func(t *testing.T) {
		key := "status-test"
		goroutines := 50

		var wg sync.WaitGroup
		wg.Add(goroutines * 2)

		for range goroutines {
			go func() {
				defer wg.Done()
				for range 20 {
					_, _ = tb.Allow(ctx, key)
				}
			}()

			go func() {
				defer wg.Done()
				for range 20 {
					result, err := tb.Status(ctx, key)
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			}()
		}

		wg.Wait()
	})
}

func TestMemoryStore_ConcurrentCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cleanup test in short mode")
	}

	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       10,
		RefillRate:     1,
		RefillInterval: 10 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore(
		ratelimiter.WithCleanupInterval(50 * time.Millisecond),
	)
	defer store.Close()

	t.Run("concurrent operations during cleanup", func(t *testing.T) {
		goroutines := 20
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(id int) {
				defer wg.Done()
				key := "cleanup-key-" + string(rune('a'+id))

				for j := range 100 {
					if j%10 == 0 {
						_ = store.Reset(ctx, key)
					} else {
						_, _, _ = store.ConsumeTokens(ctx, key, 1, config)
					}

					if j%20 == 0 {
						time.Sleep(60 * time.Millisecond)
					}
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestMiddleware_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping middleware concurrency test in short mode")
	}

	t.Parallel()

	config := ratelimiter.Config{
		Capacity:       100,
		RefillRate:     10,
		RefillInterval: 10 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	limiter, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	keyFunc := func(r *http.Request) string {
		return r.Header.Get("X-Client-ID")
	}

	middleware := ratelimiter.Middleware(limiter, keyFunc)

	var requestCount atomic.Int64
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		time.Sleep(time.Microsecond)
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("high concurrency single client", func(t *testing.T) {
		goroutines := 50
		requestsPerGoroutine := 10

		var wg sync.WaitGroup
		wg.Add(goroutines)

		var successCount atomic.Int64
		var rateLimitedCount atomic.Int64

		for range goroutines {
			go func() {
				defer wg.Done()
				for range requestsPerGoroutine {
					req := httptest.NewRequest("GET", "/test", nil)
					req.Header.Set("X-Client-ID", "client1")
					rec := httptest.NewRecorder()

					handler.ServeHTTP(rec, req)

					if rec.Code == http.StatusOK {
						successCount.Add(1)
					} else if rec.Code == http.StatusTooManyRequests {
						rateLimitedCount.Add(1)
					}
				}
			}()
		}

		wg.Wait()

		totalRequests := int64(goroutines * requestsPerGoroutine)
		assert.Equal(t, totalRequests, successCount.Load()+rateLimitedCount.Load())
		assert.LessOrEqual(t, successCount.Load(), int64(config.Capacity))
	})

	t.Run("high concurrency multiple clients", func(t *testing.T) {
		clients := 10
		goroutinesPerClient := 5
		requestsPerGoroutine := 20

		var wg sync.WaitGroup
		wg.Add(clients * goroutinesPerClient)

		for c := range clients {
			clientID := "client-" + string(rune('a'+c))

			for range goroutinesPerClient {
				go func(id string) {
					defer wg.Done()
					for range requestsPerGoroutine {
						req := httptest.NewRequest("GET", "/test", nil)
						req.Header.Set("X-Client-ID", id)
						rec := httptest.NewRecorder()

						handler.ServeHTTP(rec, req)
					}
				}(clientID)
			}
		}

		wg.Wait()
	})
}

func TestBucket_RaceConditions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race detector test in short mode")
	}

	t.Parallel()

	ctx := context.Background()
	config := ratelimiter.Config{
		Capacity:       50,
		RefillRate:     5,
		RefillInterval: 5 * time.Millisecond,
	}

	store := ratelimiter.NewMemoryStore()
	defer store.Close()

	tb, err := ratelimiter.NewBucket(store, config)
	require.NoError(t, err)

	t.Run("mixed operations race test", func(t *testing.T) {
		key := "race-key"
		stop := make(chan struct{})
		var wg sync.WaitGroup

		operations := []func(){
			func() {
				_, _ = tb.Allow(ctx, key)
			},
			func() {
				_, _ = tb.AllowN(ctx, key, 3)
			},
			func() {
				_, _ = tb.Status(ctx, key)
			},
			func() {
				_ = tb.Reset(ctx, key)
			},
		}

		for _, op := range operations {
			wg.Add(1)
			go func(operation func()) {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						operation()
					}
				}
			}(op)
		}

		time.Sleep(100 * time.Millisecond)
		close(stop)
		wg.Wait()
	})

	t.Run("burst traffic simulation", func(t *testing.T) {
		keys := []string{"burst1", "burst2", "burst3"}
		goroutines := 30

		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := range goroutines {
			go func(id int) {
				defer wg.Done()
				key := keys[id%len(keys)]

				for j := range 50 {
					if j%10 == 0 {
						time.Sleep(time.Millisecond)
					}

					n := 1 + (j % 5)
					result, err := tb.AllowN(ctx, key, n)
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			}(i)
		}

		wg.Wait()
	})
}

func TestComposite_ConcurrentKeyGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent key generation test in short mode")
	}

	t.Parallel()

	keyFunc1 := func(r *http.Request) string {
		return r.Header.Get("X-User")
	}
	keyFunc2 := func(r *http.Request) string {
		return r.URL.Path
	}
	keyFunc3 := func(r *http.Request) string {
		return r.Method
	}

	composite := ratelimiter.Composite(keyFunc1, keyFunc2, keyFunc3)

	t.Run("concurrent key generation", func(t *testing.T) {
		goroutines := 100
		var wg sync.WaitGroup
		wg.Add(goroutines)

		keys := make([]string, goroutines)

		for i := range goroutines {
			go func(id int) {
				defer wg.Done()

				req := httptest.NewRequest("GET", "/api/test", nil)
				req.Header.Set("X-User", "user"+string(rune('0'+id%10)))

				keys[id] = composite(req)
			}(i)
		}

		wg.Wait()

		for i, key := range keys {
			expectedUser := "user" + string(rune('0'+i%10))
			assert.Contains(t, key, expectedUser)
		}
	})
}
