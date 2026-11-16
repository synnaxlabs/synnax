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

// Stack runs functions that each block for the lifetime of a component. Each function
// gets a context that is a child of the context of the function started before it, so
// the lifetimes nest as one context tree. Stop ends them from the newest to the oldest.
//
// Call Go and Stop from one goroutine. Failed is safe for concurrent use.
type Stack struct {
	// tail is the context of the newest lifetime, the parent of the next one.
	tail   context.Context
	layers []*layer
	failed chan struct{}
	fail   sync.Once
}

type layer struct {
	name   string
	cancel context.CancelFunc
	done   chan struct{}
	err    error
}

// NewStack returns a Stack whose first lifetime is a child of ctx. When ctx ends, every
// lifetime ends with it, without the order that Stop gives.
func NewStack(ctx context.Context) *Stack {
	return &Stack{tail: ctx, failed: make(chan struct{})}
}

// Go starts run in a goroutine with a context that is a child of the context of the
// previous run, and labels the goroutine with name for profiles and tracebacks. The
// context ends when Stop reaches run, or when an outer context ends.
//
// A run that returns before Stop does not end its context, so the runs inside it keep
// running. When it returns an error, Failed closes. Go must not be called after Stop.
func (s *Stack) Go(name string, run func(ctx context.Context) error) {
	ctx, cancel := context.WithCancel(s.tail)
	l := &layer{name: name, cancel: cancel, done: make(chan struct{})}
	s.tail = ctx
	s.layers = append(s.layers, l)
	go pprof.Do(ctx, pprof.Labels("routine", name), func(ctx context.Context) {
		defer close(l.done)
		if l.err = run(ctx); l.err != nil {
			s.fail.Do(func() { close(s.failed) })
		}
	})
}

// Failed returns a channel that closes when a run returns an error. A caller selects on
// it to start shutdown, then calls Stop to get the error.
func (s *Stack) Failed() <-chan struct{} { return s.failed }

// Stop ends the newest lifetime and waits for its run to return, then does the same
// for each older one. It returns the errors of the runs joined. No goroutine that Go
// started runs after Stop returns, unless ctx ends first.
//
// When ctx ends first, Stop ends every lifetime that is left and returns at once. The
// runs that have not returned keep running until they see their context end. The
// error then wraps ctx.Err() and names those runs. Call Stop once.
func (s *Stack) Stop(ctx context.Context) error {
	for _, l := range slices.Backward(s.layers) {
		l.cancel()
		// A run that already returned wins over a ctx that already ended.
		select {
		case <-l.done:
			continue
		default:
		}
		select {
		case <-l.done:
		case <-ctx.Done():
			for _, l := range s.layers {
				l.cancel()
			}
			return s.result(ctx)
		}
	}
	return s.result(ctx)
}

func (s *Stack) result(ctx context.Context) error {
	var (
		errs    []error
		running []string
	)
	for _, l := range s.layers {
		select {
		case <-l.done:
			if l.err != nil {
				errs = append(errs, errors.Wrapf(l.err, "%s", l.name))
			}
		default:
			running = append(running, l.name)
		}
	}
	if len(running) > 0 {
		errs = append(errs, errors.Wrapf(
			ctx.Err(),
			"%s did not return",
			strings.Join(running, ", "),
		))
	}
	return errors.Join(errs...)
}
