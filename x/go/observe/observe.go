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
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Disconnect is a function that can be called to disconnect a handler from an
// observable.
type Disconnect = func()

// Observable is an interface that represents an entity whose state can be observed.
type Observable[T any] interface {
	// OnChange is called when the state of the observable changes. The returned
	// function can be used to disconnect the handler, stopping it from being called
	// when the observable changes.
	OnChange(func(context.Context, T)) Disconnect
}

// Observer is an interface that can notify subscribers of changes to an observable.
type Observer[T any] interface {
	Observable[T]
	// Notify notifies all subscribers of the Value. It calls the handlers while it
	// holds a lock, so a handler must not call Notify, OnChange, or a Disconnect on the
	// same Observer: the call can deadlock.
	Notify(context.Context, T)
}

type base[T any] struct {
	handlers set.Set[*func(context.Context, T)]
	mu       sync.RWMutex
}

// New creates a new observer. A Disconnect from it blocks until every running Notify
// returns, so the handler never runs after Disconnect returns.
func New[T any]() Observer[T] {
	return &base[T]{handlers: set.New[*func(context.Context, T)]()}
}

func (b *base[T]) OnChange(handler func(context.Context, T)) Disconnect {
	b.mu.Lock()
	defer b.mu.Unlock()
	p := &handler
	b.handlers.Add(p)
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.handlers.Remove(p)
	}
}

func (b *base[T]) Notify(ctx context.Context, v T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for h := range b.handlers {
		(*h)(ctx, v)
	}
}

// Noop is an observable that never calls its OnChange function and does
// not store any handlers. Use this when you want to implement the Observable
// interface and do nothing.
type Noop[T any] struct{}

var _ Observable[any] = Noop[any]{}

func (Noop[T]) OnChange(func(context.Context, T)) Disconnect { return func() {} }

// Translator wraps an Observable and transforms its emitted values using the Translate
// function. Translate returns the transformed value and a boolean indicating whether to
// notify the handler. If the boolean is false, the handler is not called.
type Translator[I any, O any] struct {
	Observable[I]
	Translate func(context.Context, I) (O, bool)
}

var _ Observable[struct{}] = Translator[any, struct{}]{}

func (t Translator[I, O]) OnChange(handler func(context.Context, O)) Disconnect {
	return t.Observable.OnChange(func(ctx context.Context, v I) {
		result, ok := t.Translate(ctx, v)
		if !ok {
			return
		}
		handler(ctx, result)
	})
}

// Merge returns an Observable that notifies a handler on a change to any of
// observables. Disconnecting the merged handler disconnects it from every one.
func Merge[T any](observables ...Observable[T]) Observable[T] {
	return merged[T](observables)
}

type merged[T any] []Observable[T]

func (m merged[T]) OnChange(handler func(context.Context, T)) Disconnect {
	disconnects := make([]Disconnect, len(m))
	for i, o := range m {
		disconnects[i] = o.OnChange(handler)
	}
	return func() {
		for _, disconnect := range disconnects {
			disconnect()
		}
	}
}
