// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/override"
)

type Option func(o *options)

// WithCodec sets the encoder (and decoder) used to serialize entries. It's
// important to note that reading data encoded in a different format may cause
// undefined behavior.
func WithCodec(codec encoding.Codec) Option {
	return func(opts *options) { opts.Codec = codec }
}

// WithMaxDeleteSize sets the most entries that one Delete may delete or rewrite once
// references cascade and detach. A larger delete fails before it changes anything. The
// default is 1 << 20.
func WithMaxDeleteSize(n int) Option {
	return func(opts *options) { opts.MaxDeleteSize = n }
}

// WithInstrumentation sets the instrumentation that the DB logs to.
func WithInstrumentation(ins alamos.Instrumentation) Option {
	return func(opts *options) { opts.Instrumentation = ins }
}

// options holds the configuration applied to a DB and inherited by
// every Tx opened from it.
type options struct {
	// Codec encodes and decodes entries to and from the underlying KV.
	encoding.Codec
	// MaxDeleteSize is the most entries that one Delete may delete or rewrite.
	MaxDeleteSize int
	// Instrumentation is where the DB logs.
	Instrumentation alamos.Instrumentation
}

var defaultOptions = options{
	Codec:         encoding.NewDecodeFallbackCodec(orc.Codec, msgpack.Codec),
	MaxDeleteSize: 1 << 20,
}

func newOptions(opts []Option) options {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	return overrideOptions(o)
}

func overrideOptions(o options) options {
	o.Codec = override.Nil(defaultOptions.Codec, o.Codec)
	o.MaxDeleteSize = override.Numeric(defaultOptions.MaxDeleteSize, o.MaxDeleteSize)
	return o
}
