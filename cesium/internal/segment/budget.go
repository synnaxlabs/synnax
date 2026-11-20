// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"sync"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

// Executor runs f, at once or on another goroutine. It must run every f it gets.
type Executor func(f func())

// GoExecutor runs each f on a new goroutine.
func GoExecutor(f func()) { go f() }

// Budget limits the open file handles of a database, and runs every sync of their
// files in one pool. To open a handle when the limit is reached, it closes the idle
// handle that was used least recently, and syncs that handle first when it has unsynced
// bytes. When every open handle is in use, the open waits for one to become idle. At
// most width file and directory syncs run at once. A Budget is safe for concurrent use.
type Budget struct {
	limit int
	exec  Executor
	slots chan struct{}
	mu    sync.Mutex
	cond  sync.Cond
	// open counts the handles that are open, opening, or closing.
	open int
	// idle is the sentinel of a ring of the open handles that no call uses, the most
	// recently used first.
	idle Handle
}

// NewBudget returns a budget of limit open handles and width syncs at once, both at
// least 1, that runs each sync on exec.
func NewBudget(limit, width int, exec Executor) (*Budget, error) {
	v := validate.New("segment.Budget")
	v.Positive("limit", limit)
	v.Positive("width", width)
	v.NotNil("executor", exec)
	if err := v.Error(); err != nil {
		return nil, err
	}
	b := &Budget{limit: limit, exec: exec, slots: make(chan struct{}, width)}
	b.cond.L = &b.mu
	b.idle.prev, b.idle.next = &b.idle, &b.idle
	return b, nil
}

// Create creates the file name in fs, which must not exist, and returns its handle. The
// first Sync of the handle also syncs the directory of name.
func (b *Budget) Create(fs xfs.FS, name string) (*Handle, error) {
	h := &Handle{b: b, fs: fs, name: name, flag: createFlag, newEntry: true}
	if _, err := h.acquire(); err != nil {
		return nil, err
	}
	h.release(false)
	return h, nil
}

// Open returns the handle of the file name in fs, which exists. The file opens on the
// first call that uses it.
func (b *Budget) Open(fs xfs.FS, name string) *Handle {
	return &Handle{b: b, fs: fs, name: name, flag: openFlag}
}

// Sync syncs each handle of hs on the executor, and returns once every sync returned.
// It returns the errors of the failed syncs, in the order of hs.
func (b *Budget) Sync(hs []*Handle) error {
	var wg sync.WaitGroup
	errs := make([]error, len(hs))
	wg.Add(len(hs))
	for i, h := range hs {
		b.exec(func() {
			defer wg.Done()
			errs[i] = h.Sync()
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

// do runs f in a slot of the pool, on the calling goroutine.
func (b *Budget) do(f func() error) error {
	b.slots <- struct{}{}
	defer func() { <-b.slots }()
	return f()
}

// run runs f in a slot of the pool, on the executor, and returns its error.
func (b *Budget) run(f func() error) error {
	done := make(chan error, 1)
	b.exec(func() { done <- b.do(f) })
	return <-done
}

// evict syncs v when it has unsynced bytes, then closes it. The caller holds b.mu,
// which evict releases while the file syncs and closes. A failed sync or close marks v
// failed.
func (b *Budget) evict(v *Handle) {
	v.unlink()
	v.busy = true
	file, written, dirty := v.file, v.written, v.failed == nil && v.written != v.synced
	b.mu.Unlock()
	var err error
	if dirty {
		err = b.run(file.Sync)
	}
	err = errors.Join(err, file.Close())
	b.mu.Lock()
	v.file, v.busy = nil, false
	b.open--
	if err != nil {
		v.fail(err)
	} else if dirty {
		v.synced = written
	}
	b.cond.Broadcast()
}
