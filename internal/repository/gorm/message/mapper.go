package messagegorm

import (
	"github.com/oggyb/insider-assessment/internal/domain/message"
)

// toDomain maps a GORM MessageModel to a domain-level Message.
func toDomain(m *MessageModel) *message.Message {
	return &message.Message{
		ID:          m.ID,
		To:          m.To,
		Content:     m.Content,
		Status:      message.Status(m.Status),
		MessageID:   m.MessageID,
		RawResponse: m.RawResponse,
		SentAt:      m.SentAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// toDomainMany maps a slice of MessageModel to a slice of domain Messages.
func toDomainMany(models []MessageModel) []*message.Message {
	out := make([]*message.Message, len(models))
	for i := range models {
		out[i] = toDomain(&models[i])
	}
	return out
}

// fromDomain maps a domain-level Message to a GORM MessageModel.
func fromDomain(d *message.Message) *MessageModel {
	return &MessageModel{
		ID:          d.ID,
		To:          d.To,
		Content:     d.Content,
		Status:      string(d.Status),
		MessageID:   d.MessageID,
		RawResponse: d.RawResponse,
		SentAt:      d.SentAt,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
