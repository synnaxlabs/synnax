// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package coalesce hands the latest value for each key from a goroutine that must not
// wait, such as the reader of a commit, to a goroutine that does the slow work for
// them.
package coalesce

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a Worker.
type Config[K comparable, V any] struct {
	// Handle does the work for the latest value of each key added since its previous
	// call. It runs on the goroutine of the Worker, one call at a time. Its context
	// ends when Close runs.
	//
	// [REQUIRED]
	Handle func(ctx context.Context, latest map[K]V)
	alamos.Instrumentation
}

var _ config.Config[Config[int, int]] = Config[int, int]{}

// Override implements config.Config.
func (c Config[K, V]) Override(other Config[K, V]) Config[K, V] {
	c.Handle = override.Nil(c.Handle, other.Handle)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c Config[K, V]) Validate() error {
	v := validate.New("coalesce")
	v.NotNil("handle", c.Handle)
	return v.Error()
}

// Worker calls Handle on a goroutine it owns with the values that Add receives. A value
// goes to a call of Handle that starts after the Add, and a later Add for the same key
// before that call replaces it. Safe for concurrent use.
type Worker[K comparable, V any] struct {
	cfg Config[K, V]
	mu  struct {
		sync.Mutex
		latest map[K]V
	}
	// wake holds a signal when mu.latest holds a value.
	wake     chan struct{}
	shutdown io.Closer
}

// Open starts a Worker. Close it to stop the goroutine.
func Open[K comparable, V any](cfgs ...Config[K, V]) (*Worker[K, V], error) {
	cfg, err := config.New(Config[K, V]{}, cfgs...)
	if err != nil {
		return nil, err
	}
	w := &Worker[K, V]{cfg: cfg, wake: make(chan struct{}, 1)}
	w.mu.latest = make(map[K]V)
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(w.run, signal.WithKey("coalesce"))
	w.shutdown = signal.NewHardShutdown(sCtx, cancel)
	return w, nil
}

// Add hands value for key to the Worker. It never waits for Handle.
func (w *Worker[K, V]) Add(key K, value V) {
	w.mu.Lock()
	w.mu.latest[key] = value
	w.mu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Close stops the Worker. Values that no call of Handle started on are not handled.
func (w *Worker[K, V]) Close() error { return w.shutdown.Close() }

func (w *Worker[K, V]) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.wake:
		}
		w.mu.Lock()
		latest := w.mu.latest
		w.mu.latest = make(map[K]V)
		w.mu.Unlock()
		if len(latest) > 0 {
			w.cfg.Handle(ctx, latest)
		}
	}
}
