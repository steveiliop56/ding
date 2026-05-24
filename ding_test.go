package ding

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNew verifies that New creates a Ding with one ring per priority level.
func TestNew(t *testing.T) {
	d := New(context.Background())

	if d == nil {
		t.Fatal("New returned nil")
	}
	if len(d.rings) != 4 {
		t.Fatalf("expected 4 rings, got %d", len(d.rings))
	}

	want := []Ring{RingCritical, RingMajor, RingNormal, RingMinor}
	for i, r := range d.rings {
		if r.ring != want[i] {
			t.Errorf("ring %d: got %d, want %d", i, r.ring, want[i])
		}
		if r.ctx == nil || r.cancel == nil {
			t.Errorf("ring %d: ctx or cancel is nil", i)
		}
	}
}

// TestRingAliases checks the human friendly names map onto the right values.
func TestRingAliases(t *testing.T) {
	cases := map[Ring]Ring{
		RingCritical: Ring0,
		RingMajor:    Ring1,
		RingNormal:   Ring2,
		RingMinor:    Ring3,
	}
	for alias, want := range cases {
		if alias != want {
			t.Errorf("alias %d != %d", alias, want)
		}
	}
}

// TestGoAndWait runs work that finishes on its own and confirms Wait blocks
// until every goroutine has returned.
func TestGoAndWait(t *testing.T) {
	d := New(context.Background())

	var count atomic.Int32
	for range 10 {
		d.Go(func(ctx context.Context) {
			count.Add(1)
		}, RingNormal)
	}

	d.Wait()

	if got := count.Load(); got != 10 {
		t.Errorf("expected 10 completed goroutines, got %d", got)
	}
}

// TestShutdownOrder verifies that cancelling the parent context drains the
// rings from the lowest priority (RingMinor) to the highest (RingCritical),
// waiting for each ring before moving to the next.
func TestShutdownOrder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := New(ctx)

	// Records the order in which rings finish shutting down.
	var mu sync.Mutex
	var order []Ring

	// A worker on every ring blocks until its own context is cancelled,
	// then records which ring it belonged to.
	var started sync.WaitGroup
	for _, ring := range []Ring{RingCritical, RingMajor, RingNormal, RingMinor} {
		started.Add(1)
		d.Go(func(c context.Context) {
			started.Done()
			<-c.Done()
			mu.Lock()
			order = append(order, ring)
			mu.Unlock()
		}, ring)
	}

	// Make sure all workers are parked on <-c.Done() before we cancel,
	// otherwise the recorded order would be unreliable.
	started.Wait()
	cancel()
	d.Wait()

	want := []Ring{RingMinor, RingNormal, RingMajor, RingCritical}
	mu.Lock()
	defer mu.Unlock()
	if len(order) != len(want) {
		t.Fatalf("expected %d rings to shut down, got %d", len(want), len(order))
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("shutdown step %d: got ring %d, want ring %d", i, order[i], want[i])
		}
	}
}

// TestContextPropagation ensures the context handed to a worker is cancelled
// when the parent context is cancelled.
func TestContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	d := New(ctx)

	done := make(chan struct{})
	d.Go(func(c context.Context) {
		<-c.Done()
		close(done)
	}, RingCritical)

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker context was not cancelled within timeout")
	}

	d.Wait()
}

// Test KV store functionality
func TestKVStore(t *testing.T) {
	ctx := context.WithValue(context.Background(), "foo", "bar")
	d := New(ctx)

	done := make(chan struct{})
	d.Go(func(c context.Context) {
		if v := c.Value("foo"); v != "bar" {
			t.Errorf("expected context value 'bar', got '%v'", v)
		}
		close(done)
	}, RingNormal)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not complete within timeout")
	}

	d.Wait()
}
