// Command basic demonstrates the smallest useful ding setup: a couple of
// background workers that are cancelled and drained when the program shuts
// down.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/steveiliop56/ding"
)

func main() {
	// The parent context is cancelled on SIGINT/SIGTERM, which kicks off the
	// graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	d := ding.New(ctx)

	// A worker that does some periodic work until told to stop.
	d.Go(func(c context.Context) {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-c.Done():
				fmt.Println("worker: shutting down cleanly")
				return
			case <-ticker.C:
				fmt.Println("worker: doing work")
			}
		}
	}, ding.RingNormal)

	fmt.Println("running... press Ctrl+C to stop")

	// Wait blocks until every ring has drained.
	d.Wait()
	fmt.Println("all workers stopped, exiting")

	os.Exit(0)
}
