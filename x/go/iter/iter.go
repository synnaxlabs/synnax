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

// Endlessly returns a Next implementation that iterates over a collection of values
// indefinitely. It's safe to ignore the error and ok values returned by next.
func Endlessly[T any](values []T) Next[T] {
	i := 0
	return NextFunc[T](func(_ context.Context) (T, bool, error) {
		val := values[i]
		if i < (len(values) - 1) {
			i++
		} else {
			i = 0
		}
		return val, true, nil
	})
}

// All returns a Next implementation that iterates over a collection of values once.
// It's safe to ignore the error value returned by next. It's also safe to leave Next
// unclosed.
func All[T any](values []T) Next[T] {
	i := 0
	return NextFunc[T](func(_ context.Context) (t T, ok bool, err error) {
		if i < len(values) {
			val := values[i]
			i++
			return val, true, nil
		}
		return t, false, nil
	})
}

// Next is a general purpose iterable interface the caller to traverse a collection
// of values in sequence.
type Next[V any] interface {
	// Next returns the next value in the sequence. If there are no more values to return
	// ok will be false. If an error occurs while iterating over the values, err will be
	// non-nil.
	Next(ctx context.Context) (value V, ok bool, err error)
}

type NextFunc[V any] func(context.Context) (V, bool, error)

var _ NextFunc[any] = NextFunc[any](nil)

// Next implements NextFunc.
func (f NextFunc[V]) Next(ctx context.Context) (V, bool, error) { return f(ctx) }

type NextCloser[V any] interface {
	Next[V]
	io.Closer
}

type NopNextCloser[V any] struct{ Wrap Next[V] }

var _ NextCloser[any] = NopNextCloser[any]{}

func (n NopNextCloser[V]) Next(ctx context.Context) (V, bool, error) { return n.Wrap.Next(ctx) }

func (n NopNextCloser[V]) Close() error { return nil }

type NextCloserTranslator[I, O any] struct {
	Wrap      NextCloser[I]
	Translate func(I) O
}

func (n NextCloserTranslator[I, O]) Next(ctx context.Context) (O, bool, error) {
	val, ok, err := n.Wrap.Next(ctx)
	return n.Translate(val), ok, err
}

type NextTranslator[I, O any] struct {
	Wrap      Next[I]
	Translate func(I) O
}

func (n NextTranslator[I, O]) Next(ctx context.Context) (O, bool, error) {
	val, ok, err := n.Wrap.Next(ctx)
	return n.Translate(val), ok, err
}

func (n NextCloserTranslator[I, O]) Close() error { return n.Wrap.Close() }
