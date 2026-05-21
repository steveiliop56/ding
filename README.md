# ding

[![CI](https://github.com/steveiliop56/ding/actions/workflows/ci.yml/badge.svg)](https://github.com/steveiliop56/ding/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/steveiliop56/ding.svg)](https://pkg.go.dev/github.com/steveiliop56/ding)

`ding` is a tiny Go library for **graceful, ordered shutdown** of background
goroutines.

Goroutines are grouped into priority **rings**. When you trigger a shutdown,
`ding` cancels and drains the rings one at a time, from the lowest priority to
the highest, waiting for each ring to fully finish before moving on to the
next. This lets you control teardown order — for example, stop accepting new
requests before flushing critical state to disk.

## Installation

```sh
go get github.com/steveiliop56/ding
```

## Rings

There are four rings, ordered from highest to lowest priority. Each has a
numeric name and a friendlier alias:

| Alias          | Value   | Shuts down |
| -------------- | ------- | ---------- |
| `RingCritical` | `Ring0` | last       |
| `RingMajor`    | `Ring1` | third      |
| `RingNormal`   | `Ring2` | second     |
| `RingMinor`    | `Ring3` | first      |

Lower-priority rings (`RingMinor` first) are drained before higher-priority
ones, so your most important work gets the most time to wind down.

## Usage

```go
package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/steveiliop56/ding"
)

func main() {
	// Cancel the context on Ctrl+C / SIGTERM to start the shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	d := ding.New(ctx)

	// Critical work drains last.
	d.Go(func(c context.Context) {
		<-c.Done()
		fmt.Println("flushing state...")
	}, ding.RingCritical)

	// Less important work drains first.
	d.Go(func(c context.Context) {
		<-c.Done()
		fmt.Println("stopping metrics reporter...")
	}, ding.RingMinor)

	// Block until every ring has drained.
	d.Wait()
}
```

## API

- `New(ctx context.Context) *Ding` — creates a `Ding`. When `ctx` is cancelled,
  the rings are drained in priority order.
- `(*Ding) Go(f func(ctx context.Context), ring Ring)` — runs `f` on the given
  ring. The `ctx` passed to `f` is cancelled when that ring shuts down.
- `(*Ding) Wait()` — blocks until every ring has finished.

## Examples

Runnable examples live in the [`examples`](./examples) directory:

- [`examples/basic`](./examples/basic) — a single worker stopped by a signal.
- [`examples/rings`](./examples/rings) — workers on every ring, showing the
  shutdown order.

```sh
go run ./examples/rings
```

## Development

```sh
go test -race -cover ./...   # run tests with the race detector and coverage
go vet ./...                 # static analysis
```

## License

[MIT](./LICENSE)
