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
	"os"
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
// bytes. When every open handle is in use, the open waits for one to become idle. A
// held handle stays open outside the limit. At most width file and directory syncs run
// at once. A Budget is safe for concurrent use.
type Budget struct {
	limit int
	exec  Executor
	slots chan struct{}
	mu    sync.Mutex
	cond  sync.Cond
	// open counts the handles that are open, opening, or closing, apart from held ones.
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

// CreateHeld creates the file name in fs, which must not exist, and returns its handle,
// which stays open outside the limit of b until Close. The first Sync of the handle
// also syncs the directory of name.
func (b *Budget) CreateHeld(fs xfs.FS, name string) (*Handle, error) {
	h := &Handle{
		b: b, fs: fs, name: name, flag: createFlag, newEntry: true, held: true,
	}
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

// Sync makes every write to each handle of hs that returned before the call durable,
// and the directory entry of a created file, with one job per handle on the executor.
// It returns once every sync returned, with the errors of the failed syncs in the order
// of hs. A sync that fails, from any caller, marks its handle failed for good.
// errors.Is matches a failed sync of a directory with ErrDirSync.
func (b *Budget) Sync(hs []*Handle) error {
	return b.jobs(len(hs), func(i int) error { return hs[i].sync() })
}

// SyncFiles makes each file of names in fs durable, then each directory of dirs, with
// one job per name on the executor. A job opens, syncs, and closes its file in a slot
// of the pool, outside the handle limit. It returns once every job returned, with the
// errors of the failed jobs. errors.Is matches a failed sync of a directory with
// ErrDirSync.
func (b *Budget) SyncFiles(fs xfs.FS, names, dirs []string) error {
	if err := b.jobs(len(names), func(i int) error {
		return b.do(func() error {
			f, err := fs.Open(names[i], os.O_RDWR)
			if err != nil {
				return err
			}
			return errors.Join(f.Sync(), f.Close())
		})
	}); err != nil {
		return err
	}
	return b.jobs(len(dirs), func(i int) error {
		if err := b.do(func() error { return fs.SyncDir(dirs[i]) }); err != nil {
			return errors.Mark(
				errors.Wrapf(err, "sync of directory %q", dirs[i]),
				ErrDirSync,
			)
		}
		return nil
	})
}

// jobs runs f for each index below n, one job each on the executor, and returns once
// every job returned, with their errors in the order of the indexes.
func (b *Budget) jobs(n int, f func(i int) error) error {
	var wg sync.WaitGroup
	errs := make([]error, n)
	wg.Add(n)
	for i := range n {
		b.exec(func() {
			defer wg.Done()
			errs[i] = f(i)
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

// evict syncs v when it has unsynced bytes, then closes it. The sync runs in a slot of
// the pool on the caller's goroutine, so a job of the executor never waits for another
// job. The caller holds b.mu, which evict releases while the file syncs and closes. A
// failed sync or close marks v failed.
func (b *Budget) evict(v *Handle) {
	v.unlink()
	v.busy = true
	file, written, dirty := v.file, v.written, v.failed == nil && v.written != v.synced
	b.mu.Unlock()
	var err error
	if dirty {
		err = b.do(file.Sync)
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
