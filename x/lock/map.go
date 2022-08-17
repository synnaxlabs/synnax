package lock

import (
	"sync"
)

// KeyLock allows a caller to acquire a lock on a key.
type KeyLock[K comparable] struct {
	mu    *sync.Mutex
	locks map[K]struct{}
}

// NewLock returns a new KeyLock.
func NewLock[K comparable]() KeyLock[K] {
	return KeyLock[K]{locks: make(map[K]struct{}), mu: &sync.Mutex{}}
}

// TryLock attempts to acquire locks on the given keys. Returns false is any
// key is already locked. Otherwise, returns true.
func (m KeyLock[K]) TryLock(keys ...K) bool {
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

// Unlock releases the locks on the given keys. Panics if any key is not locked.
func (m KeyLock[K]) Unlock(keys ...K) {
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
