// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package middleware

// Handler is a function that can process a request of type I and returns a response of
// type O or an error.
type Handler[I, O any] func(I) (O, error)

// Middleware is a function that can act on a request of type I, pass it on to the next
// middleware, and process a response of type O.
type Middleware[I, O any] interface {
	// Exec executes the middleware. It receives an input value and a function that can
	// be used to execute the next middleware in the chain.
	Exec(I, Handler[I, O]) (O, error)
}

// Chain is a chain of middleware that can be executed sequentially.
type Chain[I, O any] []Middleware[I, O]

var _ Middleware[any, any] = Chain[any, any]{}

// Exec executes the middleware chain sequentially given a value. The first middleware
// will be the first to receive the incoming, and the last to receive the outgoing
// value.
func (c Chain[I, O]) Exec(in I, finalizer Handler[I, O]) (O, error) {
	var (
		i    int
		next Handler[I, O]
	)
	next = func(in I) (O, error) {
		if i == len(c) {
			return finalizer(in)
		}
		m := c[i]
		i++
		return m.Exec(in, next)
	}
	return next(in)
}

// Collector provides a utility for constructing middleware chains.
type Collector[I, O any] struct{ chain Chain[I, O] }

var _ Middleware[any, any] = (*Collector[any, any])(nil)

// Exec executes the middleware chain with the provided input and finalizer.
func (c *Collector[I, O]) Exec(in I, finalizer Handler[I, O]) (O, error) {
	return c.chain.Exec(in, finalizer)
}

// Use adds the provided middleware to the chain.
func (c *Collector[I, O]) Use(middleware ...Middleware[I, O]) {
	c.chain = append(c.chain, middleware...)
}

// Func is an adapter that allows ordinary functions to be used as Middleware.
type Func[I, O any] func(I, Handler[I, O]) (O, error)

var _ Middleware[any, any] = (*Func[any, any])(nil)

// Exec calls f(in, next)
func (f Func[I, O]) Exec(in I, next Handler[I, O]) (O, error) {
	return f(in, next)
}
