package observe

type options struct {
	async bool
}

func newOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

type Option func(options *options)
