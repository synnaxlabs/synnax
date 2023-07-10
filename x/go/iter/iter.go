// Copyright 2023 Synnax Labs, Inc.
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
	return NexterFunc[T](func(_ context.Context) (T, bool, error) {
		val := values[i]
		if i < (len(values) - 1) {
			i++
		} else {
			i = 0
		}
		return val, true, nil
	})
}

// All returns a Nexter implementation that iterates over a collection of values once.
// It's safe to ignore the error value returned by next. It's also safe to leave Nexter
// unclosed.
func All[T any](values []T) Nexter[T] {
	i := 0
	return NexterFunc[T](func(_ context.Context) (t T, ok bool, err error) {
		if i < len(values) {
			val := values[i]
			i++
			return val, true, nil
		}
		return t, false, nil
	})
}

// Nexter is a general purpose iterable interface the caller to traverse a collection
// of values in sequence.
type Nexter[V any] interface {
	// Next returns the next value in the sequence. If there are no more values to return
	// ok will be false. If an error occurs while iterating over the values, err will be
	// non-nil.
	Next(ctx context.Context) (value V, ok bool, err error)
}

type NexterFunc[V any] func(context.Context) (V, bool, error)

var _ NexterFunc[any] = NexterFunc[any](nil)

// Next implements Nexter.
func (f NexterFunc[V]) Next(ctx context.Context) (V, bool, error) { return f(ctx) }

type NexterCloser[V any] interface {
	Nexter[V]
	io.Closer
}

type NexterNopCloser[V any] struct{ Wrap Nexter[V] }

var _ NexterCloser[any] = NexterNopCloser[any]{}

func (n NexterNopCloser[V]) Next(ctx context.Context) (V, bool, error) { return n.Wrap.Next(ctx) }

func (n NexterNopCloser[V]) Close() error { return nil }

type NexterCloserTranslator[I, O any] struct {
	Wrap      NexterCloser[I]
	Translate func(I) O
}

func (n NexterCloserTranslator[I, O]) Next(ctx context.Context) (O, bool, error) {
	val, ok, err := n.Wrap.Next(ctx)
	return n.Translate(val), ok, err
}

type NexterTranslator[I, O any] struct {
	Wrap      Nexter[I]
	Translate func(I) O
}

func (n NexterTranslator[I, O]) Next(ctx context.Context) (tv O, ok bool, err error) {
	val, ok, err := n.Wrap.Next(ctx)
	if !ok || err != nil {
		return tv, ok, err
	}
	return n.Translate(val), ok, err
}

func (n NexterCloserTranslator[I, O]) Close() error { return n.Wrap.Close() }
