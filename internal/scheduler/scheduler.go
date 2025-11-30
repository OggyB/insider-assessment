package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
)

// BatchProcessor is the dependency that actually does the work.
// The scheduler will call ProcessBatch on a fixed interval.
type BatchProcessor interface {
	ProcessBatch(ctx context.Context) error
}

// SchedulerService exposes a small control surface for the scheduler.
// Start/Stop are synchronous controls, and IsRunning reports
// whether the scheduler is currently accepting ticks.
type SchedulerService interface {
	Start() error
	Stop() error
	IsRunning() bool
}

// DefaultInterval is used when no custom interval is provided.
// This is the "safe fallback" value.
const DefaultInterval = 2 * time.Minute

// DefaultBatchTimeout is how long we allow a single batch to run
// before cancelling it via context timeout.
const DefaultBatchTimeout = 30 * time.Second

// controlTimeout is how long we wait for the control loop to
// accept a Start/Stop command and acknowledge it. This protects
// callers from hanging forever if the loop is not running.
const controlTimeout = 2 * time.Second

// controlOp represents the kind of command sent into the internal control loop.
type controlOp int

const (
	opStart controlOp = iota
	opStop
	opStatus
)

// controlMsg is sent over the ctrl channel to drive the scheduler's state.
type controlMsg struct {
	op   controlOp
	resp chan bool // used by callers to get a synchronous answer
}

// schedulerService owns the internal state and runs the control loop.
// All mutable state lives in the loop goroutine, so we don't need locks.
type schedulerService struct {
	messageService BatchProcessor
	interval       time.Duration
	batchTimeout   time.Duration
	ctrl           chan controlMsg
}

// NewSchedulerService creates a new scheduler with the given interval
// and batch timeout. If any of them is <= 0, sane defaults are used instead.
func NewSchedulerService(
	msgService BatchProcessor,
	interval time.Duration,
	batchTimeout time.Duration,
) SchedulerService {
	if interval <= 0 {
		interval = DefaultInterval
	}
	if batchTimeout <= 0 {
		batchTimeout = DefaultBatchTimeout
	}

	s := &schedulerService{
		messageService: msgService,
		interval:       interval,
		batchTimeout:   batchTimeout,
		ctrl:           make(chan controlMsg),
	}

	// The control loop is started in its own goroutine and lives
	// for the lifetime of the process.
	go s.loop()

	return s
}

// Start tells the scheduler to begin processing ticks.
// It blocks until the internal loop has acknowledged the state change,
// or returns an error if the control loop does not respond in time.
func (s *schedulerService) Start() error {
	resp := make(chan bool)
	msg := controlMsg{op: opStart, resp: resp}

	// First: make sure the control loop is actually listening
	// on the ctrl channel.
	select {
	case s.ctrl <- msg:
		// sent ok
	case <-time.After(controlTimeout):
		return fmt.Errorf("[Scheduler] Start: control loop not responding")
	}

	// Then: wait for the loop to acknowledge the state change.
	select {
	case <-resp:
		return nil
	case <-time.After(controlTimeout):
		return fmt.Errorf("[Scheduler] Start: acknowledgement timeout")
	}
}

// Stop tells the scheduler to stop accepting new ticks.
// If a batch is currently running, Stop will wait until that batch
// finishes (or times out) before returning. If the control loop does
// not respond, Stop returns an error instead of blocking forever.
func (s *schedulerService) Stop() error {
	resp := make(chan bool)
	msg := controlMsg{op: opStop, resp: resp}

	// Try to send the Stop command to the control loop.
	select {
	case s.ctrl <- msg:
		// sent ok
	case <-time.After(controlTimeout):
		return fmt.Errorf("[Scheduler] Stop: control loop not responding")
	}

	// Wait for the loop to confirm that it has stopped.
	select {
	case <-resp:
		return nil
	case <-time.After(controlTimeout):
		return fmt.Errorf("[Scheduler] Stop: acknowledgement timeout")
	}
}

// IsRunning reports whether the scheduler is currently in "running" mode.
// It does not mean that a batch is actively executing, only that new ticks
// will be processed when the timer fires.
func (s *schedulerService) IsRunning() bool {
	resp := make(chan bool)
	s.ctrl <- controlMsg{op: opStatus, resp: resp}
	return <-resp
}

// loop is the heart of the scheduler. It owns all mutable state
// and reacts to either control messages or timer ticks.
func (s *schedulerService) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// running: whether we should accept new ticks
	// inBatch: whether a batch is currently executing
	running := false
	inBatch := false

	// pendingStop is a response channel to be completed once
	// the current batch finishes, if Stop was called mid-batch.
	var pendingStop chan bool

	for {
		select {
		case msg := <-s.ctrl:
			switch msg.op {
			case opStart:
				if !running {
					log.Printf("[Scheduler] Started (interval=%s, batchTimeout=%s)\n",
						s.interval, s.batchTimeout)
				}
				running = true
				msg.resp <- true

			case opStop:
				// If we're already idle and not in a batch,
				// just acknowledge the Stop immediately.
				if !running && !inBatch {
					log.Println("[Scheduler] Stop requested, but already idle.")
					msg.resp <- true
					continue
				}

				log.Println("[Scheduler] Stop requested. Waiting for current batch (if any)...")

				// Mark as not running so future ticks are ignored.
				running = false

				if inBatch {
					// Defer the response until the batch completes.
					pendingStop = msg.resp
				} else {
					// No active batch, we can safely stop now.
					msg.resp <- true
				}

			case opStatus:
				msg.resp <- running
			}

		case <-ticker.C:
			// If we're not running or already processing a batch,
			// ignore this tick.
			if !running || inBatch {
				continue
			}

			inBatch = true
			log.Println("[Scheduler] Triggering batch...")

			// Time-bound the batch execution so Stop doesn't hang forever
			// if ProcessBatch never returns.
			ctx, cancel := context.WithTimeout(context.Background(), s.batchTimeout)

			err := s.messageService.ProcessBatch(ctx)
			cancel()

			if err != nil {
				log.Printf("[Scheduler] Batch failed: %v\n", err)
			} else {
				log.Println("[Scheduler] Batch completed.")
			}

			inBatch = false

			// If a Stop was requested while we were in a batch,
			// complete it now and clear the pending channel.
			if pendingStop != nil {
				pendingStop <- true
				pendingStop = nil
				log.Println("[Scheduler] Stopped (no active batch).")
			}
		}
	}
}
