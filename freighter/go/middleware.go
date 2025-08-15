// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter

import (
	"context"

	"github.com/synnaxlabs/x/middleware"
)

type (
	// Middleware is an interface that can be implemented to intercept and modify
	// requests either to a server (client-side) or from a client (server-side).
	Middleware = middleware.Middleware[Context, Context]
	// MiddlewareHandler is a function that acts on a given Context and returns a new
	// Context and an error.
	MiddlewareHandler = middleware.Handler[Context, Context]
	// MiddlewareCollector can be used to collect a chain of middleware and execute
	// them in sequence.
	MiddlewareCollector = middleware.Collector[Context, Context]
	// MiddlewareFunc is a helper type that transforms a function into a Middleware.
	MiddlewareFunc = middleware.Func[Context, Context]
)

// ExtractContext returns the Context from the given context.
func ExtractContext(ctx context.Context) Context { return ctx.(Context) }

// UseOnAll uses the given middleware on all the given transports.
func UseOnAll(middleware []Middleware, transports ...Transport) {
	for _, t := range transports {
		t.Use(middleware...)
	}
}

// NoopMiddlewareHandler is a middleware handler that does nothing.
var NoopMiddlewareHandler = MiddlewareHandler(func(ctx Context) (Context, error) {
	return ctx, nil
})
