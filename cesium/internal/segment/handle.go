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
	"path"
	"sync"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

var (
	// ErrClosed is the error of a call on a closed handle.
	ErrClosed = errors.New("segment handle is closed")
	// ErrDirSync is the error of a failed sync of the directory entry of a new segment.
	ErrDirSync = errors.New("directory sync failed")
)

// Budget limits the open file handles of a database. To open a handle when the limit
// is reached, it closes the idle handle that was used least recently, and syncs that
// handle first when it has unsynced bytes. When every open handle is in use, the open
// waits for one to become idle. A Budget is safe for concurrent use.
type Budget struct {
	limit int
	mu    sync.Mutex
	cond  sync.Cond
	// open counts the handles that are open, opening, or closing.
	open int
	// idle is the sentinel of a ring of the open handles that no call uses, the most
	// recently used first.
	idle Handle
}

// NewBudget returns a budget of limit open handles, at least 1.
func NewBudget(limit int) (*Budget, error) {
	if limit < 1 {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"handle budget of %d is below 1",
			limit,
		)
	}
	b := &Budget{limit: limit}
	b.cond.L = &b.mu
	b.idle.prev, b.idle.next = &b.idle, &b.idle
	return b, nil
}

// Handle is the file of one segment. Its budget opens the file on use and may close it
// between calls. A sync that fails, from any caller, marks the handle failed: it never
// syncs or writes again, and both calls return the error of that sync. A failed close
// by the budget marks it failed too. A Handle is safe for concurrent use.
type Handle struct {
	b    *Budget
	fs   xfs.FS
	name string
	// The fields below are guarded by b.mu.
	file xfs.File
	// flag opens file. It holds O_CREATE until the first open.
	flag int
	// refs counts the calls that use file.
	refs int
	// busy is true while the file opens or closes. Calls on the handle wait.
	busy bool
	// written counts writes, and synced is written at the start of the last sync
	// that passed.
	written, synced uint64
	// newEntry is true until a sync makes the directory entry of a created file
	// durable.
	newEntry bool
	failed   error
	closed   bool
	// prev and next link the handle in the idle ring of b while it is idle.
	prev, next *Handle
}

// Create creates the file name in fs, which must not exist, and returns its handle. The
// first Sync of the handle also syncs the directory of name.
func (b *Budget) Create(fs xfs.FS, name string) (*Handle, error) {
	h := &Handle{
		b:        b,
		fs:       fs,
		name:     name,
		flag:     os.O_CREATE | os.O_EXCL | os.O_RDWR,
		newEntry: true,
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
	return &Handle{b: b, fs: fs, name: name, flag: os.O_RDWR}
}

// acquire returns the open file of h and counts the caller as a user of it.
func (h *Handle) acquire() (xfs.File, error) {
	b := h.b
	b.mu.Lock()
	defer b.mu.Unlock()
	for {
		switch {
		case h.busy:
			b.cond.Wait()
		case h.closed:
			return nil, errors.Wrapf(ErrClosed, "segment file %s", h.name)
		case h.file != nil:
			h.refs++
			h.unlink()
			return h.file, nil
		case b.open < b.limit:
			return h.openFile()
		case b.idle.prev == &b.idle:
			b.cond.Wait()
		default:
			b.evict(b.idle.prev)
		}
	}
}

// openFile opens the file of h. The caller holds b.mu, which openFile releases while
// the file opens.
func (h *Handle) openFile() (xfs.File, error) {
	b := h.b
	b.open++
	h.busy = true
	b.mu.Unlock()
	file, err := h.fs.Open(h.name, h.flag)
	b.mu.Lock()
	h.busy = false
	b.cond.Broadcast()
	if err != nil {
		b.open--
		return nil, err
	}
	h.file, h.flag, h.refs = file, os.O_RDWR, 1
	return file, nil
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
		err = file.Sync()
	}
	err = errors.Combine(err, file.Close())
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

// release ends one use of the file of h, and counts a write when wrote is true.
func (h *Handle) release(wrote bool) {
	b := h.b
	b.mu.Lock()
	defer b.mu.Unlock()
	if wrote {
		h.written++
	}
	h.refs--
	if h.refs > 0 {
		return
	}
	h.prev, h.next = &b.idle, b.idle.next
	b.idle.next.prev = h
	b.idle.next = h
	b.cond.Broadcast()
}

// unlink takes h out of the idle ring when it is in it. The caller holds b.mu.
func (h *Handle) unlink() {
	if h.prev == nil {
		return
	}
	h.prev.next, h.next.prev = h.next, h.prev
	h.prev, h.next = nil, nil
}

// fail marks h failed with err. The caller holds b.mu.
func (h *Handle) fail(err error) {
	if h.failed == nil {
		h.failed = errors.Wrapf(
			err,
			"segment file %s failed, and it is never synced or written again",
			h.name,
		)
	}
}

// ReadAt reads len(p) bytes from file offset off.
func (h *Handle) ReadAt(p []byte, off int64) (int, error) {
	file, err := h.acquire()
	if err != nil {
		return 0, err
	}
	defer h.release(false)
	return file.ReadAt(p, off)
}

// WriteAt writes p at file offset off, with no sync. It returns the error of the
// failed sync when the handle is failed.
func (h *Handle) WriteAt(p []byte, off int64) (int, error) {
	file, err := h.acquire()
	if err != nil {
		return 0, err
	}
	if err = h.failure(); err != nil {
		h.release(false)
		return 0, err
	}
	n, err := file.WriteAt(p, off)
	h.release(true)
	return n, err
}

func (h *Handle) failure() error {
	h.b.mu.Lock()
	defer h.b.mu.Unlock()
	return h.failed
}

// Sync makes every write that returned before the call durable, and the directory
// entry of a created file. It returns the error of the failed sync when the handle is
// failed, from this call or an earlier one. errors.Is matches a failed sync of the
// directory with ErrDirSync.
func (h *Handle) Sync() error {
	b := h.b
	b.mu.Lock()
	written, synced, newEntry := h.written, h.synced, h.newEntry
	failed, closed := h.failed, h.closed
	b.mu.Unlock()
	switch {
	case closed:
		return errors.Wrapf(ErrClosed, "segment file %s", h.name)
	case failed != nil:
		return failed
	}
	var err error
	if written != synced {
		err = h.syncFile()
	}
	if err == nil && newEntry {
		d := dir(h.name)
		if err = h.fs.SyncDir(d); err != nil {
			err = errors.Mark(errors.Wrapf(err, "sync of directory %q", d), ErrDirSync)
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		h.fail(err)
		return h.failed
	}
	h.synced = max(h.synced, written)
	if newEntry {
		h.newEntry = false
	}
	return h.failed
}

func (h *Handle) syncFile() error {
	file, err := h.acquire()
	if err != nil {
		return err
	}
	defer h.release(false)
	return file.Sync()
}

func dir(name string) string {
	if d := path.Dir(name); d != "." {
		return d
	}
	return ""
}

// Close closes the file of h, with no sync, once no call uses it. Every later call
// returns ErrClosed.
func (h *Handle) Close() error {
	b := h.b
	b.mu.Lock()
	defer b.mu.Unlock()
	for h.busy || h.refs > 0 {
		b.cond.Wait()
	}
	if h.closed {
		return nil
	}
	h.closed = true
	if h.file == nil {
		return nil
	}
	h.unlink()
	file := h.file
	h.file = nil
	b.open--
	b.cond.Broadcast()
	return file.Close()
}
