// Package confluence implements a generic, template based component framework for
// building concurrent value passing programs. Confluence is built around two core
// types: Value and Segment.
//
// A Value can is any piece of data that can be passed through
// a go channel. Because channels are typically mutex locked, values should generally
// be exchanged as batches of data as opposed to individual entities (i.e.
// pass []int as the as opposed to int).
//
// A Segment is an entity that processes values. A Segment is typically
// a goroutine that reads values from an input channel, does some operation on them
// (sum, avg, IO write, network write, etc.), and passes a result to an output channel.
// This is not to say that the functionality of a Segment cannot extend beyond a
// simple transformation.
//
// For example, a Segment can route values from a set of inputs (called Outlet(s)) to
// a set of outputs (called Inlet(s)). The input-Outlet, outlet-Inlet naming convention
// might seem strange at first, but the general idea is that an Outlet is the end of a
// stream that emits values (i.e. <-chan Value) and an Inlet is a stream that receives
// values (i.e. chan<- Value). Inlets and Outlets are also addressable, which allows you
// to send messages to segments with different addresses based on some criteria.
//
// Collections of Segments can also be composed into a pipeline using the plumber
// package's plumber.Pipeline. The Pipeline type is itself a Segment that can be
// connected to other Segment(s). This allows for a flexible and powerful
// composition capabilities.
//
// The confluence package provides a number of built-in Segments that can be used
// by themselves or embedded into custom Segment(s) that provide functionality specific
// to your use case.
//
// A Segment is a composition of three interfaces:
//
//  1. Flow - Flow.Flow method is used to start any and all operations (goroutines,
//     network pipes, etc.) used by a Segment. The context provided to Flow should be
//     used to stop operations and clear process resources.
//
//  2. Source - A Source is the part of the Segment that can send values to output
//     streams (Inlet(s)). Inlets(s) are bound to the Sink (and therefore Segment)
//     by calling the ApplySink.OutTo(inlets ...Inlet[ValueType]) method.
//
//  3. Sink - A Sink is the part of the Segment that can receive values.
//     Input streams (Outlet(s)). Outlet(s) are bound to the Sink (and therefore Segment)
//     by calling the Sink.InFrom(outlets ...Outlet[ValueType]) method.
//
// All of this flexibility comes at the cost of needing to follow a few important rules
// and principles when writing programs based on confluence:
//
//  1. All input streams (Outlet(s)) must be bound to a Segment by using
//     the InFrom() method.
//
//  2. All output streams (Inlet(s)) must be bound to a Segment by using
//     the OutTo() method.
//
//  3. The only way to start a Segment is by calling the Flow.Flow method. A single
//     instance of a Segment (if passed as a pointer) should generally only be running
//     once at a time. This is not to say that Segment(s) shouldn't be restarted. This
//     rule is more of a guideline, and can be broken when you know what you're doing.
//     If you're worried about this happening, check out the Gate, GateSource,
//     and GateSink functions; these implement locks that prevent the segment from
//     being started while already running).
//
// Related packages:
//
//  1. freightfluence - implements transport for writing Segment(s) that
//     interact with a network freighter.
//
//  2. plumber - components for connecting Segment(s) together and routing
//     streams between them.
package confluence

import (
	"context"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
)

// Value represents an item that can be sent through a Stream.
type Value = any

// Segment is a type that reads a value from a set of Inlet(s), performs some operation,
// transformation, etc.and writes a value to a set of Outlet(s). The user of a Segment
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

// Sink is an interface that accepts values from a set of Outlet(s). The user of a Sink
// should be unaware of what occurs internally, and should only pass values through
// the Outlet interfaces.
type Sink[O Value] interface {
	InFrom(outlets ...Outlet[O])
	Flow
}

// Source is an interface that sends values to a set of Inlet(s). The user of a Source
// should be unaware of what occurs internally, and should only pass values through // the Inlet interfaces.
type Source[I Value] interface {
	OutTo(inlets ...Inlet[I])
	Flow
	InletCloser
}

// InletCloser is a type that contains a set of Inlet(s)  that can be closed. This type
// also typically implements the Source interface.
type InletCloser interface {
	CloseInlets()
}

// TransformFunc is a template for a function  that transforms a value from one type to
// another. A TransformFunc can perform IO, Network InfectedBatch, Aggregations, or any other
// type of operation.
type TransformFunc[I, O Value] struct {
	//	Transform is the function that performs the transformation. The user of the LinearTransform
	//	should define this function before Flow is called.
	Transform func(ctx context.Context, i I) (o O, ok bool, err error)
}

// Stream represents a streamImpl of values. Each streamImpl has an addressable Outlet
// and an addressable Inlet. These addresses are best represented as unique locations where values
// are received from (Inlet) and sent to (Outlet). It is also generally OK to share a streamImpl across multiple
// Segments, as long as those segments perform are replicates of one another.
type Stream[V Value] interface {
	Inlet[V]
	Outlet[V]
}

// Inlet is the end of a Stream that accepts values and can be addressed.
type Inlet[V Value] interface {
	// Inlet pipes a value through the Stream.
	Inlet() chan<- V
	// InletAddress returns the address of the Inlet.
	InletAddress() address.Address
	// SetInletAddress sets the OutletAddress of the Inlet.
	SetInletAddress(address.Address)
	// Close closes the inlet.
	Close()
	// Acquire acquires the inlet.
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

// EmptyFlow is a Flow that does nothing.
type EmptyFlow struct{}

// Flow implements the Flow interface.
func (ef EmptyFlow) Flow(ctx signal.Context, opts ...Option) {}
