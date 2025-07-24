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
	"github.com/synnaxlabs/x/httputil"
)

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}

type serverOptions struct {
	reqCodecResolver              httputil.CodecResolver
	resCodecResolver              httputil.CodecResolver
	supportedResponseContentTypes []string
}

type ServerOption func(*serverOptions)

func WithCodecResolver(r httputil.CodecResolver) ServerOption {
	return func(o *serverOptions) {
		o.reqCodecResolver = r
		o.resCodecResolver = r
	}
}

func WithResponseCodecResolver(r httputil.CodecResolver, supportedContentTypes []string) ServerOption {
	return func(o *serverOptions) {
		o.resCodecResolver = r
		o.supportedResponseContentTypes = supportedContentTypes
	}
}

func newServerOptions(opts []ServerOption) serverOptions {
	so := serverOptions{
		reqCodecResolver:              httputil.ResolveCodec,
		resCodecResolver:              httputil.ResolveCodec,
		supportedResponseContentTypes: httputil.SupportedContentTypes(),
	}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}
