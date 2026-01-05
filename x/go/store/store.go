// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package store implements a simple copy-on-read in memory store. It also
// provides various wrappers to extend functionality:
//
//	Observable - allows the caller to observe changes to the store.
package store

import (
	"context"
	"io"
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Reader is a readable Store.
type Reader[S any] interface {
	// CopyState returns a copy of the current state.
	CopyState() S
	// PeekState returns a read-only view of the current state.
	// Modifications to the returned state may cause undefined behavior.
	PeekState() (S, func())
}

// Writer is a writable Store.
type Writer[S any] interface {
	// SetState sets the state of the store. This is NOT a copy-on write operation,
	// so make sure to provide a copy of the state (i.e. use Reader.CopyState when
	// reading for write).
	SetState(context.Context, S)
}

// Store is a simple copy-on-read in memory store.
// SinkTarget create a new Store, called store.New().
type Store[S any] interface {
	Reader[S]
	Writer[S]
}

type core[S any] struct {
	copy  func(S) S
	mu    sync.RWMutex
	state S
}

// New opens a new Store. copy is a function that copies the state.
// It's up to the caller to determine the depth of the copy. Store
// serves as a proxy to the state, so it's important to yield access
// control to the Store (i.e. only alter the state through Store.SetState calls).
func New[S any](_copy func(S) S) Store[S] {
	return &core[S]{copy: _copy}
}

// SetState implements Store.
func (c *core[S]) SetState(_ context.Context, state S) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// CopyState implements Store.
func (c *core[S]) CopyState() S {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.copy(c.state)
}

// PeekState implements Store.
func (c *core[S]) PeekState() (S, func()) {
	c.mu.RLock()
	return c.state, c.mu.RUnlock
}

// Observable is a wrapper around a Store that allows the caller to observe
// State changes. SinkTarget create a new store.Observable, called store.ObservableWrap().
type Observable[S, O any] interface {
	Store[S]
	observe.Observable[O]
}

type observable[S, O any] struct {
	ObservableConfig[S, O]
	Store[S]
	observe.Observer[O]
}

type ObservableConfig[S, O any] struct {
	// Store is the store to wrap.
	// [REQUIRED]
	Store Store[S]
	// Transform is a function that receives the previous and new state and returns the
	// object that should be sent to observers. If the returned boolean is false, the
	// observer will not be triggered, and no one will be notified. If this value is
	// [REQUIRED]
	Transform func(prev, next S) (O, bool)
	// GoNotify is a boolean indicating whether to notify subscribers in a goroutine.
	// [NOT REQUIRED]
	GoNotify *bool
}

var _ config.Config[ObservableConfig[any, any]] = ObservableConfig[any, any]{}

func (o ObservableConfig[S, O]) Override(
	other ObservableConfig[S, O],
) ObservableConfig[S, O] {
	o.Transform = override.Nil(o.Transform, other.Transform)
	o.GoNotify = override.Nil(o.GoNotify, other.GoNotify)
	o.Store = override.Nil(o.Store, other.Store)
	return o
}

func (o ObservableConfig[S, O]) Validate() error {
	v := validate.New("observable")
	validate.NotNil(v, "store", o.Store)
	validate.NotNil(v, "transform", o.Transform)
	return v.Error()
}

func WrapObservable[S, O any](
	cfgs ...ObservableConfig[S, O],
) (Observable[S, O], error) {
	defaultConfig := ObservableConfig[S, O]{GoNotify: config.True()}
	cfg, err := config.New(defaultConfig, cfgs...)
	return &observable[S, O]{
		ObservableConfig: cfg,
		Store:            cfg.Store,
		Observer:         observe.New[O](),
	}, err
}

// SetState implements Store.
func (o *observable[S, O]) SetState(ctx context.Context, state S) {
	prev := o.CopyState()
	notify, shouldNotify := o.Transform(prev, state)
	if shouldNotify {
		lo.Ternary(
			*o.ObservableConfig.GoNotify,
			o.Observer.GoNotify,
			o.Notify,
		)(ctx, notify)
	}
	o.Store.SetState(ctx, state)
}

type Flushable[S any] interface {
	Store[S]
	io.WriterTo
	io.ReaderFrom
}
