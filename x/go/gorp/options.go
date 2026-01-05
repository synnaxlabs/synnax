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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/override"
)

type Option func(o *options)

// WithCodec sets the encoder (and decoder) used to serialize entries. It's
// important to note that reading data encoded in a different format may cause
// undefined behavior.
func WithCodec(ecd binary.Codec) Option {
	return func(opts *options) { opts.Codec = ecd }
}

type options struct {
	binary.Codec
}

var _ Tools = options{}

var defaultOptions = options{Codec: &binary.MsgPackCodec{}}

func newOptions(opts []Option) options {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	return overrideOptions(o)
}

func overrideOptions(o options) options {
	o.Codec = override.Nil(defaultOptions.Codec, o.Codec)
	return o
}
