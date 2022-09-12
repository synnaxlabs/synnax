package persist

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/operation"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/kfs"
	override "github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
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

type Config struct {
	// NumWorkers represents the number of goroutines that will be used to execute operations.
	// This value must be at least 1.
	NumWorkers int
}

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.NumWorkers = override.Numeric(cfg.NumWorkers, other.NumWorkers)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("cesium.persist")
	validate.GreaterThan(v, "NumWorkers", cfg.NumWorkers, 0)
	return v.Error()
}

var DefaultConfig = Config{NumWorkers: 10}

// New creates a new Persist that wraps the provided kfs.FS.
func New[F comparable, O operation.Operation[F]](kfs kfs.FS[F], _cfg Config) (*Persist[F, O], error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, _cfg)
	p := &Persist[F, O]{kfs: kfs, Config: cfg, ops: make(chan O, cfg.NumWorkers)}
	return p, err
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
