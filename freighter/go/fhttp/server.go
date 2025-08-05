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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/xmap"
)

var (
	defaultEncoders = xmap.Map[string, func() binary.Encoder]{}
	defaultDecoders = xmap.Map[string, func() binary.Decoder]{}
	defaultCodecs   = xmap.Map[string, func() binary.Codec]{
		MIMEApplicationJSON:    func() binary.Codec { return binary.JSONCodec },
		MIMEApplicationMsgPack: func() binary.Codec { return binary.MsgPackCodec },
	}
)

type serverOptions struct {
	reqDecoders xmap.Map[string, func() binary.Decoder]
	resEncoders xmap.Map[string, func() binary.Encoder]
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
		reqDecoders: defaultDecoders.Copy(),
		resEncoders: defaultEncoders.Copy(),
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}

func getReporter(so serverOptions) freighter.Reporter {
	encodings := set.New(so.reqDecoders.Keys()...)
	for contentType := range so.resEncoders {
		encodings.Add(contentType)
	}
	return freighter.Reporter{
		Protocol:  "http",
		Encodings: encodings.Elements(),
	}
}

func init() {
	for contentType, getCodec := range defaultCodecs {
		defaultDecoders[contentType] = func() binary.Decoder { return getCodec() }
		defaultEncoders[contentType] = func() binary.Encoder { return getCodec() }
	}
}
