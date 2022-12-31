// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lock

import (
	"sync"
)

// Keys allows a caller to acquire a lock on a key.
type Keys[K comparable] struct {
	mu    *sync.Mutex
	locks map[K]struct{}
}

// NewKeys returns a new Keys.
func NewKeys[K comparable]() Keys[K] {
	return Keys[K]{locks: make(map[K]struct{}), mu: &sync.Mutex{}}
}

// TryLock attempts to acquire locks on the given keys. Returns false is any
// key is already locked. Otherwise, returns true.
func (m Keys[K]) TryLock(keys ...K) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		_, ok := m.locks[key]
		if ok {
			return false
		}
	}
	for _, key := range keys {
		m.locks[key] = struct{}{}
	}
	return true
}

// TryLockWithReleaser has identical semantics to TryLock, but returns a
// Releaser that can be used to release the locks on the given keys.
func (m Keys[K]) TryLockWithReleaser(keys ...K) (bool, Releaser) {
	if m.TryLock(keys...) {
		return true, ReleaserFunc(func() { m.Unlock(keys...) })
	}
	return false, nil
}

// Unlock releases the locks on the given keys. Panics if any key is not locked.
func (m Keys[K]) Unlock(keys ...K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		_, ok := m.locks[key]
		if !ok {
			panic("[lock.map] - attempted to unlock and unlocked key")
		}
		delete(m.locks, key)
	}
}
