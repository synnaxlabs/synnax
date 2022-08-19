// Package store implements a simple copy-on-read in memory store. It also
// provides various wrappers to extend functionality:
//
//	Observable - allows the caller to observe changes to the store.
package store

import (
	"github.com/arya-analytics/x/observe"
	"io"
	"sync"
)

// State is the contents of a Store.
type State = any

// Reader is a readable Store.
type Reader[S State] interface {
	// CopyState returns a copy of the current state.
	CopyState() S
	// ReadState returns a read-only view of the current state.
	// Modifications to the returned state may cause undefined behavior.
	ReadState() S
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
// control to the Store (i.e. only alter the state through StorageKey.SetState calls).
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

// ReadState implements Store.
func (c *core[S]) ReadState() S {
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
	Store[S]
	observe.Observer[S]
	mu sync.Mutex
}

func ObservableWrap[S State](store Store[S]) Observable[S] {
	return &observable[S]{Store: store, Observer: observe.New[S]()}
}

// SetState implements StorageKey.CopyState.
func (o *observable[S]) SetState(state S) {
	o.Store.SetState(state)
	o.Observer.Notify(state)
}

// |||||| FLUSHABLE ||||||

type Flushable[S State] interface {
	Store[S]
	io.WriterTo
	io.ReaderFrom
}
