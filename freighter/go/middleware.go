package freighter

import (
	"context"

	"github.com/synnaxlabs/x/middleware"
)

type (
	// Middleware is an interface that can be implemented to intercept and modify requests
	// either to a server (client side) or from a client (server side).
	Middleware = middleware.Middleware[MD, MD]
	// Finalizer is the final middleware in a chain, and is expected to execute the request.
	Finalizer = middleware.Finalizer[MD, MD]
	// Next is a function that is called to continue the middleware chain.
	Next = func(context.Context, MD) (MD, error)
	// ContextKey is a type that can be used to store and retrieve freighter specific
	// values from a context.
	ContextKey string
)

// MDContextKey is the context key used to store freighter metadata.
const MDContextKey ContextKey = "freighter.md"

func setMDOnContext(ctx context.Context, md MD) context.Context {
	return context.WithValue(ctx, MDContextKey, md)
}

// MDFromContext returns the freighter metadata from the given context.
func MDFromContext(ctx context.Context) MD { return ctx.Value(MDContextKey).(MD) }

// MiddlewareCollector is a chain of middleware that can be executed sequentially.
// It extends the middleware.Chain type to embed request metadata as a context value.
type MiddlewareCollector struct{ middleware.Chain[MD, MD] }

// Exec maintains the middleware.Chain interface.
func (mc *MiddlewareCollector) Exec(ctx context.Context, md MD, finalizer middleware.Finalizer[MD, MD]) (MD, error) {
	return mc.Chain.Exec(ctx, md, FinalizerFunc(func(ctx context.Context, md MD) (MD, error) {
		return finalizer.Finalize(setMDOnContext(ctx, md), md)
	}))
}

// Use maintains the middleware.Collector interface.
func (mc *MiddlewareCollector) Use(m ...Middleware) { mc.Chain = append(mc.Chain, m...) }

// MiddlewareFunc is a utility type so that functions can implement Middleware.
type MiddlewareFunc func(context.Context, MD, Next) (MD, error)

var _ Middleware = MiddlewareFunc(nil)

// Exec implements Middleware.
func (m MiddlewareFunc) Exec(ctx context.Context, req MD, next Next) (MD, error) {
	return m(ctx, req, next)
}

// FinalizerFunc is a utility type so that functions can implement Finalizer.
type FinalizerFunc func(context.Context, MD) (MD, error)

// Finalize implements Finalizer.
func (f FinalizerFunc) Finalize(ctx context.Context, req MD) (MD, error) {
	return f(ctx, req)
}

var NopFinalizer = FinalizerFunc(func(_ context.Context, md MD) (MD, error) { return md, nil })
