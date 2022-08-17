package lock

import "sync"

// Locker is an interface representing a lock. It extends the sync.Locker interface
// by allowing the caller to try to acquire the lock.
type Locker interface {
	sync.Locker
	// TryLock tries to acquire the lock. Returns true if the lock was acquired.
	TryLock() bool
}
