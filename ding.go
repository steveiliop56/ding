package ding

import (
	"context"
	"sync"
)

type Ring int

const (
	Ring0 Ring = iota
	Ring1
	Ring2
	Ring3
)

// Better names for the rings
const (
	RingCritical = Ring0
	RingMajor    = Ring1
	RingNormal   = Ring2
	RingMinor    = Ring3
)

type dingRing struct {
	ring   Ring
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

type Ding struct {
	rings []*dingRing
}

func New(ctx context.Context) *Ding {
	ding := &Ding{
		rings: make([]*dingRing, 0),
	}
	for _, ring := range []Ring{RingCritical, RingMajor, RingNormal, RingMinor} {
		ctx, cancel := context.WithCancel(context.Background())
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

func (d *Ding) Go(f func(ctx context.Context), ring Ring) {
	d.rings[ring].wg.Add(1)
	go func() {
		defer func() {
			d.rings[ring].wg.Done()
		}()
		f(d.rings[ring].ctx)
	}()
}

func (d *Ding) Wait() {
	for i := len(d.rings) - 1; i >= 0; i-- {
		d.rings[i].wg.Wait()
	}
}
