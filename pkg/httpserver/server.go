package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type config struct {
	addr            string
	readTimeout     time.Duration
	writeTimeout    time.Duration
	idleTimeout     time.Duration
	shutdownTimeout time.Duration
	server          *http.Server
	logger          *slog.Logger
	startHooks      []func(*slog.Logger)
	stopHooks       []func(*slog.Logger)
}

func defaultConfig() *config {
	return &config{
		addr:            ":8080",
		shutdownTimeout: 5 * time.Second,
	}
}

// Server wraps http.Server with graceful shutdown and logging.
type Server struct {
	cfg    *config
	srv    *http.Server
	once   sync.Once
	mu     sync.Mutex
	closed bool
}

// New returns a configured Server.
func New(opts ...Option) *Server {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.logger == nil {
		cfg.logger = newNoopLogger()
	}
	return &Server{cfg: cfg}
}

// Run starts the HTTP server and blocks until shutdown.
// It returns ErrStart wrapped with the underlying error if the server fails to start.
func (s *Server) Run(ctx context.Context, handler http.Handler) error {
	if handler == nil {
		handler = http.NotFoundHandler()
	}

	s.mu.Lock()
	if s.srv != nil {
		s.mu.Unlock()
		return errors.Join(ErrStart, errors.New("server already running"))
	}

	cfg := s.cfg
	srv := cfg.server
	if srv == nil {
		srv = &http.Server{}
	}

	if srv.Addr == "" {
		srv.Addr = cfg.addr
	}
	if srv.ReadTimeout == 0 && cfg.readTimeout != 0 {
		srv.ReadTimeout = cfg.readTimeout
	}
	if srv.WriteTimeout == 0 && cfg.writeTimeout != 0 {
		srv.WriteTimeout = cfg.writeTimeout
	}
	if srv.IdleTimeout == 0 && cfg.idleTimeout != 0 {
		srv.IdleTimeout = cfg.idleTimeout
	}
	srv.Handler = handler
	s.srv = srv
	s.mu.Unlock()

	for _, h := range cfg.startHooks {
		h(cfg.logger)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var runErr error
	select {
	case <-ctx.Done():
		_ = s.Shutdown(context.Background())
		runErr = <-errCh
	case <-stop:
		_ = s.Shutdown(context.Background())
		runErr = <-errCh
	case runErr = <-errCh:
	}
	signal.Stop(stop)

	if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
		return errors.Join(ErrStart, runErr)
	}
	return nil
}

// Shutdown stops the server gracefully before Run returns.
// It is safe for repeated calls.
// Any error from http.Server.Shutdown is wrapped with ErrShutdown.
func (s *Server) Shutdown(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		if s.srv == nil {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, s.cfg.shutdownTimeout)
		defer cancel()
		err = s.srv.Shutdown(ctx)
		for _, h := range s.cfg.stopHooks {
			h(s.cfg.logger)
		}
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
	})

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Join(ErrShutdown, err)
	}
	return nil
}
