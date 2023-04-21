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

type options struct {
	binary.EncoderDecoder
	_noPrefix bool
}

var _ Options = options{}

func (o options) noPrefix() bool { return o._noPrefix }

type Option func(o *options)

// WithEncoderDecoder sets the encoder (and decoder) used to serialize entries. It's
// important to note that reading data encoded in a different format may cause
// undefined behavior.
func WithEncoderDecoder(ecd binary.EncoderDecoder) Option {
	return func(opts *options) { opts.EncoderDecoder = ecd }
}

// WithNoPrefix disables the use of type prefixes for different entries in the
// database. This should be used with caution, as it may result in collisions.
func WithNoPrefix() Option {
	return func(opts *options) { opts._noPrefix = true }
}

func newOptions(opts ...Option) options {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}
	return overrideOptions(o)
}

func overrideOptions(o options) options {
	base := defaultOptions()
	o.EncoderDecoder = override.Nil(base.EncoderDecoder, o.EncoderDecoder)
	return o
}

func defaultOptions() options {
	ed := &binary.MsgPackEncoderDecoder{}
	return options{EncoderDecoder: ed, _noPrefix: false}
}
