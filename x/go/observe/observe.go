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

	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
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
	// Notify notifies all subscribers of the Value.
	Notify(context.Context, T)
	// GoNotify starts a goroutine to notify all subscribers of the Value.
	GoNotify(context.Context, T)
	// NotifyGenerator calls the given generator function for each handler bound to the
	// observer.
	NotifyGenerator(context.Context, func() T)
}

type base[T any] struct {
	handlers set.Set[*func(context.Context, T)]
	mu       sync.RWMutex
}

// New creates a new observer.
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

func (b *base[T]) NotifyGenerator(ctx context.Context, generator func() T) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for h := range b.handlers {
		(*h)(ctx, generator())
	}
}

func (b *base[T]) GoNotify(ctx context.Context, v T) {
	go b.Notify(context.WithoutCancel(ctx), v)
}

type asyncMessage[T any] struct {
	ctx context.Context
	val T
}

type asyncHandler[T any] struct {
	ch     chan asyncMessage[T]
	done   chan struct{}
	closed sync.Once
}

type async[T any] struct {
	mu       sync.RWMutex
	ctx      context.Context
	g        signal.Go
	opts     []signal.RoutineOption
	handlers map[*asyncHandler[T]]func(context.Context, T)
}

// NewAsync creates an Observer where each registered handler receives notifications on
// a dedicated goroutine via a buffered channel. If a handler falls behind,
// notifications are dropped rather than blocking the caller. Goroutines are managed by
// the provided signal.Go, which handles panic recovery, lifecycle tracking, and
// context-driven shutdown.
func NewAsync[T any](ctx signal.Context, opts ...signal.RoutineOption) Observer[T] {
	return &async[T]{
		ctx:      ctx,
		g:        ctx,
		opts:     opts,
		handlers: make(map[*asyncHandler[T]]func(context.Context, T)),
	}
}

func (a *async[T]) OnChange(handler func(context.Context, T)) Disconnect {
	a.mu.Lock()
	defer a.mu.Unlock()
	h := &asyncHandler[T]{
		ch:   make(chan asyncMessage[T], 64),
		done: make(chan struct{}),
	}
	var doneOnce sync.Once
	a.g.Go(func(ctx context.Context) error {
		defer doneOnce.Do(func() { close(h.done) })
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case msg, ok := <-h.ch:
				if !ok {
					return nil
				}
				handler(msg.ctx, msg.val)
			}
		}
	}, a.opts...)
	a.handlers[h] = handler
	return func() {
		a.mu.Lock()
		delete(a.handlers, h)
		a.mu.Unlock()
		h.closed.Do(func() {
			close(h.ch)
			select {
			case <-h.done:
			case <-a.ctx.Done():
			}
		})
	}
}

func (a *async[T]) Notify(ctx context.Context, v T) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	msg := asyncMessage[T]{ctx: ctx, val: v}
	for h := range a.handlers {
		select {
		case h.ch <- msg:
		default:
		}
	}
}

func (a *async[T]) NotifyGenerator(ctx context.Context, generator func() T) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for h := range a.handlers {
		msg := asyncMessage[T]{ctx: ctx, val: generator()}
		select {
		case h.ch <- msg:
		default:
		}
	}
}

func (a *async[T]) GoNotify(ctx context.Context, v T) {
	go a.Notify(context.WithoutCancel(ctx), v)
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
