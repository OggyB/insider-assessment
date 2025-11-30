package response

import (
	"time"

	domain "github.com/oggyb/insider-assessment/internal/domain/message"
)

type WelcomePayload struct {
	Message string `json:"message"`
}

type HealthPayload struct {
	Status string `json:"status"`
}

type PingPayload struct {
	Pong bool `json:"pong"`
}

type WelcomeResponse struct {
	Success   bool           `json:"success"`
	Data      WelcomePayload `json:"data"`
	Timestamp string         `json:"timestamp"`
}

type HealthResponse struct {
	Success   bool          `json:"success"`
	Data      HealthPayload `json:"data"`
	Timestamp string        `json:"timestamp"`
}

type PingResponse struct {
	Success   bool        `json:"success"`
	Data      PingPayload `json:"data"`
	Timestamp string      `json:"timestamp"`
}

type SchedulerControlPayload struct {
	Message string `json:"message"`
}

type SchedulerControlResponse struct {
	Success   bool                    `json:"success"`
	Data      SchedulerControlPayload `json:"data"`
	Timestamp string                  `json:"timestamp"`
}

// MessageDTO is a public-facing representation of a message
// used in API responses. It decouples the wire format from
// the domain entity and plays nicely with Swagger.
type MessageDTO struct {
	ID        string     `json:"id"`
	To        string     `json:"to"`
	Content   string     `json:"content"`
	Status    string     `json:"status"`
	MessageID string     `json:"messageId"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type SentMessagesPayload struct {
	Items []MessageDTO `json:"items"`
	Total int64        `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}

type SentMessagesResponse struct {
	Success   bool                `json:"success"`
	Data      SentMessagesPayload `json:"data"`
	Timestamp string              `json:"timestamp"`
}

// FromDomainMessages converts domain messages into DTOs
// for use in HTTP responses.
func FromDomainMessages(msgs []*domain.Message) []MessageDTO {
	out := make([]MessageDTO, len(msgs))
	for i, m := range msgs {
		out[i] = MessageDTO{
			ID:        m.ID.String(),
			To:        m.To,
			Content:   m.Content,
			Status:    string(m.Status),
			MessageID: m.MessageID,
			SentAt:    m.SentAt,
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
		}
	}
	return out
}

type WebhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}
