package alamos

import "strings"

type options struct {
	// filters stores the LevelFilters that are used to filterTest out alamos entities.
	filters []LevelFilter
}

type Option func(*options)

func newOptions(opts ...Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func (o *options) String() string { return o.Report().String() }

func (o *options) Report() Report {
	r := make(Report)
	filters := make([]string, len(o.filters))
	for i, filter := range o.filters {
		filters[i] = filter.String()
	}
	r["filters"] = strings.Join(filters, ",")
	return r
}

func WithFilters(filters ...LevelFilter) Option {
	return func(o *options) { o.filters = append(o.filters, filters...) }
}

func defaultOptions() *options {
	return &options{
		filters: []LevelFilter{LevelFilterAll{}},
	}
}
