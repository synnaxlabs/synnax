package confluence

import (
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
)

type Options struct {
	Signal            []signal.RoutineOption
	CloseInletsOnExit bool
}

func (fo *Options) AttachInletCloser(closer InletCloser) {
	if fo.CloseInletsOnExit {
		fo.Signal = append(fo.Signal, signal.Defer(closer.CloseInlets))
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

func WithInletCloser(closer InletCloser) Option {
	return func(fo *Options) { fo.AttachInletCloser(closer) }
}

func Defer(f func()) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.Defer(f)) }
}

func CancelOnExit() Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.CancelOnExit()) }
}

func CancelOnExitErr() Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.CancelOnExitErr()) }
}

func WithAddress(addr address.Address) Option {
	return func(fo *Options) { fo.Signal = append(fo.Signal, signal.WithKey(string(addr))) }
}

func CloseInletsOnExit() Option {
	return func(fo *Options) { fo.CloseInletsOnExit = true }
}
