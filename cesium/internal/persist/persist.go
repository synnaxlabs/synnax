package persist

import (
	"context"
	"github.com/arya-analytics/cesium/internal/operation"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/kfs"
	"github.com/arya-analytics/x/signal"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

// Persist wraps a kfs.KFS and provides a mechanism for easily executing operations on it.
// Persist uses a pool of goroutines to execute operations concurrently.
// To create a new Persist, use persist.New.
type Persist[F comparable, O operation.Operation[F]] struct {
	sem *semaphore.Weighted
	kfs kfs.FS[F]
	ops chan O
	Config
	confluence.UnarySink[[]O]
}

const (
	// DefaultNumWorkers is the default maximum number of goroutines Persist can operate at once.
	DefaultNumWorkers = 10
)

type Config struct {
	// NumWorkers represents the number of goroutines that will be used to execute operations.
	// This value must be at least 1.
	NumWorkers int
	// Persist will log events to this Logger.
	Logger *zap.Logger
}

func DefaultConfig() Config { return Config{NumWorkers: DefaultNumWorkers} }

// New creates a new Persist that wraps the provided kfs.FS.
func New[F comparable, O operation.Operation[F]](kfs kfs.FS[F], config Config) *Persist[F, O] {
	p := &Persist[F, O]{kfs: kfs, Config: config, ops: make(chan O, config.NumWorkers)}
	return p
}

func (p *Persist[K, O]) Flow(ctx signal.Context, opts ...confluence.Option) {
	p.start(ctx)
	o := confluence.NewOptions(opts)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case ops, ok := <-p.In.Outlet():
				if !ok {
					close(p.ops)
					return nil
				}
				for _, op := range ops {
					p.ops <- op
				}
			}
		}
	}, append(o.Signal, signal.WithKey("persist.relay"))...)
}

func (p *Persist[K, O]) start(ctx signal.Context) {
	for i := 0; i < p.NumWorkers; i++ {
		ctx.Go(func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case op, ok := <-p.ops:
					if !ok {
						return nil
					}
					f, err := p.kfs.Acquire(op.FileKey())
					if err != nil {
						op.WriteError(err)
						continue
					}
					op.Exec(f)
					p.kfs.Release(op.FileKey())
				}

			}
		}, signal.WithKeyf("persist.worker.%d", i))
	}
}
