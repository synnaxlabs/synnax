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
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/override"
)

type Option func(o *options)

// WithCodec sets the encoder (and decoder) used to serialize entries. It's
// important to note that reading data encoded in a different format may cause
// undefined behavior.
func WithCodec(codec encoding.Codec) Option {
	return func(opts *options) { opts.Codec = codec }
}

type options struct {
	encoding.Codec
}

var defaultOptions = options{Codec: msgpack.Codec}

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
