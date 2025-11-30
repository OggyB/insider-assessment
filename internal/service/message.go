package service

import (
	"context"
	"fmt"
	"github.com/oggyb/insider-assessment/internal/cache"
	domain "github.com/oggyb/insider-assessment/internal/domain/message"
	"github.com/oggyb/insider-assessment/internal/sms"
	"log"
	"sync"
	"time"
)

type MessageService interface {
	GetSent(ctx context.Context, page, limit int) ([]*domain.Message, int64, error)
	ProcessBatch(ctx context.Context) error
}

type messageService struct {
	repo      domain.Repository
	smsClient sms.Client
	cache     cache.Cache

	// Batch processing configuration, injected from config at startup.
	batchSize         int
	maxWorkers        int
	perMessageTimeout time.Duration
}

// NewMessageService creates a message service with the given dependencies
// and batch processing settings. The config values are passed explicitly
// from the caller (e.g. main) so this package does not depend on env.
func NewMessageService(
	repo domain.Repository,
	smsClient sms.Client,
	cache cache.Cache,
	batchSize int,
	maxWorkers int,
	perMessageTimeout time.Duration,
) MessageService {
	// Apply sane defaults if config values are missing or invalid.
	if batchSize <= 0 {
		batchSize = 100
	}
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	if perMessageTimeout <= 0 {
		perMessageTimeout = 5 * time.Second
	}

	return &messageService{
		repo:              repo,
		smsClient:         smsClient,
		cache:             cache,
		batchSize:         batchSize,
		maxWorkers:        maxWorkers,
		perMessageTimeout: perMessageTimeout,
	}
}

func (s *messageService) GetSent(ctx context.Context, page, limit int) ([]*domain.Message, int64, error) {
	return s.repo.GetSent(ctx, page, limit)
}

// ProcessBatch pulls a batch of pending messages from the repository and
// processes them using a small worker pool. The batch size, worker count
// and per-message timeout are provided at construction time.
func (s *messageService) ProcessBatch(ctx context.Context) error {
	batchSize := s.batchSize
	maxWorkers := s.maxWorkers
	perMessageTimeout := s.perMessageTimeout

	// Fetch pending messages from the repository.
	messages, err := s.repo.GetPending(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch pending messages: %w", err)
	}

	// Nothing to do; exit quickly so the scheduler can tick again.
	if len(messages) == 0 {
		log.Println("[Service] No pending messages to process.")
		return nil
	}

	log.Printf(
		"[Service] Processing %d messages with worker pool (batchSize=%d, maxWorkers=%d)...",
		len(messages), batchSize, maxWorkers,
	)

	// Decide how many workers we need for this batch.
	workerCount := len(messages)
	if workerCount > maxWorkers {
		workerCount = maxWorkers
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	var wg sync.WaitGroup

	// Simple worker pool: each worker processes a "stride" of messages.
	// For example, with 4 workers:
	//   worker 1: indices 0, 4, 8, ...
	//   worker 2: indices 1, 5, 9, ...
	//   worker 3: indices 2, 6, 10, ...
	//   worker 4: indices 3, 7, 11, ...
	for w := 0; w < workerCount; w++ {
		wg.Add(1)

		go func(workerID, start int) {
			defer wg.Done()

			for i := start; i < len(messages); i += workerCount {
				// If the parent context has been cancelled (e.g. by the scheduler),
				// stop processing new messages and exit this worker.
				if ctx.Err() != nil {
					log.Printf("[Worker %d] Context cancelled, stopping worker", workerID)
					return
				}

				msg := messages[i]

				// Wrap the parent context with a per-message timeout.
				msgCtx, cancel := context.WithTimeout(ctx, perMessageTimeout)

				log.Printf("[Worker %d] is processing.", i)
				if err := s.processMessage(msgCtx, msg); err != nil {
					log.Printf("[Worker %d] Failed to process %s: %v",
						workerID, msg.ID.String(), err)
				}

				// Make sure we always release the derived context.
				cancel()
			}
		}(w+1, w)
	}

	// Wait until all workers have finished processing their share.
	wg.Wait()

	log.Println("[Service] Batch worker pool completed.")
	return nil
}

// processMessage sends a single pending message via the SMS provider and
// updates its status in the repository.
//
// Flow:
//   - Call the SMS client with the message content and recipient.
//   - On failure: mark the message as FAILED and persist this status.
//   - On success: mark the message as SUCCESS, persist it, and optionally
//     cache the sent timestamp in Redis for quick lookup.
//
// The provided context may be cancelled or time out by the caller (e.g. the
// scheduler), in which case the send operation should respect that.
func (s *messageService) processMessage(ctx context.Context, msg *domain.Message) error {
	id := msg.ID.String()

	// Try to send the message via the external SMS provider.
	externalID, rawResp, err := s.smsClient.Send(ctx, msg.To, msg.Content)
	if err != nil {
		log.Printf("[Service] Failed to send message %s: %v. Marking as FAILED.", id, err)
		msg.MarkFailed(rawResp)

		// Best-effort: persist the FAILED status so this message is not retried
		// indefinitely as PENDING.
		if uErr := s.repo.UpdateStatus(ctx, msg); uErr != nil {
			log.Printf("[Service] Failed to persist FAILED status for %s: %v", id, uErr)
		}

		return fmt.Errorf("send message %s: %w", id, err)
	}

	// Mark as successfully sent and persist the new state.
	msg.MarkSent(externalID, rawResp)
	if err := s.repo.UpdateStatus(ctx, msg); err != nil {
		log.Printf("[Service] Failed to persist SUCCESS status for %s: %v", id, err)
		return fmt.Errorf("update status for %s: %w", id, err)
	}

	// Optionally cache the sent timestamp in Redis keyed by external message ID.
	if s.cache != nil && externalID != "" {
		sentAt := time.Now().Format(time.RFC3339)
		if msg.SentAt != nil {
			sentAt = msg.SentAt.Format(time.RFC3339)
		}

		key := cache.SentMessages.Key(externalID)
		if err := s.cache.Set(ctx, key, sentAt, 24*time.Hour); err != nil {
			log.Printf("[Service] Failed to cache in Redis for %s: %v", externalID, err)
		}
	}

	return nil
}
