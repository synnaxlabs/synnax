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

// ErrFault is the error a wrapped FS raises when WithFailErr names no other.
var ErrFault = errors.New("faulty file system")

// FaultOp names a file system operation a wrapped FS can fail.
type FaultOp string

const (
	FaultOpOpen   FaultOp = "open"
	FaultOpReadAt FaultOp = "read_at"
	FaultOpWrite  FaultOp = "write"
	FaultOpRename FaultOp = "rename"
	FaultOpStat   FaultOp = "stat"
	FaultOpRemove FaultOp = "remove"
)

// Option selects an operation a wrapped FS fails.
type Option func(*options)

type options struct {
	// fail holds the paths that fail, keyed by operation. An empty set fails its
	// operation on every path.
	fail map[FaultOp]set.Set[string]
	// err replaces every failed operation. Defaults to ErrFault.
	err error
	// after holds every failure back until this operation has run.
	after FaultOp
}

func newOptions(opts ...Option) options {
	o := options{fail: make(map[FaultOp]set.Set[string])}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func failOn(op FaultOp, names []string) Option {
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
func WithFailOpen(names ...string) Option { return failOn(FaultOpOpen, names) }

// WithFailReadAt fails ReadAt on each of the given paths. With no path, every ReadAt
// fails.
func WithFailReadAt(names ...string) Option { return failOn(FaultOpReadAt, names) }

// WithFailWrite fails Write on each of the given paths. With no path, every Write
// fails.
func WithFailWrite(names ...string) Option { return failOn(FaultOpWrite, names) }

// WithFailRename fails Rename on each of the given old paths. With no path, every
// Rename fails.
func WithFailRename(names ...string) Option { return failOn(FaultOpRename, names) }

// WithFailStat fails Stat on each of the given paths. With no path, every Stat fails.
func WithFailStat(names ...string) Option { return failOn(FaultOpStat, names) }

// WithFailRemove fails Remove on each of the given paths. With no path, every Remove
// fails.
func WithFailRemove(names ...string) Option { return failOn(FaultOpRemove, names) }

// WithFailErr raises err in place of ErrFault.
func WithFailErr(err error) Option { return func(o *options) { o.err = err } }

// WithFailAfter holds every failure back until op has run at least once, so a test can
// fail the second half of a sequence.
func WithFailAfter(op FaultOp) Option { return func(o *options) { o.after = op } }

// FaultyFS wraps an FS to fail the operations its options select. It also counts the
// file handles open against it, so a test can assert that a failed operation left none
// behind.
type FaultyFS struct {
	xfs.FS
	open *atomic.Int64
	mu   *faultState
}

type faultState struct {
	sync.Mutex
	opts options
	// ran holds every operation the FS has seen, backing WithFailAfter.
	ran set.Set[FaultOp]
}

// WrapFS wraps fs so that every operation opts selects fails. Paths reach the options
// exactly as the caller passed them.
func WrapFS(fs xfs.FS, opts ...Option) *FaultyFS {
	return &FaultyFS{
		FS:   fs,
		open: &atomic.Int64{},
		mu:   &faultState{opts: newOptions(opts...), ran: make(set.Set[FaultOp])},
	}
}

// SetOptions replaces the failures the FS raises. Passing no option clears them.
func (fs *FaultyFS) SetOptions(opts ...Option) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.mu.opts = newOptions(opts...)
}

// OpenFiles returns the number of handles opened through the FS and not yet closed.
func (fs *FaultyFS) OpenFiles() int { return int(fs.open.Load()) }

// fault records that op ran against name and returns the error it must fail with, or
// nil to let it through.
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
	if fs.mu.opts.err != nil {
		return fs.mu.opts.err
	}
	return ErrFault
}

func (fs *FaultyFS) Sub(name string) (xfs.FS, error) {
	sub, err := fs.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &FaultyFS{FS: sub, open: fs.open, mu: fs.mu}, nil
}

func (fs *FaultyFS) Open(name string, flag int) (xfs.File, error) {
	if err := fs.fault(FaultOpOpen, name); err != nil {
		return nil, err
	}
	f, err := fs.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	fs.open.Add(1)
	return &faultyFile{File: f, fs: fs, name: name}, nil
}

func (fs *FaultyFS) Rename(oldPath, newPath string) error {
	if err := fs.fault(FaultOpRename, oldPath); err != nil {
		return err
	}
	return fs.FS.Rename(oldPath, newPath)
}

func (fs *FaultyFS) Stat(name string) (xfs.FileInfo, error) {
	if err := fs.fault(FaultOpStat, name); err != nil {
		return nil, err
	}
	return fs.FS.Stat(name)
}

func (fs *FaultyFS) Remove(name string) error {
	if err := fs.fault(FaultOpRemove, name); err != nil {
		return err
	}
	return fs.FS.Remove(name)
}

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

func (f *faultyFile) Close() error {
	f.fs.open.Add(-1)
	return f.File.Close()
}
