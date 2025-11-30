// Package response provides small helpers for writing JSON API responses
// with a consistent envelope structure.
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// JSONResponse is the common response envelope for all API endpoints.
type JSONResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorBody  `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// ErrorBody holds details about an API error.
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RespondJSON writes a successful JSON response with the given status code and payload.
func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	resp := JSONResponse{
		Success:   true,
		Data:      payload,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	writeJSON(w, status, resp)
}

// RespondError writes an error JSON response with the given status code and message.
func RespondError(w http.ResponseWriter, status int, msg string) {
	resp := JSONResponse{
		Success: false,
		Error: &ErrorBody{
			Code:    status,
			Message: msg,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	writeJSON(w, status, resp)
}

// writeJSON encodes v as JSON and writes it to the response writer.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
