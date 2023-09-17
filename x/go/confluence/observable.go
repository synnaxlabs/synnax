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
	"github.com/synnaxlabs/x/signal"
)

type Subscriber[V Value] struct {
	AbstractUnarySource[V]
	observe.Observable[V]
}

func (s *Subscriber[V]) Flow(ctx signal.Context, opts ...Option) {
	ctx.Go(func(ctx context.Context) error {
		s.Observable.OnChange(func(ctx context.Context, v V) {
			signal.SendUnderContext(ctx, s.Out.Inlet(), v)
		})
		<-ctx.Done()
		return ctx.Err()
	})
}

type TransformSubscriber[V Value, T Value] struct {
	AbstractUnarySource[T]
	Transform TransformFunc[V, T]
	observe.Observable[V]
}

func (ts *TransformSubscriber[V, T]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(ts.Out)
	ctx.Go(func(ctx context.Context) error {
		remove := ts.Observable.OnChange(func(ctx context.Context, v V) {
			t, ok, _ := ts.Transform(ctx, v)
			if ok {
				_ = signal.SendUnderContext(ctx, ts.Out.Inlet(), t)
			}
		})
		<-ctx.Done()
		remove()
		return ctx.Err()
	}, o.Signal...)
}

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
	f TransformFunc[V, T],
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

type GeneratorTransformObservable[V Value, T Value] struct {
	UnarySink[V]
	GeneratorTransform GeneratorTransformFunc[V, T]
	observe.Observer[T]
}

func NewGeneratorTransformObservable[V Value, T Value](
	f GeneratorTransformFunc[V, T],
) *GeneratorTransformObservable[V, T] {
	o := &GeneratorTransformObservable[V, T]{
		GeneratorTransform: f,
		Observer:           observe.New[T](),
	}
	o.UnarySink.Sink = o.sink
	return o
}

func (o *GeneratorTransformObservable[V, T]) sink(ctx context.Context, v V) error {
	t, ok, err := o.GeneratorTransform(ctx, v)
	if err != nil || !ok {
		return err
	}
	o.Observer.NotifyGenerator(ctx, t)
	return nil
}
