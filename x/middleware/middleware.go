package middleware

import "context"

// Middleware is an entity that be executed in a middleware chain. V represents the value
// type pass through the chain.
type Middleware[I, O any] interface {
	// Exec executes the middleware. It receives a context and a request. It also receives
	// a next function that can be used to execute the next middleware in the chain.
	Exec(ctx context.Context, in I, next func(context.Context, I) (O, error)) (out O, err error)
}

// Finalizer is the final middleware in a chain, and is expected to execute the request.
type Finalizer[I, O any] interface {
	// Finalize is a special middleware that is executed after the middleware chain
	// has completed. It receives the context and the request that was sent.
	Finalize(ctx context.Context, in I) (out O, err error)
}

// Chain is a chain of middleware that can be executed sequentially.
type Chain[I, O any] []Middleware[I, O]

// Exec executes the middleware chain sequentially given a value. The first
// middleware will be the first to receive the incoming, and the last to receive the outgoing value.
func (c Chain[I, O]) Exec(ctx context.Context, in I, finalizer Finalizer[I, O]) (out O, err error) {
	var (
		i    = 0
		next func(context.Context, I) (O, error)
	)
	next = func(_ctx context.Context, _in I) (out O, err error) {
		if _ctx.Err() != nil {
			return out, _ctx.Err()
		}
		if i == len(c) {
			return finalizer.Finalize(_ctx, _in)
		}
		_m := c[i]
		i++
		return _m.Exec(_ctx, _in, next)
	}
	return next(ctx, in)
}

// Collector allows middleware to be collected and executed in a chain.
type Collector[I, O any] struct{ Chain[I, O] }

// Use adds middleware to the collector.
func (p *Collector[I, O]) Use(m ...Middleware[I, O]) { p.Chain = append(p.Chain, m...) }
