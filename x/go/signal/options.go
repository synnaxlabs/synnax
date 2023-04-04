package signal

import "github.com/synnaxlabs/alamos"

type Option func(o *options)

type options struct {
	alamos.Instrumentation
}

func WithInstrumentation(i alamos.Instrumentation) Option {
	return func(o *options) {
		o.Instrumentation = i
	}
}

func newOptions(opts []Option) options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return *o
}
