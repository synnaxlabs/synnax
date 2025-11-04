// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iter

import (
	"context"
	"io"
)

// Endlessly returns a Nexter implementation that iterates over a collection of values
// indefinitely. It's safe to ignore the error and ok values returned by next.
func Endlessly[T any](values []T) Nexter[T] {
	i := 0
	return NexterFunc[T](func(_ context.Context) (T, bool) {
		val := values[i]
		if i < (len(values) - 1) {
			i++
		} else {
			i = 0
		}
		return val, true
	})
}

// All returns a Nexter implementation that iterates over a collection of values once.
// It's safe to ignore the error value returned by next. It's also safe to leave Nexter
// unclosed.
func All[T any](values []T) Nexter[T] {
	i := 0
	return NexterFunc[T](func(_ context.Context) (t T, ok bool) {
		if i < len(values) {
			val := values[i]
			i++
			return val, true
		}
		return t, false
	})
}

// Nexter is a general purpose iterable interface the caller to traverse a collection
// of values in sequence.
type Nexter[V any] interface {
	// Next returns the next value in the sequence. If there are no more values to return
	// ok will be false. Nexter has no built-in mechanism for internal errors during
	// iteration. If you need this, see NexterCloser.
	Next(ctx context.Context) (value V, ok bool)
}

// NexterFunc wraps a function so that it can be used as a Nexter.
type NexterFunc[V any] func(context.Context) (V, bool)

var _ Nexter[any] = NexterFunc[any](nil)

// Next implements Nexter.
func (f NexterFunc[V]) Next(ctx context.Context) (V, bool) { return f(ctx) }

// NexterCloser is a general purpose iterable interface that allows the caller to
// traverse a collection of values in sequence and close the iterator when finished.
// NexterCloser is useful for iterators that need to clean up resources and/or return
// errors encountered during iteration.
type NexterCloser[V any] interface {
	Nexter[V]
	io.Closer
}

type nexterNopCloser[V any] struct{ Wrap Nexter[V] }

// NexterNopCloser allows a Nexter to implement NexterCloser by ignoring the Close
// method.
func NexterNopCloser[V any](wrap Nexter[V]) NexterCloser[V] {
	return nexterNopCloser[V]{Wrap: wrap}
}

var _ NexterCloser[any] = nexterNopCloser[any]{}

// Next implements Nexter.
func (n nexterNopCloser[V]) Next(ctx context.Context) (V, bool) { return n.Wrap.Next(ctx) }

// Close implements NexterCloser.
func (n nexterNopCloser[V]) Close() error { return nil }

// NexterCloserTranslator wraps the Next method of a NexterCloser with the defined
// Translate function.
type NexterCloserTranslator[I, O any] struct {
	Wrap      NexterCloser[I]
	Translate func(I) O
}

// Next implements Nexter.
func (n NexterCloserTranslator[I, O]) Next(ctx context.Context) (v O, ok bool) {
	rv, ok := n.Wrap.Next(ctx)
	if !ok {
		return v, ok
	}
	return n.Translate(rv), ok
}

// Close implements NexterCloser.
func (n NexterCloserTranslator[I, O]) Close() error { return n.Wrap.Close() }

// NexterTranslator wraps the Next method of a Nexter with the defined Translate function.
type NexterTranslator[I, O any] struct {
	// Wrap is the Nexter to wrap.
	Wrap Nexter[I]
	// Translate is the function to apply to the value returned by Wrap.Next.
	Translate func(I) O
}

// Next implements Nexter.
func (n NexterTranslator[I, O]) Next(ctx context.Context) (tv O, ok bool) {
	val, ok := n.Wrap.Next(ctx)
	if !ok {
		return tv, ok
	}
	return n.Translate(val), ok
}

func MapToSlice[I, O any](ctx context.Context, n Nexter[I], f func(I) O) []O {
	var values []O
	for v, ok := n.Next(ctx); ok; v, ok = n.Next(ctx) {
		values = append(values, f(v))
	}
	return values
}

// MapToSliceWithFilter iterates over a Nexter, applying a transformation function
// that both maps and filters values. The function f returns the transformed value
// and a boolean indicating whether to include it in the result slice.
// Only values where f returns true as the second return value are included.
func MapToSliceWithFilter[I, O any](ctx context.Context, n Nexter[I], f func(I) (O, bool)) []O {
	var values []O
	for v, ok := n.Next(ctx); ok; v, ok = n.Next(ctx) {
		val, ok := f(v)
		if ok {
			values = append(values, val)
		}
	}
	return values
}
