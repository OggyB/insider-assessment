package handler

import (
	"encoding/json"
	"github.com/oggyb/insider-assessment/internal/request"
	"github.com/oggyb/insider-assessment/internal/response"
	"github.com/oggyb/insider-assessment/internal/scheduler"
	"github.com/oggyb/insider-assessment/internal/service"
	"net/http"
	"strconv"
)

// MessageHandler wires HTTP endpoints to the message service
// and the background scheduler.
type MessageHandler struct {
	msgSvc service.MessageService
	schSvc scheduler.SchedulerService
}

// NewMessageHandler constructs a new MessageHandler with its dependencies.
func NewMessageHandler(msgSvc service.MessageService, schSvc scheduler.SchedulerService) *MessageHandler {
	return &MessageHandler{
		msgSvc: msgSvc,
		schSvc: schSvc,
	}
}

// StartStopScheduler godoc
// @Summary     Control scheduler
// @Description Starts or stops the background scheduler based on the given action.
// @Tags        scheduler
// @Accept      json
// @Produce     json
// @Param       request body request.SchedulerRequest true "Scheduler action (start|stop)"
// @Success     200 {object} response.SchedulerControlResponse
// @Failure     400 {object} map[string]string
// @Router      /scheduler [post]
func (h *MessageHandler) StartStopScheduler(w http.ResponseWriter, r *http.Request) {
	var req request.SchedulerRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.RespondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch req.Action {
	case "start":
		if err := h.schSvc.Start(); err != nil {
			response.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		payload := response.SchedulerControlPayload{
			Message: "scheduler started",
		}
		response.RespondJSON(w, http.StatusOK, payload)
		return

	case "stop":
		if err := h.schSvc.Stop(); err != nil {
			response.RespondError(w, http.StatusBadRequest, err.Error())
			return
		}

		payload := response.SchedulerControlPayload{
			Message: "scheduler stopped",
		}
		response.RespondJSON(w, http.StatusOK, payload)
		return

	default:
		response.RespondError(w, http.StatusBadRequest, "action must be 'start' or 'stop'")
		return
	}
}

// GetSentMessages godoc
// @Summary     List sent messages
// @Description Returns a paginated list of successfully sent messages.
// @Tags        messages
// @Produce     json
// @Param       page  query int false "Page number"         default(1)
// @Param       limit query int false "Page size (max 100)" default(20)
// @Success     200 {object} response.SentMessagesResponse
// @Failure     500 {object} map[string]string
// @Router      /messages/sent [get]
func (h *MessageHandler) GetSentMessages(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	page := 1
	limit := 20

	if v, err := strconv.Atoi(pageStr); err == nil && v > 0 {
		page = v
	}

	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
		limit = v
	}

	items, total, err := h.msgSvc.GetSent(r.Context(), page, limit)
	if err != nil {
		response.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	payload := response.SentMessagesPayload{
		Items: response.FromDomainMessages(items),
		Total: total,
		Page:  page,
		Limit: limit,
	}

	response.RespondJSON(w, http.StatusOK, payload)
}
