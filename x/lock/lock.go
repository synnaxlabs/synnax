package lock

import (
	"github.com/cockroachdb/errors"
	"sync"
)

var ErrLocked = errors.New("locked")

// Locker is an interface representing a lock. It extends the sync.Locker interface
// by allowing the caller to try to acquire the lock.
type Locker interface {
	sync.Locker
	// TryLock tries to acquire the lock. Returns true if the lock was acquired.
	TryLock() bool
}

type Releaser interface {
	// Release releases the lock. Panics if the lock is not held.
	Release()
}

type ReleaserFunc func()

func (f ReleaserFunc) Release() { f() }
