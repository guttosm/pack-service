package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// Server wraps http.Server with graceful shutdown capabilities.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

// NewServer creates a new Server instance with optimized settings.
func NewServer(handler http.Handler, port string) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:           ":" + port,
			Handler:        handler,
			ReadTimeout:    15 * time.Second,
			WriteTimeout:   15 * time.Second,
			IdleTimeout:    60 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1MB
		},
		shutdownTimeout: 10 * time.Second,
	}
}

// Run starts the server and blocks until shutdown signal is received.
func (s *Server) Run() error {
	errChan := make(chan error, 1)

	go func() {
		log.Info().Str("addr", s.httpServer.Addr).Msg("Server starting")
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("Received signal, initiating graceful shutdown")
	}

	return s.Shutdown()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
		return err
	}

	log.Info().Msg("Server stopped gracefully")
	return nil
}
