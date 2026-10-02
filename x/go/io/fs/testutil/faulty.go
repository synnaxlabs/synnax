// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// ErrFault is the error a wrapped FS raises in place of a failed operation.
var ErrFault = errors.New("faulty file system")

// FaultOp names a file system operation a wrapped FS can fail.
type FaultOp string

const (
	// FaultOpOpen names FS.Open.
	FaultOpOpen FaultOp = "open"
	// FaultOpReadAt names File.ReadAt.
	FaultOpReadAt FaultOp = "read_at"
	// FaultOpWrite names File.Write.
	FaultOpWrite FaultOp = "write"
	// FaultOpRename names FS.Rename.
	FaultOpRename FaultOp = "rename"
	// FaultOpStat names FS.Stat.
	FaultOpStat FaultOp = "stat"
	// FaultOpRemove names FS.Remove.
	FaultOpRemove FaultOp = "remove"
)

// FaultyFSOption selects an operation a wrapped FS fails.
type FaultyFSOption func(*options)

type options struct {
	// fail holds the paths that fail, keyed by operation. An empty set fails its
	// operation on every path.
	fail map[FaultOp]set.Set[string]
	// after holds every failure back until this operation has run.
	after FaultOp
}

func newOptions(opts ...FaultyFSOption) options {
	o := options{fail: make(map[FaultOp]set.Set[string])}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func failOn(op FaultOp, names []string) FaultyFSOption {
	return func(o *options) {
		existing, ok := o.fail[op]
		if !ok {
			existing = make(set.Set[string])
			o.fail[op] = existing
		}
		existing.Add(names...)
	}
}

// WithFailOpen fails Open on each of the given paths. With no path, every Open fails.
func WithFailOpen(names ...string) FaultyFSOption { return failOn(FaultOpOpen, names) }

// WithFailReadAt fails ReadAt on each of the given paths. With no path, every ReadAt
// fails.
func WithFailReadAt(names ...string) FaultyFSOption {
	return failOn(FaultOpReadAt, names)
}

// WithFailWrite fails Write on each of the given paths. With no path, every Write
// fails.
func WithFailWrite(names ...string) FaultyFSOption {
	return failOn(FaultOpWrite, names)
}

// WithFailRename fails Rename on each of the given old paths. With no path, every
// Rename fails.
func WithFailRename(names ...string) FaultyFSOption {
	return failOn(FaultOpRename, names)
}

// WithFailStat fails Stat on each of the given paths. With no path, every Stat fails.
func WithFailStat(names ...string) FaultyFSOption { return failOn(FaultOpStat, names) }

// WithFailRemove fails Remove on each of the given paths. With no path, every Remove
// fails.
func WithFailRemove(names ...string) FaultyFSOption {
	return failOn(FaultOpRemove, names)
}

// WithFailAfter holds every failure back until op has run at least once, so a test can
// fail the second half of a sequence.
func WithFailAfter(op FaultOp) FaultyFSOption {
	return func(o *options) { o.after = op }
}

// FaultyFS wraps an FS to fail the operations its options select. It also counts the
// file handles open against it, so a test can assert that a failed operation left none
// behind.
type FaultyFS struct {
	wrapped xfs.FS
	open    *atomic.Int64
	mu      *faultState
}

var _ xfs.FS = (*FaultyFS)(nil)

type faultState struct {
	sync.Mutex
	opts options
	ran  set.Set[FaultOp]
}

// WrapFaultyFS wraps fs so that every operation opts selects fails. Paths reach the
// options exactly as the caller passed them.
func WrapFaultyFS(fs xfs.FS, opts ...FaultyFSOption) *FaultyFS {
	return &FaultyFS{
		wrapped: fs,
		open:    &atomic.Int64{},
		mu:      &faultState{opts: newOptions(opts...), ran: make(set.Set[FaultOp])},
	}
}

// SetOptions replaces the failures the FS raises. Passing no option clears them.
func (fs *FaultyFS) SetOptions(opts ...FaultyFSOption) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.mu.opts = newOptions(opts...)
}

// OpenFiles returns the number of handles opened through the FS and not yet closed.
func (fs *FaultyFS) OpenFiles() int { return int(fs.open.Load()) }

func (fs *FaultyFS) fault(op FaultOp, name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.mu.ran.Add(op)
	if fs.mu.opts.after != "" && !fs.mu.ran.Contains(fs.mu.opts.after) {
		return nil
	}
	names, ok := fs.mu.opts.fail[op]
	if !ok || (len(names) > 0 && !names.Contains(name)) {
		return nil
	}
	return ErrFault
}

// Sub returns a FaultyFS over the sub-directory, sharing this FS's failures and handle
// count. Paths reach the options relative to the sub-FS.
func (fs *FaultyFS) Sub(name string) (xfs.FS, error) {
	sub, err := fs.wrapped.Sub(name)
	if err != nil {
		return nil, err
	}
	return &FaultyFS{wrapped: sub, open: fs.open, mu: fs.mu}, nil
}

// Open opens the file, counting the handle it returns until that handle is closed.
// Reads and writes through the handle fail as the options select. Open returns ErrFault
// when WithFailOpen covers name.
func (fs *FaultyFS) Open(name string, flag int) (xfs.File, error) {
	if err := fs.fault(FaultOpOpen, name); err != nil {
		return nil, err
	}
	f, err := fs.wrapped.Open(name, flag)
	if err != nil {
		return nil, err
	}
	fs.open.Add(1)
	return &faultyFile{File: f, fs: fs, name: name}, nil
}

// Rename returns ErrFault when WithFailRename covers oldPath, and renames otherwise.
func (fs *FaultyFS) Rename(oldPath, newPath string) error {
	if err := fs.fault(FaultOpRename, oldPath); err != nil {
		return err
	}
	return fs.wrapped.Rename(oldPath, newPath)
}

// Stat returns ErrFault when WithFailStat covers name, and stats otherwise.
func (fs *FaultyFS) Stat(name string) (xfs.FileInfo, error) {
	if err := fs.fault(FaultOpStat, name); err != nil {
		return nil, err
	}
	return fs.wrapped.Stat(name)
}

// Remove returns ErrFault when WithFailRemove covers name, and removes otherwise.
func (fs *FaultyFS) Remove(name string) error {
	if err := fs.fault(FaultOpRemove, name); err != nil {
		return err
	}
	return fs.wrapped.Remove(name)
}

// List passes through to the wrapped FS. No option fails it.
func (fs *FaultyFS) List(name string) ([]xfs.FileInfo, error) {
	return fs.wrapped.List(name)
}

// Exists passes through to the wrapped FS. No option fails it.
func (fs *FaultyFS) Exists(name string) (bool, error) { return fs.wrapped.Exists(name) }

type faultyFile struct {
	xfs.File
	fs   *FaultyFS
	name string
}

func (f *faultyFile) ReadAt(p []byte, off int64) (int, error) {
	if err := f.fs.fault(FaultOpReadAt, f.name); err != nil {
		return 0, err
	}
	return f.File.ReadAt(p, off)
}

func (f *faultyFile) Write(p []byte) (int, error) {
	if err := f.fs.fault(FaultOpWrite, f.name); err != nil {
		return 0, err
	}
	return f.File.Write(p)
}

func (f *faultyFile) Close() error { f.fs.open.Add(-1); return f.File.Close() }
