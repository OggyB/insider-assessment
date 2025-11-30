package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/oggyb/insider-assessment/internal/config"
	"github.com/oggyb/insider-assessment/internal/db/gormdb"
	domain "github.com/oggyb/insider-assessment/internal/domain/message"
	mesgRepo "github.com/oggyb/insider-assessment/internal/repository/gorm/message"
	"gorm.io/gorm"
)

func main() {
	ctx := context.Background()

	// Load application configuration (DB, Redis, etc.) from env/.env.
	cfg := config.New()

	// Open a Postgres connection through our GORM adapter.
	gormAdapter, err := gormdb.New(cfg.PostgresDSN())
	if err != nil {
		log.Fatalf("[Seed] Failed to connect to database: %v", err)
	}

	log.Printf("[Seed] Connected to database %q", cfg.DB.Name)

	// 1) AutoMigrate: make sure the messages table exists.
	// We go through the adapter to access the underlying *gorm.DB.
	rawDB := gormAdapter.Conn().(*gorm.DB)

	if err := rawDB.AutoMigrate(&mesgRepo.MessageModel{}); err != nil {
		log.Fatalf("[Seed] AutoMigrate failed: %v", err)
	}
	log.Println("[Seed] Messages table is up to date (AutoMigrate completed).")

	// 2) Primitive seeding: always insert N random PENDING messages.
	const seedCount = 50

	// The repository expects a db.DB interface, so we pass the adapter,
	// not the raw *gorm.DB.
	repo := mesgRepo.NewRepository(gormAdapter)

	log.Printf("[Seed] Inserting %d random messages...", seedCount)

	for i := 0; i < seedCount; i++ {
		to := randomPhone()
		content := randomContent(i + 1)

		// Use the domain constructor so we respect domain rules:
		// status = PENDING, timestamps, etc.
		msg, _ := domain.NewMessage(to, content)

		if err := repo.Save(ctx, msg); err != nil {
			log.Fatalf("[Seed] Failed to save message #%d: %v", i+1, err)
		}

		log.Printf("[Seed] Created message #%d: id=%s to=%s",
			i+1, msg.ID.String(), msg.To)
	}

	log.Printf("[Seed] Done. Inserted %d messages into table 'messages'.", seedCount)
}

// randomPhone generates a simple fake phone number in an E.164-like format.
// Example output: +905123456789
func randomPhone() string {
	base := "+905"
	n := rand.Intn(900000000) + 100000000 // 9 digits
	return fmt.Sprintf("%s%d", base, n)
}

// randomContent generates a simple SMS body for seeding.
func randomContent(i int) string {
	now := time.Now().Format("15:04:05")
	return fmt.Sprintf("Seed message #%d sent at %s", i, now)
}
