// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package coalesce hands keys from a goroutine that must not wait, such as the reader
// of a commit, to a goroutine that does the slow work for them.
package coalesce

import (
	"context"
	"io"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a Worker.
type Config[K comparable] struct {
	// Handle does the work for the keys added since its previous call. It runs on the
	// goroutine of the Worker, one call at a time. Its context ends when Close runs.
	//
	// [REQUIRED]
	Handle func(ctx context.Context, keys set.Set[K])
	alamos.Instrumentation
}

var _ config.Config[Config[int]] = Config[int]{}

// Override implements config.Config.
func (c Config[K]) Override(other Config[K]) Config[K] {
	c.Handle = override.Nil(c.Handle, other.Handle)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements config.Config.
func (c Config[K]) Validate() error {
	v := validate.New("coalesce")
	v.NotNil("handle", c.Handle)
	return v.Error()
}

// Worker calls Handle on a goroutine it owns with the keys that Add receives. A key
// goes to a call of Handle that starts after the Add, and keys added more than once
// before that call go to it once. Safe for concurrent use.
type Worker[K comparable] struct {
	cfg Config[K]
	mu  struct {
		sync.Mutex
		keys set.Set[K]
	}
	// wake holds a signal when mu.keys holds a key.
	wake     chan struct{}
	shutdown io.Closer
}

// Open starts a Worker. Close it to stop the goroutine.
func Open[K comparable](cfgs ...Config[K]) (*Worker[K], error) {
	cfg, err := config.New(Config[K]{}, cfgs...)
	if err != nil {
		return nil, err
	}
	w := &Worker[K]{cfg: cfg, wake: make(chan struct{}, 1)}
	w.mu.keys = make(set.Set[K])
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	sCtx.Go(w.run, signal.WithKey("coalesce"))
	w.shutdown = signal.NewHardShutdown(sCtx, cancel)
	return w, nil
}

// Add hands keys to the Worker. It never waits for Handle.
func (w *Worker[K]) Add(keys ...K) {
	if len(keys) == 0 {
		return
	}
	w.mu.Lock()
	w.mu.keys.Add(keys...)
	w.mu.Unlock()
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

// Close stops the Worker. Keys that no call of Handle started on are not handled.
func (w *Worker[K]) Close() error { return w.shutdown.Close() }

func (w *Worker[K]) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-w.wake:
		}
		w.mu.Lock()
		keys := w.mu.keys
		w.mu.keys = make(set.Set[K])
		w.mu.Unlock()
		if len(keys) > 0 {
			w.cfg.Handle(ctx, keys)
		}
	}
}
