package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/chinmay706/gitf/internal/downloader"
)

// Server wraps the gitf downloader behind an HTTP API.
type Server struct {
	dl         *downloader.Downloader
	port       int
	corsOrigin string
	logger     *slog.Logger
	httpServer *http.Server
}

type Option func(*Server)

func WithPort(port int) Option {
	return func(s *Server) { s.port = port }
}

func WithCORSOrigin(origin string) Option {
	return func(s *Server) { s.corsOrigin = origin }
}

func WithLogger(l *slog.Logger) Option {
	return func(s *Server) { s.logger = l }
}

// New creates a configured server. Call ListenAndServe to start.
func New(dl *downloader.Downloader, opts ...Option) *Server {
	s := &Server{
		dl:         dl,
		port:       8080,
		corsOrigin: "*",
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/preview", s.handlePreview)
	mux.HandleFunc("POST /api/v1/download", s.handleDownload)

	handler := s.chain(mux,
		s.recoveryMiddleware,
		s.corsMiddleware,
		s.requestIDMiddleware,
		s.logMiddleware,
	)

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:        func(_ net.Listener) context.Context { return context.Background() },
	}

	return s
}

// ListenAndServe starts the server and blocks until ctx is cancelled,
// then gracefully shuts down with a 5-second drain period.
func (s *Server) ListenAndServe(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting server", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(shutdownCtx)
	}
}

// chain wraps a handler with the given middleware in reverse order so that the
// first middleware in the list is the outermost wrapper.
func (s *Server) chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
