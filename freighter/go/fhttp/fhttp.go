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
	"github.com/gofiber/fiber/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/xmap"
)

var DefaultContentTypes = []string{"application/json", "application/msgpack"}

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}

type serverOptions struct {
	reqDecoders xmap.Map[string, func() binary.Decoder]
	resEncoders xmap.Map[string, func() binary.Encoder]
}

type ServerOption func(*serverOptions)

func WithRequestDecoders(decoders map[string]func() binary.Decoder) ServerOption {
	return func(o *serverOptions) { o.reqDecoders = decoders }
}

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
		reqDecoders: xmap.Map[string, func() binary.Decoder]{
			"application/json":    func() binary.Decoder { return binary.JSONCodec },
			"application/msgpack": func() binary.Decoder { return binary.MsgPackCodec },
		},
		resEncoders: xmap.Map[string, func() binary.Encoder]{
			"application/json":    func() binary.Encoder { return binary.JSONCodec },
			"application/msgpack": func() binary.Encoder { return binary.MsgPackCodec },
		},
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}
