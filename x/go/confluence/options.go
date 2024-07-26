// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"time"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

type Options struct {
	Signal            []signal.RoutineOption
	CloseInletsOnExit bool
}

func (fo *Options) AttachClosables(closables ...Closable) {
	if fo.CloseInletsOnExit {
		for _, inlet := range closables {
			inlet.Acquire(1)
		}
		fo.Signal = append(fo.Signal, signal.Defer(func() {
			for _, inlet := range closables {
				inlet.Close()
			}
		}, signal.WithKey("close-inlets")))
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

// WithBaseRetryInterval sets the base interval for the breaker used to restart the
// segment. The base retry interval is how much time the breaker waits before trying
// to restart for the first time. (Default: 1 second)
func WithBaseRetryInterval(retryInterval time.Duration) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithBaseRetryInterval(retryInterval))
	}
}

// WithRetryScale sets the scale on the breaker used to restart the scale. The scale
// defines the rate by which the interval between two retries grow. (Default: 1)
func WithRetryScale(scale float32) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithRetryScale(scale))
	}
}

func WithAddress(addr address.Address) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.WithKey(string(addr))) }
}

// CloseOutputInletsOnExit closes the output stream attached to the confluence segment
// when the segment exits.
func CloseOutputInletsOnExit() Option {
	return func(fo *Options) { fo.CloseInletsOnExit = true }
}

func WithClosables(closables ...Closable) Option {
	return func(fo *Options) { fo.AttachClosables(closables...) }
}

// Defer adds a function to be executed when the segment exits (fail or done).
func Defer(fn func(), opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.Defer(fn, opts...)) }
}

func DeferErr(fn func() error, opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.DeferErr(fn, opts...)) }
}
