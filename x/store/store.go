// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"io"
	"sync"
)

// State is the contents of a Store.
type State = any

// Reader is a readable Store.
type Reader[S State] interface {
	// CopyState returns a copy of the current state.
	CopyState() S
	// PeekState returns a read-only view of the current state.
	// Modifications to the returned state may cause undefined behavior.
	PeekState() S
}

// Writer is a writable Store.
type Writer[S State] interface {
	// SetState sets the state of the store. This is NOT a copy-on write operation,
	// so make sure to provide a copy of the state (i.e. use Reader.CopyState when
	// reading for write).
	SetState(S)
}

// Store is a simple copy-on-read in memory store.
// SinkTarget create a new Store, called store.New().
type Store[S State] interface {
	Reader[S]
	Writer[S]
}

// |||||| CORE ||||||

type core[S State] struct {
	copy  func(S) S
	mu    sync.RWMutex
	state S
}

// New opens a new Store. copy is a function that copies the state.
// It's up to the caller to determine the depth of the copy. Store
// serves as a proxy to the state, so it's important to yield access
// control to the Store (i.e. only alter the state through Store.SetState calls).
func New[S State](copy func(S) S) Store[S] {
	return &core[S]{copy: copy}
}

// SetState implements Store.
func (c *core[S]) SetState(state S) {
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
func (c *core[S]) PeekState() S {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// |||||| OBSERVABLE ||||||

// Observable is a wrapper around a Store that allows the caller to observe
// State changes. SinkTarget create a new store.Observable, called store.ObservableWrap().
type Observable[S State] interface {
	Store[S]
	observe.Observable[S]
}

type observable[S State] struct {
	ObservableConfig[S]
	Store[S]
	observe.Observer[S]
	mu sync.Mutex
}

type ObservableConfig[S State] struct {
	// ShouldNotify is a function that diffs the old and new state and returns
	// true if subscribers of the state should be notified.
	ShouldNotify func(prevState S, nextState S) bool
	// GoNotify is a boolean indicating whether to notify subscribers in a goroutine.
	GoNotify *bool
}

var _ config.Config[ObservableConfig[any]] = ObservableConfig[any]{}

func (o ObservableConfig[S]) Override(
	other ObservableConfig[S],
) ObservableConfig[S] {
	o.ShouldNotify = override.Nil(o.ShouldNotify, other.ShouldNotify)
	o.GoNotify = override.Nil(o.GoNotify, other.GoNotify)
	return o
}

func (o ObservableConfig[S]) Validate() error {
	return nil
}

func ObservableWrap[S State](store Store[S], cfgs ...ObservableConfig[S]) Observable[S] {
	cfg, _ := config.OverrideAndValidate(ObservableConfig[S]{
		GoNotify: config.BoolPointer(true),
	}, cfgs...)
	return &observable[S]{ObservableConfig: cfg, Store: store, Observer: observe.New[S]()}
}

// SetState implements Store.CopyState.
func (o *observable[S]) SetState(state S) {
	if o.ShouldNotify == nil || o.ShouldNotify(o.PeekState(), state) {
		if *o.ObservableConfig.GoNotify {
			o.Observer.GoNotify(state)
		} else {
			o.Observer.Notify(state)
		}
	}
	o.Store.SetState(state)
}

// |||||| FLUSHABLE ||||||

type Flushable[S State] interface {
	Store[S]
	io.WriterTo
	io.ReaderFrom
}
