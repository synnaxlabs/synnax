// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
