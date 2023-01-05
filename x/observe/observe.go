// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package observe

// Observable is an interface that represents an entity whose state can be observed.
type Observable[T any] interface {
	// OnChange is called when the state of the observable changes.
	OnChange(handler func(T))
}

// Observer is an interface that can notify subscribers of changes to an observable.
type Observer[T any] interface {
	Observable[T]
	// Notify notifies all subscribers of the Value.
	Notify(T)
	// GoNotify starts a goroutine to notify all subscribers of the Value.
	GoNotify(T)
}

type base[T any] struct {
	handlers []func(T)
}

// New creates a new observer with the given options.
func New[T any]() Observer[T] {
	return &base[T]{}
}

// OnChange implements the Observable interface.
func (b *base[T]) OnChange(handler func(T)) {
	b.handlers = append(b.handlers, handler)
}

// Notify implements the Observer interface.
func (b *base[T]) Notify(v T) {
	for _, handler := range b.handlers {
		handler(v)
	}
}

// GoNotify implements the Observer interface.
func (b *base[T]) GoNotify(v T) { go b.Notify(v) }
