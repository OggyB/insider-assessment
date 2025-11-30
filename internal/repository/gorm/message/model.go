package messagegorm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MessageModel is the GORM persistence model for messages.
// It maps directly to the "messages" table in Postgres.
type MessageModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	To          string     `gorm:"size:20;not null"`
	Content     string     `gorm:"size:255;not null"`
	Status      string     `gorm:"size:20;not null"`
	RawResponse string     `gorm:"type:text"`
	MessageID   string     `gorm:"size:100;index"`
	SentAt      *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"not null;index"`
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName overrides the default table name used by GORM.
func (MessageModel) TableName() string {
	return "messages"
}

// BeforeCreate ensures a UUID is set before inserting a new record.
func (m *MessageModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	return nil
}
