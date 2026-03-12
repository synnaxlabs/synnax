// Copyright 2026 Synnax Labs, Inc.
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
	"fmt"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
)

// ObservableTransformPublisher is a Source that subscribes to an ObservableSubscriber,
// transforms the value through a provided transform function, and publishes the
// transformed value to its outlets.
type ObservableTransformPublisher[V Value, T Value] struct {
	alamos.Instrumentation
	AbstractUnarySource[T]
	Transform TransformFunc[V, T]
	observe.Observable[V]
}

// Flow implements the Flow interface.
func (ts *ObservableTransformPublisher[V, T]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(ts.Out)
	// Register the handler synchronously so that any goroutines it spawns (e.g.
	// async observer handlers) exist before Flow returns. This prevents goroutine
	// leak false positives in tests that snapshot baselines between setup and spec.
	remove := ts.OnChange(func(_ context.Context, v V) {
		t, ok, err := ts.Transform(ctx, v)
		if err != nil {
			ts.L.Error(fmt.Sprintf("observable transform publisher: transform error: %s", err))
			return
		}
		if !ok {
			return
		}
		if err = signal.SendUnderContext(ctx, ts.Out.Inlet(), t); err != nil && ctx.Err() == nil {
			ts.L.Error(fmt.Sprintf("observable transform publisher: send error: %s", err))
		}
	})
	ctx.Go(func(ctx context.Context) error {
		<-ctx.Done()
		remove()
		return ctx.Err()
	}, o.Signal...)
}

// ObservableSubscriber is a Sink that allows callers to subscribe to values passed to it.
type ObservableSubscriber[V Value] struct {
	UnarySink[V]
	observe.Observer[V]
}

func (o *ObservableSubscriber[V]) sink(ctx context.Context, v V) error {
	o.Notify(ctx, v)
	return nil
}

// NewObservableSubscriber creates a new ObservableSubscriber Segment.
func NewObservableSubscriber[V Value]() *ObservableSubscriber[V] {
	o := &ObservableSubscriber[V]{Observer: observe.New[V]()}
	o.Sink = o.sink
	return o
}

type GeneratorTransformObservable[V Value, T Value] struct {
	UnarySink[V]
	Generator GeneratorFunc[V, T]
	observe.Observer[T]
}

func NewGeneratorTransformObservable[V Value, T Value](
	f GeneratorFunc[V, T],
) *GeneratorTransformObservable[V, T] {
	o := &GeneratorTransformObservable[V, T]{
		Generator: f,
		Observer:  observe.New[T](),
	}
	o.Sink = o.sink
	return o
}

func (o *GeneratorTransformObservable[V, T]) sink(ctx context.Context, v V) error {
	t, ok, err := o.Generator(ctx, v)
	if err != nil || !ok {
		return err
	}
	o.NotifyGenerator(ctx, t)
	return nil
}
