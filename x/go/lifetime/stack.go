// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package lifetime runs the lifetimes of components nested in the order they start,
// and stops them in the reverse order.
package lifetime

import (
	"context"
	"runtime/pprof"
	"slices"
	"strings"
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// Stack runs functions that each block for the lifetime of a component. A caller nests
// the lifetimes as one context tree: it passes each Go the context that the previous Go
// returned. Stop ends the lifetimes from the newest to the oldest.
//
// Call Go and Stop from one goroutine. Failed is safe for concurrent use.
type Stack struct {
	layers []*layer
	failed chan struct{}
	fail   sync.Once
}

type layer struct {
	name   string
	cancel context.CancelFunc
	done   chan struct{}
	err    error
	closer func() error
	// closed is nil until Stop calls closer, and closes when closer returns.
	closed   chan struct{}
	closeErr error
}

// NewStack returns an empty Stack.
func NewStack() *Stack { return &Stack{failed: make(chan struct{})} }

// Go starts run in a goroutine with a child of ctx, and returns that child for the
// next Go. It labels the goroutine with name for profiles and tracebacks. The child
// ends when Stop reaches run, or when ctx ends. Stop calls closer after run returns and
// before it ends the next older lifetime. closer may be nil.
//
// A run that returns before Stop does not end its context, so the runs inside it keep
// running. When it returns an error, Failed closes. Go must not be called after Stop.
func (s *Stack) Go(
	ctx context.Context,
	name string,
	run func(ctx context.Context) error,
	closer func() error,
) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	l := &layer{name: name, cancel: cancel, done: make(chan struct{}), closer: closer}
	s.layers = append(s.layers, l)
	go pprof.Do(ctx, pprof.Labels("routine", name), func(ctx context.Context) {
		defer close(l.done)
		if l.err = run(ctx); l.err != nil {
			s.fail.Do(func() { close(s.failed) })
		}
	})
	return ctx
}

// Failed returns a channel that closes when a run returns an error. A caller selects on
// it to start shutdown, then calls Stop to get the error.
func (s *Stack) Failed() <-chan struct{} { return s.failed }

// Stop ends the newest lifetime, waits for its run to return, and waits for its closer
// to return. It then does the same for each older one. It returns the errors of the
// runs and closers joined. No goroutine that Go or Stop started runs after Stop
// returns, unless ctx ends first.
//
// ctx bounds the whole of Stop. When ctx ends first, Stop ends every lifetime that is
// left, calls no more closers, and returns at once. A run that has not returned keeps
// running until it sees its context end, and a closer that has not returned keeps
// running in its own goroutine until it returns. The error then wraps ctx.Err() and
// names those runs and closers. Call Stop once.
func (s *Stack) Stop(ctx context.Context) error {
	for _, l := range slices.Backward(s.layers) {
		l.cancel()
		if !wait(ctx, l.done) {
			return s.abort(ctx)
		}
		if l.closer == nil {
			continue
		}
		l.closed = make(chan struct{})
		go func() {
			defer close(l.closed)
			l.closeErr = l.closer()
		}()
		if !wait(ctx, l.closed) {
			return s.abort(ctx)
		}
	}
	return s.result(ctx)
}

// wait returns true when done closes before ctx ends. A done that is already closed
// wins over a ctx that already ended.
func wait(ctx context.Context, done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
	}
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Stack) abort(ctx context.Context) error {
	for _, l := range s.layers {
		l.cancel()
	}
	return s.result(ctx)
}

func (s *Stack) result(ctx context.Context) error {
	var (
		errs    []error
		running []string
		closing []string
	)
	for _, l := range s.layers {
		select {
		case <-l.done:
			if l.err != nil {
				errs = append(errs, errors.Wrapf(l.err, "%s", l.name))
			}
		default:
			running = append(running, l.name)
			continue
		}
		if l.closed == nil {
			continue
		}
		select {
		case <-l.closed:
			if l.closeErr != nil {
				errs = append(errs, errors.Wrapf(l.closeErr, "%s close", l.name))
			}
		default:
			closing = append(closing, l.name)
		}
	}
	if len(running) > 0 {
		errs = append(errs, errors.Wrapf(
			ctx.Err(),
			"%s did not return",
			strings.Join(running, ", "),
		))
	}
	if len(closing) > 0 {
		errs = append(errs, errors.Wrapf(
			ctx.Err(),
			"%s did not close",
			strings.Join(closing, ", "),
		))
	}
	return errors.Join(errs...)
}
