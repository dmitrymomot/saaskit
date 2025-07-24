package session_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/pkg/cookie"
	"github.com/dmitrymomot/saaskit/pkg/session"
)

func BenchmarkMemoryStore_Create(b *testing.B) {
	store := session.NewMemoryStore(0)
	defer store.Close()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sess := session.NewSession("token"+string(rune(i)), nil, "", 1*time.Hour)
		_ = store.Create(ctx, sess)
	}
}

func BenchmarkMemoryStore_Get(b *testing.B) {
	store := session.NewMemoryStore(0)
	defer store.Close()
	ctx := context.Background()

	// Pre-populate store
	for i := 0; i < 1000; i++ {
		sess := session.NewSession("token"+string(rune(i)), nil, "", 1*time.Hour)
		_ = store.Create(ctx, sess)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get(ctx, "token"+string(rune(i%1000)))
	}
}

func BenchmarkMemoryStore_Update(b *testing.B) {
	store := session.NewMemoryStore(0)
	defer store.Close()
	ctx := context.Background()

	sess := session.NewSession("bench-token", nil, "", 1*time.Hour)
	_ = store.Create(ctx, sess)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sess.Set("counter", i)
		_ = store.Update(ctx, sess)
	}
}

func BenchmarkManager_Ensure(b *testing.B) {
	cookieMgr, _ := cookie.New([]string{"benchmark-secret-key-that-is-long-enough"})
	manager := session.New(
		session.WithCookieManager(cookieMgr),
		session.WithCleanupInterval(0),
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		_, _ = manager.Ensure(ctx, w, r)
	}
}

func BenchmarkManager_Get(b *testing.B) {
	cookieMgr, _ := cookie.New([]string{"benchmark-secret-key-that-is-long-enough"})
	manager := session.New(
		session.WithCookieManager(cookieMgr),
		session.WithCleanupInterval(0),
	)
	ctx := context.Background()

	// Create session
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	_, _ = manager.Ensure(ctx, w, r)

	// Prepare request with cookie
	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.Get(ctx, req)
	}
}

func BenchmarkManager_SetValue(b *testing.B) {
	cookieMgr, _ := cookie.New([]string{"benchmark-secret-key-that-is-long-enough"})
	manager := session.New(
		session.WithCookieManager(cookieMgr),
		session.WithCleanupInterval(0),
	)
	ctx := context.Background()

	// Create session
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Set(ctx, w, r, "key", i)
	}
}

func BenchmarkTransport_Cookie(b *testing.B) {
	cookieMgr, _ := cookie.New([]string{"benchmark-secret-key-that-is-long-enough"})
	trans := session.NewCookieTransport(cookieMgr, "bench-sid")

	token := "benchmark-token"
	ttl := 1 * time.Hour

	b.Run("SetToken", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			_ = trans.SetToken(w, token, ttl)
		}
	})

	b.Run("GetToken", func(b *testing.B) {
		// Prepare request with cookie
		w := httptest.NewRecorder()
		_ = trans.SetToken(w, token, ttl)

		r := httptest.NewRequest("GET", "/", nil)
		for _, c := range w.Result().Cookies() {
			r.AddCookie(c)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = trans.GetToken(r)
		}
	})
}

func BenchmarkTransport_Header(b *testing.B) {
	trans := session.NewHeaderTransport("X-Session-Token")
	token := "benchmark-token"
	ttl := 1 * time.Hour

	b.Run("SetToken", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			w := httptest.NewRecorder()
			_ = trans.SetToken(w, token, ttl)
		}
	})

	b.Run("GetToken", func(b *testing.B) {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Session-Token", "Bearer "+token)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = trans.GetToken(r)
		}
	})
}

func BenchmarkSession_Operations(b *testing.B) {
	userID := uuid.New()
	sess := session.NewSession("token", &userID, "fingerprint", 1*time.Hour)

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sess.Set("key", i)
		}
	})

	b.Run("Get", func(b *testing.B) {
		sess.Set("key", "value")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = sess.Get("key")
		}
	})

	b.Run("IsAuthenticated", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = sess.IsAuthenticated()
		}
	})

	b.Run("ValidateFingerprint", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = sess.ValidateFingerprint("fingerprint")
		}
	})
}

func BenchmarkMiddleware(b *testing.B) {
	cookieMgr, _ := cookie.New([]string{"benchmark-secret-key-that-is-long-enough"})
	manager := session.New(
		session.WithCookieManager(cookieMgr),
		session.WithCleanupInterval(0),
	)

	// Create session
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	_, _ = manager.Ensure(context.Background(), w, r)

	// Prepare request with session
	req := httptest.NewRequest("GET", "/", nil)
	for _, c := range w.Result().Cookies() {
		req.AddCookie(c)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate work
		if sess, ok := session.FromContext(r.Context()); ok {
			w.Header().Set("X-Session-ID", sess.ID.String())
		}
	})

	middleware := manager.Middleware(handler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)
	}
}
