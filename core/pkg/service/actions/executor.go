// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions

import (
	"context"
	"sync"

	"github.com/synnaxlabs/x/gorp"
)

// Executor runs action dispatches serialized per entry key, each in its own
// transaction. Two concurrent dispatches on the same entry can never interleave their
// read-reduce-write cycles, so neither can overwrite the other's committed actions.
// Notifications always follow the commit they describe.
type Executor[K comparable, A any] struct {
	db         *gorp.DB
	dispatcher Dispatcher[K, A]
	// mu guards locks.
	mu    sync.Mutex
	locks map[K]*keyLock
}

// keyLock serializes the dispatches for one key. refs counts holders and waiters so
// the entry can be dropped once idle.
type keyLock struct {
	mu   sync.Mutex
	refs int
}

// NewExecutor constructs an Executor that opens transactions against db and notifies
// through d.
func NewExecutor[K comparable, A any](
	db *gorp.DB,
	d Dispatcher[K, A],
) *Executor[K, A] {
	return &Executor[K, A]{db: db, dispatcher: d, locks: make(map[K]*keyLock)}
}

// Dispatch runs stage inside the key's lock and its own transaction, then notifies the
// actions after the commit. When stage or the commit fails, nothing persists and no
// notification is emitted.
func (e *Executor[K, A]) Dispatch(
	ctx context.Context,
	key K,
	dispatchKey string,
	actions []A,
	stage func(gorp.Tx) error,
) error {
	return e.Serialize(key, func() error {
		if err := e.db.WithTx(ctx, stage); err != nil {
			return err
		}
		e.dispatcher.Notify(ctx, key, dispatchKey, actions)
		return nil
	})
}

// Serialize runs fn while holding the key's dispatch lock: calls for the same key run
// one at a time, calls for different keys run concurrently. Use it for dispatch flows
// too rich for Dispatch; fn owns its transaction and notification order.
func (e *Executor[K, A]) Serialize(key K, fn func() error) error {
	l := e.acquire(key)
	l.mu.Lock()
	defer e.release(key, l)
	return fn()
}

func (e *Executor[K, A]) acquire(key K) *keyLock {
	e.mu.Lock()
	defer e.mu.Unlock()
	l, ok := e.locks[key]
	if !ok {
		l = &keyLock{}
		e.locks[key] = l
	}
	l.refs++
	return l
}

func (e *Executor[K, A]) release(key K, l *keyLock) {
	l.mu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	l.refs--
	if l.refs == 0 {
		delete(e.locks, key)
	}
}
