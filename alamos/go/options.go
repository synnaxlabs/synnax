// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

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

func WithFilters(filters ...LevelFilter) Option {
	return func(o *options) { o.filters = append(o.filters, filters...) }
}

func defaultOptions() *options {
	return &options{
		filters: []LevelFilter{LevelFilterAll{}},
	}
}
