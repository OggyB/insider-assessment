package server

import "net/http"

// Middleware is a simple HTTP middleware function.
//
// It takes a handler and returns a new handler that wraps it
type Middleware func(http.Handler) http.Handler

// Chain applies the given middleware in order around the provided handler.
func Chain(h http.Handler, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
