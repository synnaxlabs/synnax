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
	BindTo(app *fiber.App)
}

var streamReporter = freighter.Reporter{
	Protocol:  "websocket",
	Encodings: httputil.SupportedContentTypes(),
}

var unaryReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: httputil.SupportedContentTypes(),
}

type serverOptions struct {
	codecResolver httputil.CodecResolver
}

type ServerOption func(*serverOptions)

func WithCodecResolver(r httputil.CodecResolver) ServerOption {
	return func(o *serverOptions) {
		o.codecResolver = r
	}
}

func newServerOptions(opts []ServerOption) (so serverOptions) {
	so.codecResolver = httputil.ResolveCodec
	for _, opt := range opts {
		opt(&so)
	}
	return so
}
