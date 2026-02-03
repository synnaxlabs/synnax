// Copyright 2026 Synnax Labs, Inc.
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
	"go/types"
	"sync"
)

// Disconnect is a function that can be called to disconnect a handler from an
// observable.
type Disconnect = func()

// Observable is an interface that represents an entity whose state can be observed.
type Observable[T any] interface {
	// OnChange is called when the state of the observable changes. The returned
	// function can be used to disconnect the handler, stopping it from being called
	// when the observable changes.
	OnChange(handler func(context.Context, T)) Disconnect
}

// Observer is an interface that can notify subscribers of changes to an observable.
type Observer[T any] interface {
	Observable[T]
	// Notify notifies all subscribers of the Value.
	Notify(context.Context, T)
	// GoNotify starts a goroutine to notify all subscribers of the Value.
	GoNotify(context.Context, T)
	// NotifyGenerator calls the given generator function for each handler bound
	// to the observer.
	NotifyGenerator(context.Context, func() T)
}

type base[T any] struct {
	handlers map[*func(context.Context, T)]struct{}
	mu       sync.RWMutex
}

// New creates a new observer with the given options.
func New[T any]() Observer[T] { return &base[T]{} }

// OnChange implements the Observable interface.
func (b *base[T]) OnChange(handler func(context.Context, T)) Disconnect {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := &handler
	if b.handlers == nil {
		b.handlers = make(map[*func(context.Context, T)]struct{})
	}
	b.handlers[p] = struct{}{}
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.handlers, p)
	}
}

// Notify implements the Observer interface.
func (b *base[T]) Notify(ctx context.Context, v T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for h := range b.handlers {
		(*h)(ctx, v)
	}
}

// NotifyGenerator implements the Observer interface.
func (b *base[T]) NotifyGenerator(ctx context.Context, generator func() T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for h := range b.handlers {
		(*h)(ctx, generator())
	}
}

// GoNotify implements the Observer interface.
func (b *base[T]) GoNotify(ctx context.Context, v T) { go b.Notify(ctx, v) }

// Noop is an observable that never calls its OnChange function and does
// not store any handlers. Use this when you want to implement the Observable
// interface and do nothing.
type Noop[T any] struct{}

var _ Observable[any] = Noop[any]{}

// OnChange implements Observable.
func (Noop[T]) OnChange(_ func(context.Context, T)) Disconnect { return func() {} }

// Translator wraps an Observable and transforms its emitted values using the Translate
// function. Translate returns the transformed value and a boolean indicating whether
// to notify the handler. If the boolean is false, the handler is not called.
type Translator[I any, O any] struct {
	Observable[I]
	Translate func(context.Context, I) (O, bool)
}

var _ Observable[types.Nil] = Translator[any, types.Nil]{}

func (t Translator[I, O]) OnChange(handler func(context.Context, O)) Disconnect {
	return t.Observable.OnChange(func(ctx context.Context, v I) {
		result, ok := t.Translate(ctx, v)
		if !ok {
			return
		}
		handler(ctx, result)
	})
}
