package ding

import (
	"context"
	"sync"
)

// Ring represents the priority of a task. Lower numbers indicate higher priority meaning they will be canceled later.
// For example, Ring 0 is the highest priority and will be canceled last,
// while Ring 3 is the lowest priority and will be canceled first.
type Ring int

const (
	// Ring 0 is the highest priority, it will get canceled last
	Ring0 Ring = iota
	// Ring 1 is the second highest priority, it will get canceled before Ring 0
	Ring1
	// Ring 2 is the third highest priority, it will get canceled before Ring 1
	Ring2
	// Ring 3 is the lowest priority, it will get canceled first
	Ring3
)

const (
	// RingCritical is the highest priority, it will get canceled last
	RingCritical = Ring0
	// RingMajor is the second highest priority, it will get canceled before RingCritical
	RingMajor = Ring1
	// RingNormal is the third highest priority, it will get canceled before RingMajor
	RingNormal = Ring2
	// RingMinor is the lowest priority, it will get canceled first
	RingMinor = Ring3
)

// dingRing is a struct that represents a single ring of tasks.
// It contains the ring's priority, context, cancel function, and wait group.
type dingRing struct {
	ring   Ring
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Ding is a struct that manages multiple rings of tasks with different priorities.
// Each ring has its own context and wait group.
type Ding struct {
	rings []*dingRing
}

// New creates a new Ding instance with the provided context.
// It initializes the rings and starts a goroutine to handle cancellation when the context is done.
func New(ctx context.Context) *Ding {
	ding := &Ding{
		rings: make([]*dingRing, 0),
	}
	cleanCtx := context.WithoutCancel(ctx)
	for _, ring := range []Ring{RingCritical, RingMajor, RingNormal, RingMinor} {
		ctx, cancel := context.WithCancel(cleanCtx)
		ding.rings = append(ding.rings, &dingRing{
			ring:   ring,
			ctx:    ctx,
			cancel: cancel,
			wg:     sync.WaitGroup{},
		})
	}
	go func() {
		<-ctx.Done()
		for i := len(ding.rings) - 1; i >= 0; i-- {
			ding.rings[i].cancel()
			ding.rings[i].wg.Wait()
		}
	}()
	return ding
}

// Go starts a new goroutine for the provided function f with the context of the specified ring.
// It works in the same way the standard library's sync.WaitGroup works, but it also takes
// into account the priority of the ring and cancels the context of the ring when the main context is done.
// Unlike the standard library's sync.WaitGroup, Ding's Go function also provides the f function with a
// context to watch so it can gracefully shutdown.
func (d *Ding) Go(f func(ctx context.Context), ring Ring) {
	d.rings[ring].wg.Add(1)
	go func() {
		defer func() {
			d.rings[ring].wg.Done()
		}()
		f(d.rings[ring].ctx)
	}()
}

// Wait blocks until all tasks in all rings have completed.
func (d *Ding) Wait() {
	for i := len(d.rings) - 1; i >= 0; i-- {
		d.rings[i].wg.Wait()
	}
}
