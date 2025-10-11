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
	// Middleware is an interface that can be implemented to intercept and modify requests
	// either to a server (client side) or from a client (server side).
	Middleware = middleware.Middleware[Context, Context]
	// Finalizer is the final middleware in a chain, and is expected to execute the request.
	Finalizer = middleware.Finalizer[Context, Context]
	// Next is a function that is called to continue the middleware chain.
	Next = func(Context) (Context, error)
	// ContextKey is a type that can be used to store and retrieve freighter specific
	// values from a context.
	ContextKey string
)

// MDContextKey is the context key used to store freighter metadata.
const MDContextKey ContextKey = "freighter.md"

// MDFromContext returns the freighter metadata from the given context.
func MDFromContext(ctx context.Context) Context { return ctx.(Context) }

// MiddlewareCollector is a chain of middleware that can be executed sequentially.
// It extends the middleware.Chain type to embed request metadata as a context value.
type MiddlewareCollector struct {
	middleware.Chain[Context, Context]
}

// Exec maintains the middleware.Chain interface.
func (mc *MiddlewareCollector) Exec(fCtx Context, finalizer middleware.Finalizer[Context, Context]) (Context, error) {
	return mc.Chain.Exec(fCtx, FinalizerFunc(func(md Context) (Context, error) {
		return finalizer.Finalize(md)
	}))
}

// Use maintains the middleware.Collector interface.
func (mc *MiddlewareCollector) Use(m ...Middleware) { mc.Chain = append(mc.Chain, m...) }

// MiddlewareFunc is a utility type so that functions can implement Middleware.
type MiddlewareFunc func(Context, Next) (Context, error)

var _ Middleware = MiddlewareFunc(nil)

// Exec implements Middleware.
func (m MiddlewareFunc) Exec(fCtx Context, next Next) (Context, error) {
	return m(fCtx, next)
}

// FinalizerFunc is a utility type so that functions can implement Finalizer.
type FinalizerFunc func(Context) (Context, error)

// Finalize implements Finalizer.
func (f FinalizerFunc) Finalize(req Context) (Context, error) {
	return f(req)
}

// NopFinalizer is a Finalizer that returns the request metadata unmodified.
var NopFinalizer = FinalizerFunc(func(md Context) (Context, error) { return md, nil })

func UseOnAll(middleware []Middleware, transports ...Transport) {
	for _, t := range transports {
		t.Use(middleware...)
	}
}
