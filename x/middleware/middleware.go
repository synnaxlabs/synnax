package middleware

import "context"

// Middleware is an entity that be executed in a middleware chain. V represents the value
// type pass through the chain.
type Middleware[V any] interface {
	// Exec executes the middleware. It receives a context and a request. It also receives
	// a next function that can be used to execute the next middleware in the chain.
	Exec(ctx context.Context, val V, next func(context.Context, V) error) error
}

// Finalizer is the final middleware in a chain, and is expected to execute the request.
type Finalizer[V any] interface {
	// Finalize is a special middleware that is executed after the middleware chain
	// has completed. It receives the context and the request that was sent.
	Finalize(ctx context.Context, value V) error
}

// Chain is a chain of middleware that can be executed sequentially.
type Chain[V any] []Middleware[V]

// Exec executes the middleware chain sequentially given a value. The first
// middleware will be the first to receive the incoming, and the last to receive the outgoing value.
func (c Chain[V]) Exec(ctx context.Context, val V, finalizer Finalizer[V]) error {
	var (
		i    = 0
		next func(context.Context, V) error
	)
	next = func(_ctx context.Context, _val V) error {
		if _ctx.Err() != nil {
			return _ctx.Err()
		}
		if i == len(c) {
			return finalizer.Finalize(_ctx, _val)
		}
		_m := c[i]
		i++
		return _m.Exec(_ctx, _val, next)
	}
	return next(ctx, val)
}

// Collector allows middleware to be collected and executed in a chain.
type Collector[RQ any] struct{ Chain[RQ] }

// Use adds middleware to the collector.
func (p *Collector[RQ]) Use(m ...Middleware[RQ]) { p.Chain = append(p.Chain, m...) }
