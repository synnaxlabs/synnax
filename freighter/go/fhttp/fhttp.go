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
	xmap "github.com/synnaxlabs/x/map"
)

var defaultContentTypes = []string{"application/json", "application/msgpack"}

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}

type serverOptions struct {
	reqDecoders xmap.Map[string, binary.Decoder]
	resEncoders xmap.Map[string, binary.Encoder]
}

type ServerOption func(*serverOptions)

func WithRequestDecoders(decoders map[string]binary.Decoder) ServerOption {
	return func(o *serverOptions) { o.reqDecoders = decoders }
}

func WithResponseEncoders(encoders map[string]binary.Encoder) ServerOption {
	return func(o *serverOptions) { o.resEncoders = encoders }
}

func WithRequestAndResponseCodecs(codecs map[string]binary.Codec) ServerOption {
	return func(o *serverOptions) {
		for contentType, codec := range codecs {
			o.reqDecoders[contentType] = codec
			o.resEncoders[contentType] = codec
		}
	}
}

func newServerOptions(opts []ServerOption) serverOptions {
	so := serverOptions{
		reqDecoders: xmap.Map[string, binary.Decoder]{
			"application/json":    &binary.JSONCodec{},
			"application/msgpack": &binary.MsgPackCodec{},
		},
		resEncoders: xmap.Map[string, binary.Encoder]{
			"application/json":    &binary.JSONCodec{},
			"application/msgpack": &binary.MsgPackCodec{},
		},
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}
