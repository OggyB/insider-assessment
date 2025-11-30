package scheduler

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeBatchProcessor is a test double that counts ProcessBatch calls,
// signals when the first batch starts, and can block until explicitly released.
type fakeBatchProcessor struct {
	callCount int32

	started chan struct{} // signals when a batch starts (first call only)
	block   chan struct{} // keeps ProcessBatch blocked until closed
}

func newFakeBatchProcessor() *fakeBatchProcessor {
	return &fakeBatchProcessor{
		started: make(chan struct{}, 1),
		block:   make(chan struct{}),
	}
}

func (f *fakeBatchProcessor) ProcessBatch(ctx context.Context) error {
	atomic.AddInt32(&f.callCount, 1)

	// Signal "started" only once (non-blocking).
	select {
	case f.started <- struct{}{}:
	default:
	}

	// Wait until either the test releases the block or the context is done.
	select {
	case <-f.block:
	case <-ctx.Done():
	}

	return nil
}

func (f *fakeBatchProcessor) Calls() int32 {
	return atomic.LoadInt32(&f.callCount)
}

func TestScheduler_StartTriggersBatch(t *testing.T) {
	fake := newFakeBatchProcessor()

	// Short tick interval, reasonably long batch timeout so we don't hit it in this test.
	s := NewSchedulerService(fake, 10*time.Millisecond, 2*time.Second)

	// Depending on your current interface, this may be:
	//   _ = s.Start()
	// veya
	//   if err := s.Start(); err != nil { ... }
	s.Start()
	defer s.Stop()

	// We expect ProcessBatch to be triggered shortly after Start.
	select {
	case <-fake.started:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected ProcessBatch to be called after Start, but it wasn't")
	}

	if !s.IsRunning() {
		t.Fatalf("expected scheduler to be running after Start()")
	}
}

func TestScheduler_StopWaitsForBatchCompletion(t *testing.T) {
	fake := newFakeBatchProcessor()

	// Very frequent ticks, but long enough batch timeout so ctx doesn't kill the batch
	// before we manually unblock it.
	s := NewSchedulerService(fake, 5*time.Millisecond, 2*time.Second)

	s.Start()

	// Wait until the first batch actually starts so Stop happens mid-batch.
	select {
	case <-fake.started:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("ProcessBatch was not called in time")
	}

	// Call Stop in a separate goroutine so we can assert it blocks.
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	// Stop should NOT return immediately while the batch is still blocked.
	select {
	case <-done:
		t.Fatalf("Stop() returned before batch finished")
	case <-time.After(50 * time.Millisecond):
		// good: Stop is still waiting for the batch to complete
	}

	// Now let the batch complete.
	close(fake.block)

	// After unblocking the batch, Stop should return in a reasonable time.
	select {
	case <-done:
		// ok
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("Stop() did not return after batch completion")
	}

	if s.IsRunning() {
		t.Fatalf("expected scheduler to not be running after Stop()")
	}
}

func TestScheduler_StartStopStartFlow(t *testing.T) {
	fake := newFakeBatchProcessor()
	s := NewSchedulerService(fake, 10*time.Millisecond, 2*time.Second)

	// 1) First start
	s.Start()
	select {
	case <-fake.started:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("first Start: ProcessBatch was not called")
	}

	// Release the first batch.
	close(fake.block)

	// Stop the scheduler.
	s.Stop()
	if s.IsRunning() {
		t.Fatalf("scheduler should be stopped after Stop()")
	}

	// Prepare a new block channel for the next batch.
	fake.block = make(chan struct{})

	// 2) Start again
	s.Start()
	if !s.IsRunning() {
		t.Fatalf("scheduler should be running after second Start()")
	}

	// We expect another batch to be triggered.
	select {
	case <-fake.started:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("second Start: ProcessBatch was not called")
	}
}

func TestScheduler_RaceStartStop(t *testing.T) {
	fake := newFakeBatchProcessor()
	s := NewSchedulerService(fake, 5*time.Millisecond, 50*time.Millisecond)

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			_ = s.Start()
		}()

		go func() {
			defer wg.Done()
			_ = s.Stop()
		}()
	}

	wg.Wait()
}
