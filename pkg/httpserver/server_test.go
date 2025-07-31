package httpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	httpserver "github.com/dmitrymomot/saaskit/pkg/httpserver"
)

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "unable to get free port")
	addr := l.Addr().String()
	require.NoError(t, l.Close(), "close listener")
	return addr
}

func TestRunAndShutdown(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	srv := httpserver.New(httpserver.WithAddr(addr), httpserver.WithShutdownTimeout(100*time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(ctx, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	}()

	var resp *http.Response
	var err error
	// Wait for server to start listening with more generous timeouts
	for range 50 {
		resp, err = http.Get("http://" + addr)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.NoError(t, err, "http get after 50 retries")
	require.NoError(t, resp.Body.Close(), "close body")

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err, "run")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
	require.NoError(t, srv.Shutdown(context.Background()), "shutdown")
}

func TestManualShutdown(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	start := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithAddr(addr),
		httpserver.WithShutdownTimeout(100*time.Millisecond),
		httpserver.WithStartHook(func(_ *slog.Logger) { close(start) }),
	)

	done := make(chan error, 1)
	go func() {
		done <- srv.Run(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	}()
	<-start
	require.NoError(t, srv.Shutdown(context.Background()), "shutdown")
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
}

func TestStartError(t *testing.T) {
	t.Parallel()
	srv := httpserver.New(httpserver.WithAddr(":invalid"))
	err := srv.Run(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrStart)
}

func TestHooks(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	var started, stopped atomic.Bool
	start := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithAddr(addr),
		httpserver.WithStartHook(func(_ *slog.Logger) {
			started.Store(true)
			close(start)
		}),
		httpserver.WithStopHook(func(_ *slog.Logger) { stopped.Store(true) }),
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(ctx, http.NewServeMux()) }()
	<-start
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
	require.NoError(t, srv.Shutdown(context.Background()), "shutdown")

	assert.True(t, started.Load(), "start hook not executed")
	assert.True(t, stopped.Load(), "stop hook not executed")
}

func TestAlreadyRunning(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	started := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithAddr(addr),
		httpserver.WithShutdownTimeout(50*time.Millisecond),
		httpserver.WithStartHook(func(_ *slog.Logger) { close(started) }),
	)
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = srv.Run(ctx, http.NewServeMux()) }()
	<-started

	err := srv.Run(context.Background(), http.NewServeMux())
	require.Error(t, err)
	assert.ErrorIs(t, err, httpserver.ErrStart)
	cancel()
	_ = srv.Shutdown(context.Background())
}

func TestDoubleShutdown(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	start := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithAddr(addr),
		httpserver.WithShutdownTimeout(50*time.Millisecond),
		httpserver.WithStartHook(func(_ *slog.Logger) { close(start) }),
	)
	done := make(chan error, 1)
	go func() { done <- srv.Run(context.Background(), http.NewServeMux()) }()
	<-start
	require.NoError(t, srv.Shutdown(context.Background()), "first shutdown")
	require.NoError(t, srv.Shutdown(context.Background()), "second shutdown")
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
}

func TestWithServer(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	hs := &http.Server{ReadTimeout: time.Second}
	start := make(chan struct{})
	srv := httpserver.New(
		httpserver.WithServer(hs),
		httpserver.WithAddr(addr),
		httpserver.WithStartHook(func(_ *slog.Logger) { close(start) }),
	)
	done := make(chan error, 1)
	go func() { done <- srv.Run(context.Background(), http.NewServeMux()) }()
	<-start
	assert.Equal(t, time.Second, hs.ReadTimeout, "read timeout not set")
	assert.Equal(t, addr, hs.Addr, "address not set")
	assert.NotNil(t, hs.Handler, "handler not set")
	_ = srv.Shutdown(context.Background())
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
}

func TestSignalShutdown(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	srv := httpserver.New(
		httpserver.WithAddr(addr),
		httpserver.WithShutdownTimeout(50*time.Millisecond),
	)
	done := make(chan error, 1)
	go func() { done <- srv.Run(context.Background(), http.NewServeMux()) }()
	// Wait for server to start listening
	for range 50 {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGTERM)
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
}

func TestOptionPanics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		fn   func()
	}{
		{"addr", func() { httpserver.WithAddr("") }},
		{"read", func() { httpserver.WithReadTimeout(-time.Second) }},
		{"write", func() { httpserver.WithWriteTimeout(-time.Second) }},
		{"idle", func() { httpserver.WithIdleTimeout(-time.Second) }},
		{"shutdown", func() { httpserver.WithShutdownTimeout(-time.Second) }},
		{"server", func() { httpserver.WithServer(nil) }},
		{"start hook", func() { httpserver.WithStartHook(nil) }},
		{"stop hook", func() { httpserver.WithStopHook(nil) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				assert.NotNil(t, recover(), "expected panic")
			}()
			tt.fn()
		})
	}

	t.Run("logger nil allowed", func(t *testing.T) {
		t.Parallel()
		defer func() { _ = recover() }()
		httpserver.WithLogger(nil)
	})
}

func TestOptionsApply(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	l := slog.New(slog.NewTextHandler(io.Discard, nil))
	hs := &http.Server{}
	gotLogger := make(chan *slog.Logger, 1)
	srv := httpserver.New(
		httpserver.WithServer(hs),
		httpserver.WithAddr(addr),
		httpserver.WithReadTimeout(time.Second),
		httpserver.WithWriteTimeout(2*time.Second),
		httpserver.WithIdleTimeout(3*time.Second),
		httpserver.WithShutdownTimeout(50*time.Millisecond),
		httpserver.WithLogger(l),
		httpserver.WithStartHook(func(lg *slog.Logger) { gotLogger <- lg }),
	)
	done := make(chan error, 1)
	go func() { done <- srv.Run(context.Background(), nil) }()
	logger := <-gotLogger
	assert.Equal(t, addr, hs.Addr, "addr option not applied")
	assert.Equal(t, time.Second, hs.ReadTimeout, "read timeout not applied")
	assert.Equal(t, 2*time.Second, hs.WriteTimeout, "write timeout not applied")
	assert.Equal(t, 3*time.Second, hs.IdleTimeout, "idle timeout not applied")
	assert.Equal(t, l, logger, "logger option not applied")
	_ = srv.Shutdown(context.Background())
	<-done
}

func TestTimeouts(t *testing.T) {
	t.Parallel()
	addr := freeAddr(t)
	start := make(chan struct{})
	hs := &http.Server{}
	srv := httpserver.New(
		httpserver.WithServer(hs),
		httpserver.WithAddr(addr),
		httpserver.WithReadTimeout(time.Second),
		httpserver.WithWriteTimeout(2*time.Second),
		httpserver.WithIdleTimeout(3*time.Second),
		httpserver.WithShutdownTimeout(50*time.Millisecond),
		httpserver.WithStartHook(func(_ *slog.Logger) { close(start) }),
	)
	done := make(chan error, 1)
	go func() { done <- srv.Run(context.Background(), nil) }()
	<-start
	assert.Equal(t, time.Second, hs.ReadTimeout, "read timeout not applied")
	assert.Equal(t, 2*time.Second, hs.WriteTimeout, "write timeout not applied")
	assert.Equal(t, 3*time.Second, hs.IdleTimeout, "idle timeout not applied")
	_ = srv.Shutdown(context.Background())
	select {
	case err := <-done:
		require.NoError(t, err, "run error")
	case <-time.After(time.Second):
		require.Fail(t, "run did not finish")
	}
}
