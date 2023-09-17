// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package observe

import (
	"context"
	"sync"
)

type Disconnect = func()

// Observable is an interface that represents an entity whose state can be observed.
type Observable[T any] interface {
	// OnChange is called when the state of the observable changes.
	OnChange(handler func(context.Context, T)) (disconnect Disconnect)
}

// Observer is an interface that can notify subscribers of changes to an observable.
type Observer[T any] interface {
	Observable[T]
	// Notify notifies all subscribers of the Value.
	Notify(context.Context, T)
	// GoNotify starts a goroutine to notify all subscribers of the Value.
	GoNotify(context.Context, T)
	// NotifyGenerator calls the given generator function for each subscriber.
	NotifyGenerator(context.Context, func() T)
}

type base[T any] struct {
	mu       sync.Mutex
	handlers map[*func(context.Context, T)]struct{}
}

// New creates a new observer with the given options.
func New[T any]() Observer[T] { return &base[T]{} }

// OnChange implements the Observable interface.
func (b *base[T]) OnChange(handler func(context.Context, T)) Disconnect {
	b.mu.Lock()
	p := &handler
	if b.handlers == nil {
		b.handlers = make(map[*func(context.Context, T)]struct{})
	}
	b.handlers[p] = struct{}{}
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		delete(b.handlers, p)
		b.mu.Unlock()
	}
}

// Notify implements the Observer interface.
func (b *base[T]) Notify(ctx context.Context, v T) {
	for handler := range b.handlers {
		(*handler)(ctx, v)
	}
}

// NotifyGenerator implements the Observer interface.
func (b *base[T]) NotifyGenerator(ctx context.Context, generator func() T) {
	for handler := range b.handlers {
		(*handler)(ctx, generator())
	}
}

// GoNotify implements the Observer interface.
func (b *base[T]) GoNotify(ctx context.Context, v T) { go b.Notify(ctx, v) }

// Noop is an observable that never calls it's OnChange function and does
// not store any handlers. Use this when you want to implement the Observable
// interface and do nothing.
type Noop[T any] struct{}

var _ Observable[any] = Noop[any]{}

// OnChange implements Observable.
func (Noop[T]) OnChange(_ func(context.Context, T)) Disconnect { return func() {} }

type Translator[I any, O any] struct {
	Observable[I]
	Translate func(I) O
}

func (t Translator[I, O]) OnChange(handler func(context.Context, O)) Disconnect {
	return t.Observable.OnChange(func(ctx context.Context, v I) { handler(ctx, t.Translate(v)) })
}
