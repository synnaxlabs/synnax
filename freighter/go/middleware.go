package freighter

import (
	"context"
	"github.com/synnaxlabs/x/middleware"
)

type (
	// Middleware is an interface that can be implemented to intercept and modify requests
	// either to a server (client side) or from a client (server side).
	Middleware = middleware.Middleware[MD]
	// Finalizer is the final middleware in a chain, and is expected to execute the request.
	Finalizer = middleware.Finalizer[MD]
	// Next is a function that is called to continue the middleware chain.
	Next = func(context.Context, MD) error
	// MiddlewareChain is a chain of middleware that can be executed sequentially.
	MiddlewareChain = middleware.Chain[MD]
	// MiddlewareCollector implements the Transport.Use method and collects middleware
	// for execution in a MiddlewareChain.
	MiddlewareCollector = middleware.Collector[MD]
)

// MiddlewareFunc is a utility type so that functions can implement Middleware.
type MiddlewareFunc func(context.Context, MD, Next) error

// Handle implements Middleware.
func (m MiddlewareFunc) Handle(ctx context.Context, req MD, next Next) error {
	return m(ctx, req, next)
}

// FinalizerFunc is a utility type so that functions can implement Finalizer.
type FinalizerFunc func(context.Context, MD) error

// Finalize implements Finalizer.
func (f FinalizerFunc) Finalize(ctx context.Context, req MD) error {
	return f(ctx, req)
}
