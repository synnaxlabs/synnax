// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"fmt"
	"sync"

	xfs "github.com/synnaxlabs/x/io/fs"
)

// opFS records the writes, syncs, truncates, and removes of a file system, fails a
// chosen sync, and flips a bit of chosen reads.
type opFS struct {
	xfs.FS
	mu  sync.Mutex
	ops []string
	// failSync fails the sync with this number, from 1, and no other.
	failSync int
	syncs    int
	dirErr   error
	// flips holds, for a file offset, how many more reads that cover it get a flipped
	// bit. A negative count flips every read.
	flips map[int64]int
	reads map[int64]int
}

func newOpFS(fs xfs.FS) *opFS {
	return &opFS{FS: fs, flips: map[int64]int{}, reads: map[int64]int{}}
}

func (f *opFS) recordf(format string, args ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, fmt.Sprintf(format, args...))
}

// Ops returns the recorded operations.
func (f *opFS) Ops() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.ops...)
}

// Flip flips a bit at file offset off in the next n reads that cover it, or in every
// read for a negative n.
func (f *opFS) Flip(off int64, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.flips[off] = n
}

// Reads returns how many reads covered file offset off.
func (f *opFS) Reads(off int64) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reads[off]
}

func (f *opFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &opFile{File: file, fs: f, name: name}, nil
}

func (f *opFS) Remove(name string) error {
	f.recordf("remove %s", name)
	return f.FS.Remove(name)
}

func (f *opFS) SyncDir(name string) error {
	f.mu.Lock()
	err := f.dirErr
	f.mu.Unlock()
	if err != nil {
		return err
	}
	f.recordf("syncdir %q", name)
	return f.FS.SyncDir(name)
}

type opFile struct {
	xfs.File
	fs   *opFS
	name string
}

func (f *opFile) WriteAt(p []byte, off int64) (int, error) {
	f.fs.recordf("write %s %d %d", f.name, off, len(p))
	return f.File.WriteAt(p, off)
}

func (f *opFile) Truncate(size int64) error {
	f.fs.recordf("truncate %s %d", f.name, size)
	return f.File.Truncate(size)
}

func (f *opFile) Sync() error {
	f.fs.mu.Lock()
	f.fs.syncs++
	fail := f.fs.syncs == f.fs.failSync
	f.fs.mu.Unlock()
	if fail {
		return errSync
	}
	f.fs.recordf("sync %s", f.name)
	return f.File.Sync()
}

func (f *opFile) ReadAt(p []byte, off int64) (int, error) {
	n, err := f.File.ReadAt(p, off)
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	for at, left := range f.fs.flips {
		if at < off || at >= off+int64(n) {
			continue
		}
		f.fs.reads[at]++
		if left != 0 {
			p[at-off] ^= 0x01
			if left > 0 {
				f.fs.flips[at] = left - 1
			}
		}
	}
	return n, err
}
