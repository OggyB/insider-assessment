package messagegorm

import (
	"context"

	"github.com/oggyb/insider-assessment/internal/db"
	"github.com/oggyb/insider-assessment/internal/domain/message"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository is a GORM-backed implementation of the message.Repository interface.
type Repository struct {
	db *gorm.DB
}

// NewRepository constructs a message repository using the given DB adapter.
func NewRepository(d db.DB) *Repository {
	return &Repository{
		db: d.Conn().(*gorm.DB),
	}
}

// GetPending returns up to limit pending messages ordered by creation time,
// using SELECT ... FOR UPDATE SKIP LOCKED to avoid double-processing in concurrent workers.
func (r *Repository) GetPending(ctx context.Context, limit int) ([]*message.Message, error) {
	var models []MessageModel

	err := r.db.WithContext(ctx).
		Where("status = ?", message.StatusPending).
		Order("created_at ASC").
		Limit(limit).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	return toDomainMany(models), nil
}

// GetSent returns a paginated list of successfully sent messages and the total count.
func (r *Repository) GetSent(ctx context.Context, page, limit int) ([]*message.Message, int64, error) {
	var models []MessageModel
	var total int64

	query := r.db.WithContext(ctx).
		Model(&MessageModel{}).
		Where("status = ?", message.StatusSuccess)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit

	err := query.
		Order("sent_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&models).Error

	if err != nil {
		return nil, 0, err
	}

	return toDomainMany(models), total, nil
}

// UpdateStatus persists the current status and metadata of a message.
func (r *Repository) UpdateStatus(ctx context.Context, m *message.Message) error {
	updates := map[string]interface{}{
		"status":       string(m.Status),
		"message_id":   m.MessageID,
		"raw_response": m.RawResponse,
		"sent_at":      m.SentAt,
	}

	return r.db.WithContext(ctx).
		Model(&MessageModel{}).
		Where("id = ?", m.ID).
		Updates(updates).Error
}

// Save inserts a new message record into the database.
func (r *Repository) Save(ctx context.Context, msg *message.Message) error {
	dbModel := fromDomain(msg)
	return r.db.WithContext(ctx).Create(dbModel).Error
}

// compile-time interface check
var _ message.Repository = (*Repository)(nil)
