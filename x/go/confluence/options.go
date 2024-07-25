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

func CancelOnFail() Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.CancelOnFail()) }
}

// document
func RecoverWithErrOnPanic() Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.RecoverWithErrOnPanic())
	}
}

func RecoverWithoutErrOnPanic() Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.RecoverWithoutErrOnPanic())
	}
}

func WithRetryOnPanic(maxRetries ...int) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithRetryOnPanic(maxRetries...))
	}
}

func WithBaseRetryInterval(retryInterval time.Duration) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithBaseRetryInterval(retryInterval))
	}
}

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

func Defer(fn func(), opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.Defer(fn, opts...)) }
}

func DeferErr(fn func() error, opts ...signal.RoutineOption) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.DeferErr(fn, opts...)) }
}
