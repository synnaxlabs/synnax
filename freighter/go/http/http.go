// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/http"
)

type BindableTransport interface {
	freighter.Transport
	BindTo(*fiber.App)
}

var streamReporter = freighter.Reporter{
	Protocol:  "websocket",
	Encodings: http.SupportedContentTypes(),
}

var unaryReporter = freighter.Reporter{
	Protocol:  "http",
	Encodings: http.SupportedContentTypes(),
}

type serverOptions struct{ codecResolver http.CodecResolver }

type ServerOption func(*serverOptions)

func WithCodecResolver(r http.CodecResolver) ServerOption {
	return func(o *serverOptions) { o.codecResolver = r }
}

func newServerOptions(opts []ServerOption) serverOptions {
	so := serverOptions{codecResolver: http.ResolveCodec}
	for _, opt := range opts {
		opt(&so)
	}
	return so
}
