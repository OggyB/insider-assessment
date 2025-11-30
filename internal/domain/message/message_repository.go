package message

import "context"

// Repository defines the persistence operations for Message aggregates.
//
// It is implemented by infrastructure layers (e.g. GORM, sqlc, etc.)
// while the domain and service layers depend only on this interface.
type Repository interface {
	// Save persists a new message.
	Save(ctx context.Context, m *Message) error

	// GetPending returns up to limit messages that are still waiting to be sent.
	GetPending(ctx context.Context, limit int) ([]*Message, error)

	// GetSent returns a paginated list of successfully sent messages
	// along with the total number of sent records.
	GetSent(ctx context.Context, page, limit int) ([]*Message, int64, error)

	// UpdateStatus updates the status and metadata of an existing message.
	UpdateStatus(ctx context.Context, m *Message) error
}
