// Package message holds the domain model and invariants for messages.
package message

import (
	"errors"
	"github.com/google/uuid"
	"strings"
	"time"
)

const (
	// MaxContentLength is the maximum allowed length for message content.
	MaxContentLength = 255
)

type Status string

const (
	StatusPending Status = "PENDING"
	StatusSuccess Status = "SUCCESS"
	StatusFailed  Status = "FAILED"
)

var (
	// ErrEmptyRecipient is returned when no recipient phone number is provided.
	ErrEmptyRecipient = errors.New("recipient phone number is required")
	// ErrEmptyContent is returned when the message body is empty.
	ErrEmptyContent = errors.New("message content is required")
	// ErrContentTooLong is returned when the message body exceeds MaxContentLength.
	ErrContentTooLong = errors.New("message content exceeds maximum length")
)

// Message is the core domain entity representing an outgoing SMS message.
type Message struct {
	ID          uuid.UUID
	To          string
	Content     string
	Status      Status
	MessageID   string
	RawResponse string
	SentAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewMessage constructs a new pending Message and enforces basic domain rules.
func NewMessage(to, content string) (*Message, error) {
	to = strings.TrimSpace(to)
	content = strings.TrimSpace(content)

	if to == "" {
		return nil, ErrEmptyRecipient
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len(content) > MaxContentLength {
		return nil, ErrContentTooLong
	}

	return &Message{
		ID:        uuid.New(),
		To:        to,
		Content:   content,
		Status:    StatusPending,
		CreatedAt: time.Now(),
	}, nil
}

// MarkSent marks the message as successfully sent and records provider metadata.
func (m *Message) MarkSent(msgID string, raw string) {
	now := time.Now()
	m.SentAt = &now
	m.Status = StatusSuccess
	m.MessageID = msgID
	m.RawResponse = raw
}

// MarkFailed marks the message as failed and stores the raw provider response.
func (m *Message) MarkFailed(raw string) {
	m.Status = StatusFailed
	m.RawResponse = raw
}
