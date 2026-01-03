// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package middleware

// Middleware is a function that can act on a request of type I, pass it on to the
// next middleware, and process a response of type O.
type Middleware[I, O any] interface {
	// Exec executes the middleware. It receives a context and a request. It also receives
	// a next function that can be used to execute the next middleware in the chain.
	Exec(in I, next func(I) (O, error)) (out O, err error)
}

// Finalizer is the final middleware in a chain, and is expected to execute the request.
type Finalizer[I, O any] interface {
	// Finalize is a special middleware that is executed after the middleware chain
	// has completed. It receives the context and the request that was sent.
	Finalize(in I) (out O, err error)
}

// Chain is a chain of middleware that can be executed sequentially.
type Chain[I, O any] []Middleware[I, O]

// Exec executes the middleware chain sequentially given a value. The first
// middleware will be the first to receive the incoming, and the last to receive the outgoing value.
func (c Chain[I, O]) Exec(in I, finalizer Finalizer[I, O]) (out O, err error) {
	var (
		i    = 0
		next func(I) (O, error)
	)
	next = func(_in I) (out O, err error) {
		if i == len(c) {
			return finalizer.Finalize(_in)
		}
		_m := c[i]
		i++
		return _m.Exec(_in, next)
	}
	return next(in)
}
