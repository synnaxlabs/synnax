// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package confluence implements a generic, template based component framework for
// building concurrent value passing programs. Confluence is built around two main
// types: Value and Segment.
//
// A Value can is any piece of data that can be passed through a go channel. Because
// channels are typically mutex locked, values should generally be exchanged as batches
// of data as opposed to individual entities (i.e. pass []int as the as opposed to int).
//
// A Segment is an entity that processes values. A Segment is typically a goroutine that
// reads values from an input channel, does some operation on them (sum, avg, IO write,
// network write, etc.), and passes a result to an output channel. This is not to say
// that the functionality of a Segment cannot extend beyond a simple transformation.
//
// For example, a Segment can route values from a set of inputs (called Outlets) to a
// set of outputs (called Inlets). The input-Outlet, outlet-Inlet naming convention
// might seem strange at first, but the general idea is that an Outlet is the end of a
// stream that emits values (i.e. <-chan Value) and an Inlet is a stream that receives
// values (i.e. chan<- Value). Inlets and Outlets are also addressable, which allows you
// to send messages to segments with different addresses based on some criteria.
//
// Collections of segments also be composed into a pipeline using the plumber package's
// plumber.Pipeline. The Pipeline type is itself a Segment that can be connected to
// other Segments. This allows for a flexible and powerful composition capabilities.
//
// The confluence package provides a number of built-in Frame that can be used by
// themselves or embedded into custom Segment(sink) that provide functionality specific
// to your use case.
//
// A Segment is a composition of three interfaces:
//
//  1. Flow - Flow.Flow method is used to start any and all operations (goroutines,
//     network pipes, etc.) used by a Segment. The context provided to Flow should be
//     used to stop operations and clear process resources.
//
//  2. Source - A Source is the part of the Segment that can send values to output
//     streams (Inlet(sink)). Inlets(sink) are bound to the Sink (and therefore Segment)
//     by calling the ApplySink.OutTo(inlets ...Inlet[ValueType]) method.
//
//  3. Sink - A Sink is the part of the Segment that can receive values.
//     Input streams (Outlet(sink)). Outlet(sink) are bound to the Sink (and therefore Segment)
//     by calling the Sink.InFrom(outlets ...Outlet[ValueType]) method.
//
// All of this flexibility comes at the cost of needing to follow a few important rules
// and principles when writing programs based on confluence:
//
//  1. All input streams (Outlets) must be bound to a Segment by using
//     the InFrom() method.
//
//  2. All output streams (Inlets) must be bound to a Segment by using
//     the OutTo() method.
//
//  3. The only way to start a Segment is by calling the Flow.Flow method. A single
//     instance of a Segment (if passed as a pointer) should generally only be running
//     once at a time. This is not to say that Segments shouldn't be restarted. This
//     rule is more of a guideline, and can be broken when you know what you're doing.
//
// Related packages:
//
//  1. freightfluence - implements transport for writing Segment(sink) that
//     interact with a network freighter.
//
//  2. plumber - components for connecting Segment(sink) together and routing
//     streams between them.
package confluence

import (
	"context"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

// Value represents an item that can be sent through a Stream.
type Value = any

// Segment is a type that reads a value from a set of Inlet(sink), performs some operation,
// transformation, etc.and writes a value to a set of Outlet(sink). The user of a Segment
// should be unaware of what occurs internally, and should only pass values through the
// Inlet and Outlet interfaces.
type Segment[I, O Value] interface {
	Sink[I]
	Source[O]
}

// Flow is an entity that starts goroutines that process a stream of values.
// All processing goroutines should be started when Flow is called, and should stop
// when the provided context is cancelled. Any options defined or provided should
// not affect the algorithmic structure of the Flow, and should instead modify
// runtime context or behavior.
type Flow interface {
	// Flow starts the Flow process under the provided signal.Context.
	Flow(ctx signal.Context, opts ...Option)
}

// Sink is an interface that accepts values from a set of Outlet(sink). The user of a Sink
// should be unaware of what occurs internally, and should only pass values through
// the Outlet interfaces.
type Sink[O Value] interface {
	InFrom(outlets ...Outlet[O])
	Flow
}

// Source is an interface that sends values to a set of Inlet(sink). The user of a Source
// should be unaware of what occurs internally, and should only pass values through // the Inlet interfaces.
type Source[I Value] interface {
	OutTo(inlets ...Inlet[I])
	Flow
}

// TransformFunc is a function that transforms a value from one type to
// another. It takes in the input value i and converts it to an output value O. If
// the returned error is not nil, the transform goroutine will exit with the error.
// If shouldSend is false, the output value will not be sent, and the transform will
// move on to processing the next value.
type TransformFunc[I, O Value] func(ctx context.Context, i I) (o O, shouldSend bool, err error)

// GeneratorFunc returns a function that generates a new value. If shouldSend is false,
// the returned function will not be used and the generator will proceed to the next cycle.
// If the returned error is not nil, the generator goroutine will exit with the error.
type GeneratorFunc[I, O Value] func(ctx context.Context, i I) (gen func() O, shouldSend bool, err error)

// Inlet is the end of a Stream that accepts values and can be addressed.
type Inlet[V Value] interface {
	// Inlet pipes a value through the Stream.
	Inlet() chan<- V
	// InletAddress returns the address of the Inlet.
	InletAddress() address.Address
	// SetInletAddress sets the OutletAddress of the Inlet.
	SetInletAddress(address.Address)
	Closable
}

type Closable interface {
	// Close closes the Closable.
	Close()
	// Acquire acquires the Closable.
	Acquire(n int32)
}

// Outlet is the end of a Stream that emits values and can be addressed.
type Outlet[V Value] interface {
	// Outlet receives a value from the Stream.
	Outlet() <-chan V
	// OutletAddress returns the address of the Outlet.
	OutletAddress() address.Address
	// SetOutletAddress sets the OutletAddress of the Outlet.
	SetOutletAddress(address.Address)
}

// NopFlow implements Flow and does nothing.
type NopFlow struct{}

// Flow implements Flow.
func (NopFlow) Flow(signal.Context, ...Option) {}

// Drain drains the provided Outlet.
func Drain[V Value](out Outlet[V]) {
	for range out.Outlet() {
	}
}
