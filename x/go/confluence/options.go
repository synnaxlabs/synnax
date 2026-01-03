// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

type Options struct {
	// Signal is a slice of signal.RoutineOptions defining the behaviour of the goroutine
	// running the segment.
	Signal []signal.RoutineOption
	// CloseOutputInletsOnExit indicates that the segment should close the inlets to the
	// output streams when the segment exits.
	CloseOutputInletsOnExit bool
}

// AttachClosables is only a valid option when CloseOutputInletsOnExit is also set.
// It attaches the passed closables to defer functions that are called when the segment
// exists to close the closable via its Close() method. The closable must also have an
// Acquire(int) method.
func (fo *Options) AttachClosables(closables ...Closable) {
	if fo.CloseOutputInletsOnExit {
		for _, inlet := range closables {
			inlet.Acquire(1)
		}
		fo.Signal = append(fo.Signal, signal.Defer(func() {
			for _, inlet := range closables {
				inlet.Close()
			}
		}, signal.WithKey("close_inlets")))
	}
}

func NewOptions(opts []Option) *Options {
	fo := &Options{}
	for _, opt := range opts {
		opt(fo)
	}
	return fo
}

type Option func(fo *Options)

// CancelOnFail cancels the context associated with the segment when it fails.
func CancelOnFail() Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.CancelOnFail()) }
}

// RecoverWithErrOnPanic recovers the goroutine run in the segment and makes it fail
// instead of panicking.
func RecoverWithErrOnPanic() Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.RecoverWithErrOnPanic())
	}
}

// RecoverWithoutErrOnPanic recovers the goroutine run in the segment and makes it exit
// instead of panicking.
func RecoverWithoutErrOnPanic() Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.RecoverWithoutErrOnPanic())
	}
}

// WithRetryOnPanic attempts to recover the segment if it panics and restarts it.
// If an argument is passed into it, it retries for the specified amount of time and
// exits with an error if it panics on its last attempt.
// If at any retry the goroutine exits with or without error, the goroutine exits and
// no longer attempts to restart.
func WithRetryOnPanic(maxRetries ...int) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithRetryOnPanic(maxRetries...))
	}
}

// WithAddress adds a key to the goroutine subtending the segment.
func WithAddress(addr address.Address) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.WithKey(string(addr))) }
}

// CloseOutputInletsOnExit closes the output stream attached to the confluence segment
// when the segment exits.
func CloseOutputInletsOnExit() Option {
	return func(fo *Options) { fo.CloseOutputInletsOnExit = true }
}

// WithClosables is only meaningful when CloseOutputInletsOnExit is also set. It
// defers a variadic list of closables (objects implementing Close() and Acquire())
// methods to be called Close() when the segment exits. It is commonly used to close
// resources after the segment finishes.
func WithClosables(closables ...Closable) Option {
	return func(fo *Options) { fo.AttachClosables(closables...) }
}

// Defer adds a function to be executed when the segment exits (fail or done). Deferred
// functions run in LIFO order.
func Defer(fn func(), opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.Defer(fn, opts...)) }
}

// DeferErr attaches the provided function f to the segment to run after exit like
// in Defer.
// Unlike Defer, if the function returns a non-nil error, the segment will fail.
func DeferErr(fn func() error, opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.DeferErr(fn, opts...)) }
}
