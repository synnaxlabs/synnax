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
	"runtime/debug"
	"runtime/pprof"
	"slices"
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
	// closeErr is written by the goroutine of closer, and read only after it returns.
	closeErr error
	closed   bool
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
// runs and closers joined, with a closer panic as an error. No goroutine that Go or
// Stop started runs after Stop returns, unless ctx ends first.
//
// ctx bounds the whole of Stop. When ctx ends first, Stop ends every lifetime that is
// left, calls no more closers, and returns at once. The run or closer it waited on
// keeps running until it returns. The error then wraps ctx.Err(), names that run or
// closer, and counts the older lifetimes left open. Call Stop once.
func (s *Stack) Stop(ctx context.Context) error {
	var errs []error
	for i, l := range slices.Backward(s.layers) {
		l.cancel()
		if !wait(ctx, l.done) {
			return s.abort(ctx, errs, i, "return")
		}
		if l.err != nil {
			errs = append(errs, errors.Wrapf(l.err, "%s", l.name))
		}
		if l.closer == nil {
			continue
		}
		closed := make(chan struct{})
		go func() {
			defer close(closed)
			// %v, not a wrap, so a context.Canceled panic is not a clean stop.
			defer func() {
				if r := recover(); r != nil {
					l.closeErr = errors.Newf("panic: %v\n%s", r, debug.Stack())
				}
			}()
			l.closeErr = l.closer()
		}()
		if !wait(ctx, closed) {
			return s.abort(ctx, errs, i, "close")
		}
		if l.closeErr != nil {
			errs = append(errs, errors.Wrapf(l.closeErr, "%s close", l.name))
		} else {
			l.closed = true
		}
	}
	return errors.Join(errs...)
}

// Closed reports whether Stop called the closer of the lifetime named name and the
// closer returned nil. Call Closed after Stop returns.
func (s *Stack) Closed(name string) bool {
	for _, l := range s.layers {
		if l.name == name {
			return l.closed
		}
	}
	return false
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

// abort ends the lifetimes older than the one at i, whose run or closer did not step
// before ctx ended. It adds the errors of the older runs that returned before the end.
func (s *Stack) abort(ctx context.Context, errs []error, i int, step string) error {
	older := s.layers[:i]
	for _, l := range older {
		select {
		case <-l.done:
			if l.err != nil {
				errs = append(errs, errors.Wrapf(l.err, "%s", l.name))
			}
		default:
		}
	}
	for _, l := range older {
		l.cancel()
	}
	noun := "lifetimes"
	if len(older) == 1 {
		noun = "lifetime"
	}
	errs = append(errs, errors.Wrapf(
		ctx.Err(),
		"%s did not %s, %d older %s left open",
		s.layers[i].name,
		step,
		len(older),
		noun,
	))
	return errors.Join(errs...)
}
