//go:build !integration

package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(handler, "8080")

	assert.NotNil(t, server)
	assert.NotNil(t, server.httpServer)
	assert.Equal(t, ":8080", server.httpServer.Addr)
	assert.Equal(t, 15*time.Second, server.httpServer.ReadTimeout)
	assert.Equal(t, 15*time.Second, server.httpServer.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.httpServer.IdleTimeout)
	assert.Equal(t, 10*time.Second, server.shutdownTimeout)
}

func TestServer_Shutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(handler, "8080")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = server.httpServer.Shutdown(ctx)
	}()

	err := server.Shutdown()
	assert.NoError(t, err)
}

func TestServer_Run(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(handler, "0")

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Run()
	}()

	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "http://localhost:0/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(syscall.SIGTERM)

	select {
	case err := <-errChan:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Server did not shutdown in time")
	}
}

func TestServer_Run_WithError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(handler, "invalid-port")

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Run()
	}()

	select {
	case err := <-errChan:
		assert.Error(t, err)
	case <-time.After(1 * time.Second):
		proc, _ := os.FindProcess(os.Getpid())
		_ = proc.Signal(syscall.SIGTERM)
		time.Sleep(100 * time.Millisecond)
	}
}

func TestServer_Run_GracefulShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := NewServer(handler, "0")

	done := make(chan bool, 1)
	go func() {
		err := server.Run()
		assert.NoError(t, err)
		done <- true
	}()

	time.Sleep(50 * time.Millisecond)

	proc, _ := os.FindProcess(os.Getpid())
	_ = proc.Signal(syscall.SIGTERM)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		require.Fail(t, "Server did not shutdown gracefully")
	}
}

