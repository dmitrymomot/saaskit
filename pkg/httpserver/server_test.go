package httpserver_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	httpserver "github.com/dmitrymomot/saaskit/pkg/httpserver"
)

func freeAddr(t *testing.T) string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("unable to get free port: %v", err)
	}
	addr := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
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
	for i := 0; i < 20; i++ {
		resp, err = http.Get("http://" + addr)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("http get: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("close body: %v", err)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
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
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
	}
}

func TestStartError(t *testing.T) {
	t.Parallel()
	srv := httpserver.New(httpserver.WithAddr(":invalid"))
	err := srv.Run(context.Background(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	if err == nil || !errors.Is(err, httpserver.ErrStart) {
		t.Fatalf("expected httpserver.ErrStart, got %v", err)
	}
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
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	if !started.Load() || !stopped.Load() {
		t.Fatalf("hooks not executed")
	}
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

	if err := srv.Run(context.Background(), http.NewServeMux()); err == nil || !errors.Is(err, httpserver.ErrStart) {
		t.Fatalf("expected httpserver.ErrStart, got %v", err)
	}
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
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
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
	if hs.ReadTimeout != time.Second || hs.Addr != addr || hs.Handler == nil {
		t.Fatalf("server not configured")
	}
	_ = srv.Shutdown(context.Background())
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
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
	for i := 0; i < 20; i++ {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGTERM)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
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
				if recover() == nil {
					t.Fatal("expected panic")
				}
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
	if hs.Addr != addr {
		t.Fatalf("addr option not applied")
	}
	if hs.ReadTimeout != time.Second || hs.WriteTimeout != 2*time.Second || hs.IdleTimeout != 3*time.Second {
		t.Fatalf("timeout options not applied")
	}
	if logger != l {
		t.Fatalf("logger option not applied")
	}
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
	if hs.ReadTimeout != time.Second || hs.WriteTimeout != 2*time.Second || hs.IdleTimeout != 3*time.Second {
		t.Fatalf("timeouts not applied to server")
	}
	_ = srv.Shutdown(context.Background())
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("run did not finish")
	}
}
