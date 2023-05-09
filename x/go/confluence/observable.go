// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"context"
	"github.com/synnaxlabs/x/observe"
)

// Observable is a Sink that allows callers to subscribe to values passed to it.
type Observable[V Value] struct {
	UnarySink[V]
	observe.Observer[V]
}

func (o *Observable[V]) sink(ctx context.Context, v V) error {
	o.Observer.Notify(ctx, v)
	return nil
}

// NewObservable creates a new Observable Segment.
func NewObservable[V Value]() *Observable[V] {
	o := &Observable[V]{Observer: observe.New[V]()}
	o.UnarySink.Sink = o.sink
	return o
}

type TransformObservable[V Value, T Value] struct {
	UnarySink[V]
	Transform TransformFunc[V, T]
	observe.Observer[T]
}

func NewTransformObservable[V Value, T Value](
	f func(ctx context.Context, v V) (T, bool, error),
) *TransformObservable[V, T] {
	o := &TransformObservable[V, T]{
		Transform: f,
		Observer:  observe.New[T](),
	}
	o.UnarySink.Sink = o.sink
	return o
}

func (o *TransformObservable[V, T]) sink(ctx context.Context, v V) error {
	t, ok, err := o.Transform(ctx, v)
	if err != nil || !ok {
		return err
	}
	o.Observer.Notify(ctx, t)
	return nil
}
