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

func WithMaxRestart(maxRestart int) Option {
	return func(fo *Options) {
		fo.Signal = append(fo.Signal, signal.WithMaxRestart(maxRestart))
	}
}

func WithAddress(addr address.Address) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.WithKey(string(addr))) }
}

func CloseInletsOnExit() Option {
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
