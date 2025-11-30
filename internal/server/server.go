package server

import (
	"context"
	"net/http"
	"time"

	"github.com/oggyb/insider-assessment/internal/middleware"
	routes "github.com/oggyb/insider-assessment/internal/router"
)

// Server owns the underlying http.Server instance.
type Server struct {
	http *http.Server
}

// New creates a new HTTP server bound to the given address and configured
// with the provided application dependencies and middleware chain.
func New(addr string, deps routes.AppDeps) *Server {
	mux := http.NewServeMux()
	routes.Register(mux, deps)

	root := Chain(
		mux,
		middleware.RequestLogger(),
	)

	return &Server{
		http: &http.Server{
			Addr:              addr,
			Handler:           root,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}
}

// Start runs the HTTP server and blocks until ListenAndServe returns.
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server, waiting for in-flight
// requests to complete until the given context expires.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}
