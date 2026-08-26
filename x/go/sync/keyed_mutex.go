// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sync provides synchronization primitives that extend the standard library's
// sync package.
package sync

import "sync"

// KeyedMutex provides a mutex per key, created on first use and discarded once no
// caller holds or waits for it. The zero value is ready to use. A KeyedMutex must not
// be copied after first use.
type KeyedMutex[K comparable] struct {
	// mu guards locks.
	mu    sync.Mutex
	locks map[K]*keyLock
}

// keyLock serializes the callers for one key. refs counts holders and waiters so the
// entry can be dropped once idle.
type keyLock struct {
	mu   sync.Mutex
	refs int
}

// Do runs fn while holding the mutex for key and returns fn's error: calls for the
// same key run one at a time, calls for different keys run concurrently.
func (k *KeyedMutex[K]) Do(key K, fn func() error) error {
	l := k.acquire(key)
	l.mu.Lock()
	defer k.release(key, l)
	return fn()
}

func (k *KeyedMutex[K]) acquire(key K) *keyLock {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.locks == nil {
		k.locks = make(map[K]*keyLock)
	}
	l, ok := k.locks[key]
	if !ok {
		l = &keyLock{}
		k.locks[key] = l
	}
	l.refs++
	return l
}

func (k *KeyedMutex[K]) release(key K, l *keyLock) {
	l.mu.Unlock()
	k.mu.Lock()
	defer k.mu.Unlock()
	l.refs--
	if l.refs == 0 {
		delete(k.locks, key)
	}
}
