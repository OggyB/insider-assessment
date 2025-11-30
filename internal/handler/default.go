package handler

import (
	"net/http"

	"github.com/oggyb/insider-assessment/internal/response"
)

// HomeHandler serves basic root, health and ping endpoints.
type HomeHandler struct{}

// NewHomeHandler returns a new HomeHandler.
func NewHomeHandler() *HomeHandler { return &HomeHandler{} }

// Index godoc
// @Summary     Welcome endpoint
// @Description Simple root endpoint that returns a welcome message.
// @Tags        home
// @Produce     json
// @Success     200 {object} response.WelcomeResponse
// @Router      / [get]
func (h *HomeHandler) Index(w http.ResponseWriter, r *http.Request) {
	payload := response.WelcomePayload{
		Message: "Welcome to Insider Messaging Assessment",
	}

	response.RespondJSON(w, http.StatusOK, payload)
}

// Health godoc
// @Summary     Health check
// @Description Returns a basic status payload to indicate the API is running.
// @Tags        home
// @Produce     json
// @Success     200 {object} response.HealthResponse
// @Router      /health [get]
func (h *HomeHandler) Health(w http.ResponseWriter, r *http.Request) {
	payload := response.HealthPayload{
		Status: "ok",
	}

	response.RespondJSON(w, http.StatusOK, payload)
}
