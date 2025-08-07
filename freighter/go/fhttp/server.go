// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"maps"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/binary"
)

var (
	defaultEncoders = map[string]func() binary.Encoder{}
	defaultDecoders = map[string]func() binary.Decoder{}
	defaultCodecs   = map[string]binary.Codec{
		MIMEApplicationJSON:    binary.JSONCodec,
		MIMEApplicationMsgPack: binary.MsgPackCodec,
	}
)

type serverOptions struct {
	reqDecoders map[string]func() binary.Decoder
	resEncoders map[string]func() binary.Encoder
}

type ServerOption func(*serverOptions)

func WithResponseEncoders(encoders map[string]func() binary.Encoder) ServerOption {
	return func(o *serverOptions) { o.resEncoders = encoders }
}

func WithAdditionalCodecs(codecs map[string]func() binary.Codec) ServerOption {
	return func(o *serverOptions) {
		for contentType, getCodec := range codecs {
			o.reqDecoders[contentType] = func() binary.Decoder { return getCodec() }
			o.resEncoders[contentType] = func() binary.Encoder { return getCodec() }
		}
	}
}

func newServerOptions(opts []ServerOption) serverOptions {
	so := serverOptions{
		reqDecoders: maps.Clone(defaultDecoders),
		resEncoders: maps.Clone(defaultEncoders),
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

func getReporter(offers []string) freighter.Reporter {
	return freighter.Reporter{
		Protocol:  "http",
		Encodings: offers,
	}
}

func init() {
	for contentType, codec := range defaultCodecs {
		defaultDecoders[contentType] = func() binary.Decoder { return codec }
		defaultEncoders[contentType] = func() binary.Encoder { return codec }
	}
}
