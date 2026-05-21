// Command rings demonstrates how ding shuts workers down in priority order.
//
// Workers are placed on each of the four rings. When the parent context is
// cancelled, ding drains the rings from the lowest priority (RingMinor) to
// the highest (RingCritical), waiting for each ring to finish before moving
// on to the next. This lets you, for example, stop accepting new requests
// before flushing critical state.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/steveiliop56/ding"
)

func worker(name string, shutdownDelay time.Duration) func(context.Context) {
	return func(c context.Context) {
		<-c.Done()
		// Simulate cleanup work that takes a little time.
		time.Sleep(shutdownDelay)
		fmt.Printf("%s stopped\n", name)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	d := ding.New(ctx)

	d.Go(worker("critical (ring 0)", 100*time.Millisecond), ding.RingCritical)
	d.Go(worker("major    (ring 1)", 100*time.Millisecond), ding.RingMajor)
	d.Go(worker("normal   (ring 2)", 100*time.Millisecond), ding.RingNormal)
	d.Go(worker("minor    (ring 3)", 100*time.Millisecond), ding.RingMinor)

	fmt.Println("workers started, triggering shutdown...")
	cancel()

	// Wait for the ordered shutdown to complete. Expect the output to run
	// from minor -> normal -> major -> critical.
	d.Wait()
	fmt.Println("shutdown complete")
}
