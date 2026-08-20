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

// ErrFault is the error a wrapped FS raises when Options names no other.
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

// Failure makes one operation fail.
type Failure struct {
	// Op is the operation that fails. Rename matches on its old path.
	Op FaultOp
	// Name is the path Op fails on. An empty Name fails Op on every path.
	Name string
	// After holds the failure back until this operation has run at least once,
	// letting a test fail the second half of a sequence. A zero After fails Op
	// straight away.
	After FaultOp
}

// Options selects the operations a wrapped FS fails. A zero Options fails nothing.
type Options struct {
	// Fail lists the failures the FS raises.
	Fail []Failure
	// Err replaces every failed operation. Defaults to ErrFault.
	Err error
}

// FaultyFS wraps an FS to fail the operations its Options select. It also counts the
// file handles open against it, so a test can assert that a failed operation left none
// behind.
type FaultyFS struct {
	xfs.FS
	open *atomic.Int64
	mu   *faultState
}

type faultState struct {
	sync.RWMutex
	opts Options
	ran  set.Set[FaultOp]
}

// WrapFS wraps fs so that every operation opts selects fails. Paths reach opts exactly
// as the caller passed them.
func WrapFS(fs xfs.FS, opts Options) *FaultyFS {
	return &FaultyFS{
		FS:   fs,
		open: &atomic.Int64{},
		mu:   &faultState{opts: opts, ran: make(set.Set[FaultOp])},
	}
}

// SetOptions replaces the failures the FS raises. Passing a zero Options clears them.
func (fs *FaultyFS) SetOptions(opts Options) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.mu.opts = opts
}

// OpenFiles returns the number of handles opened through the FS and not yet closed.
func (fs *FaultyFS) OpenFiles() int { return int(fs.open.Load()) }

// fault records that op ran against name and returns the error it must fail with, or
// nil to let it through.
func (fs *FaultyFS) fault(op FaultOp, name string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.mu.ran.Add(op)
	for _, f := range fs.mu.opts.Fail {
		if f.Op != op || (f.Name != "" && f.Name != name) {
			continue
		}
		if f.After != "" && !fs.mu.ran.Contains(f.After) {
			continue
		}
		if fs.mu.opts.Err != nil {
			return fs.mu.opts.Err
		}
		return ErrFault
	}
	return nil
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
