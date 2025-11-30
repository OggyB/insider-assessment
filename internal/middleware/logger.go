package middleware

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger logs basic information about each HTTP request,
// including method, path, remote address and how long it took to serve.
func RequestLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			log.Printf("%s %s %s [%s]", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		})
	}
}
