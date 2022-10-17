package persist

import (
	"context"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/kfs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type Operation[F comparable] interface {
	Target() F
	Exec(f kfs.File[F], err error)
}

// Persist wraps a kfs.KFS and provides a mechanism for easily executing operations on it.
// Persist uses a pool of goroutines to execute operations concurrently.
// To create a new Persist, use persist.New.
type Persist[K comparable] struct {
	kfs kfs.FS[K]
	ops chan Operation[K]
	Config
	confluence.UnarySink[[]Operation[K]]
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
func New[K comparable](kfs kfs.FS[K], _cfg Config) (*Persist[K], error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, _cfg)
	p := &Persist[K]{kfs: kfs, Config: cfg, ops: make(chan Operation[K], cfg.NumWorkers)}
	return p, err
}

func (p *Persist[K]) Flow(ctx signal.Context, opts ...confluence.Option) {
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

func (p *Persist[K]) start(ctx signal.Context) {
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
					f, err := p.kfs.Acquire(op.Target())
					op.Exec(f, err)
					p.kfs.Release(op.Target())
				}

			}
		}, signal.WithKeyf("persist.worker.%d", i))
	}
}
