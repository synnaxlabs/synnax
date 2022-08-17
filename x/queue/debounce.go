package queue

import (
	"context"
	"github.com/arya-analytics/x/confluence"
	"github.com/arya-analytics/x/signal"
	"time"
)

type DebounceConfig struct {
	// Interval is the time between flushes.
	Interval time.Duration
	// Threshold is the maximum number of values to store in Debounce.
	// Debounce will flush when this threshold is reached, regardless of the Interval.
	Threshold int
}

// Debounce is a simple, goroutine safe queue that flushes data to a channel on a timer or queue size threshold.
type Debounce[V confluence.Value] struct {
	Config DebounceConfig
	confluence.LinearTransform[[]V, []V]
}

// Flow starts the queue.
func (d *Debounce[V]) Flow(ctx signal.Context, opts ...confluence.Option) {
	fo := confluence.NewOptions(opts)
	ctx.Go(func(ctx context.Context) error {
		var (
			t = time.NewTicker(d.Config.Interval)
		)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			values, ok := d.fill(t.C)
			if !ok {
				return nil
			}
			if len(values) == 0 {
				continue
			}
			d.Out.Inlet() <- values
		}
	}, fo.Signal...)
}

func (d *Debounce[V]) fill(C <-chan time.Time) ([]V, bool) {
	ops := make([]V, 0, d.Config.Threshold)
	for {
		select {
		case values, ok := <-d.In.Outlet():
			if !ok {
				return ops, false
			}
			ops = append(ops, values...)
			if len(ops) >= d.Config.Threshold {
				return ops, true
			}
		case <-C:
			return ops, true
		}
	}
}
