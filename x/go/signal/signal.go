// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"context"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// Context is an extension of the standard context.Context that provides a way to signal
// a goroutine to maybeStop.
type Context interface {
	context.Context
	Go
	WaitGroup
	Census
}

// WithCancel returns a Context derived from core that is canceled by the given cancel
// function. If any goroutine returns a non-nil error, the Context will be canceled.
func WithCancel(ctx context.Context, opts ...Option) (Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	c := newCore(ctx, cancel, opts...)
	return c, cancel
}

// WithTimeout returns a Context derived from core that is canceled by the given
// timeout. If any goroutine returns a non-nil error, the Context will be canceled.
func WithTimeout(
	ctx context.Context,
	d time.Duration,
	opts ...Option,
) (Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, d)
	c := newCore(ctx, cancel, opts...)
	return c, cancel
}

// Isolated opens a new signal context completely isolated from any parent contexts.
// This is equivalent to context.Background(), and is used to manage and independent set
// of go-routines.
func Isolated(opts ...Option) (Context, context.CancelFunc) {
	return WithCancel(context.Background(), opts...)
}

// Wrap uses the context as the underlying context for the returned signal Context. When
// the passed in context is cancelled, the signal context will be cancelled.
func Wrap(ctx context.Context, opts ...Option) Context {
	return newCore(ctx, func() {}, opts...)
}

func newCore(
	ctx context.Context,
	cancel context.CancelFunc,
	opts ...Option,
) *core {
	c := &core{options: newOptions(opts), Context: ctx, cancel: cancel}
	c.mu.stopped = make(chan struct{})
	return c
}

type core struct {
	options
	context.Context
	cancel   context.CancelFunc
	internal errgroup.Group
	mu       struct {
		stopped  chan struct{}
		routines []*routine
		sync.RWMutex
	}
}

// Stopped returns a channel that is closed when all routines in the context have
// stopped. This can be used to wait for all routines to complete.
func (c *core) Stopped() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mu.stopped
}

// maybeStop checks if all routines have stopped and closes the stopped channel if they
// have. This is called after each routine completes to potentially signal that all
// routines are done.
func (c *core) maybeStop() {
	select {
	case <-c.mu.stopped:
		return
	default:
		for _, r := range c.mu.routines {
			if r.state.state == Starting || r.state.state == Running {
				return
			}
		}
		close(c.mu.stopped)
	}
}

// routineDiagnostics returns a formatted string containing information about all
// routines in the context, including their keys, states, and any failure reasons.
func (c *core) routineDiagnostics() string {
	var b strings.Builder
	for _, i := range c.routines() {
		b.WriteString(i.PrettyString())
	}
	return b.String()
}

// unsafeRunningKeys returns a slice of keys for all routines that are currently in the
// Running state. This method is not thread-safe and should only be called while holding
// the appropriate locks.
func (c *core) unsafeRunningKeys() []string {
	running := make([]string, 0, len(c.mu.routines))
	for _, r := range c.mu.routines {
		if r.state.state == Running {
			running = append(running, r.key)
		}
	}
	return running
}

// SendUnderContext attempts to send a value v to channel ch while respecting the
// context's cancellation. It returns ctx.Err() if the context is canceled before the
// send can complete.
func SendUnderContext[V any](ctx context.Context, ch chan<- V, v V) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- v:
		return nil
	}
}

// RecvUnderContext attempts to receive a value from channel ch while respecting the
// context's cancellation. It returns ctx.Err() if the context is canceled before the
// receive can complete. If successful, it returns the received value and nil error.
func RecvUnderContext[V any](ctx context.Context, ch <-chan V) (V, error) {
	select {
	case <-ctx.Done():
		var o V
		return o, ctx.Err()
	case v := <-ch:
		return v, nil
	}
}
