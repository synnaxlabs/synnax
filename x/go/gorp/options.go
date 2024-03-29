// Copyright 2023 Synnax Labs, Inc.
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

// WithEncoderDecoder sets the encoder (and decoder) used to serialize entries. It's
// important to note that reading data encoded in a different format may cause
// undefined behavior.
func WithEncoderDecoder(ecd binary.EncoderDecoder) Option {
	return func(opts *options) { opts.EncoderDecoder = ecd }
}

type options struct {
	binary.EncoderDecoder
}

var _ Tools = options{}

var defaultOptions = options{EncoderDecoder: &binary.MsgPackEncoderDecoder{}}

func newOptions(opts []Option) options {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	return overrideOptions(o)
}

func overrideOptions(o options) options {
	o.EncoderDecoder = override.Nil(defaultOptions.EncoderDecoder, o.EncoderDecoder)
	return o
}
