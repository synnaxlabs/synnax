package middleware

import "context"

// Middleware is a general entity that can be executed in a middleware chain.
// RQ represents the outgoing request type, RS represents the incoming response type.
type Middleware[RQ, RS any] interface {
	// Handle is the function that is executed by the middleware chain.
	// It receives a context and a request. It also receives a next function that
	// can be used to execute the next middleware in the chain.
	Handle(
		ctx context.Context,
		req RQ,
		next func() (RS, error),
	) (RS, error)
}

// ExecSequentially executes the given middleware chain sequentially against the given
// request.
func ExecSequentially[RQ, RS any](
	ctx context.Context,
	req RQ,
	middleware []Middleware[RQ, RS],
) (RS, error) {
	return (&Executor[RQ, RS]{Middleware: middleware}).Exec(ctx, req)
}

// Executor executes a chain of middleware.
type Executor[RQ, RS any] struct {
	// Middleware is the chain of middleware that will be executed. IMPORTANT: The
	// last middleware in the chain is not allowed to call the next function, or
	// the Executor will panic.
	Middleware []Middleware[RQ, RS]
}

// Exec executes the middleware chain sequentially given a request. The first
// middleware will be the first to receive the request, and the last to receive the response.
func (m *Executor[RQ, RS]) Exec(ctx context.Context, req RQ) (RS, error) {
	var (
		c    = 0
		mw   = make([]Middleware[RQ, RS], len(m.Middleware))
		next func() (RS, error)
	)
	copy(mw, m.Middleware)
	next = func() (RS, error) {
		if c == len(mw) {
			panic("[middleware] -  last middleware should not call next")
		}
		_m := mw[c]
		c++
		return _m.Handle(ctx, req, next)
	}
	return next()
}
