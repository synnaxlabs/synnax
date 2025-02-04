// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"golang.org/x/net/context"
	"golang.org/x/sync/errgroup"
	"strings"
	"sync"
	"time"
)

// Context is an extension of the standard context.Context that provides a way to
// signal a goroutine to maybeStop.
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
func WithTimeout(ctx context.Context, d time.Duration, opts ...Option) (Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, d)
	c := newCore(ctx, cancel)
	return c, cancel
}

func Isolated(opts ...Option) (Context, context.CancelFunc) {
	return WithCancel(context.Background(), opts...)
}

func Wrap(ctx context.Context, opts ...Option) Context {
	return newCore(ctx, func() {}, opts...)
}

func newCore(
	ctx context.Context,
	cancel context.CancelFunc,
	opts ...Option,
) *core {
	c := &core{
		options: newOptions(opts),
		Context: ctx,
		cancel:  cancel,
	}
	c.mu.stopped = make(chan struct{})
	return c
}

type core struct {
	options
	context.Context
	cancel   context.CancelFunc
	internal errgroup.Group
	mu       struct {
		sync.RWMutex
		routines []*routine
		stopped  chan struct{}
	}
}

func (c *core) routineDiagnostics() string {
	// create a strings.Builder, iterate through each piece of info, and pretty
	// print it.
	var b strings.Builder
	for _, i := range c.routines() {
		b.WriteString(i.PrettyString())
	}
	return b.String()
}

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

func (c *core) Stopped() <-chan struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mu.stopped
}

func SendUnderContext[V any](ctx context.Context, ch chan<- V, v V) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case ch <- v:
		return nil
	}
}
