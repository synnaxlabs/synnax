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

func CancelOnExitErr() Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.CancelOnExitErr()) }
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
