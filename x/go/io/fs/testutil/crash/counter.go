// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash

import (
	"os"
	"sync"
	"syscall"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
)

// counter counts the calls that change a disk. It lets one such call run at a time, so
// the disk does not change while before and after run.
type counter struct {
	// before runs at the start of each call that can change the disk, and after runs
	// behind each call that did. Both run with mu held.
	before func()
	after  func(event int)
	// grown says, for each file, how its durable length grew.
	grown map[fault.ID]growth
	// last is the file that the newest Sync made longer. A Sync that makes no new byte
	// durable gives the engine nothing new to confirm, so it does not count.
	last fault.ID
	// events is the count of calls that changed the disk.
	events int
	// faults is the count of calls that the disk failed with a sync error or with no
	// space.
	faults int
	mu     sync.Mutex
}

// blockSize is the unit that a disk loses or keeps as a whole.
const blockSize = 4096

// growth says how the durable length of a file grew.
type growth struct {
	// size is the durable length of the file.
	size int64
	// before is the durable length of the file before the newest Sync that made it
	// longer. No later Sync confirms a byte from before on.
	before int64
	// event is the number, among the events, of the first Sync into the block of the
	// byte at before. A stale block there takes each commit from that Sync on.
	event int
	// tailEvent is the number of the first Sync into the block of the last durable
	// byte.
	tailEvent int
	// cut says that a Sync made the file shorter. The blocks of such a file have no
	// known first Sync, so the file is no target of lasting corruption.
	cut bool
}

// begin starts a call that can change the disk.
func (c *counter) begin() {
	c.mu.Lock()
	c.before()
}

// end ends a call that began. It notes a fault of the disk in err, and counts the call
// as an event when it changed the disk.
func (c *counter) end(err error, changed bool) {
	if errors.Is(err, syscall.EIO) || errors.Is(err, syscall.ENOSPC) {
		c.faults++
	}
	if changed {
		c.events++
		c.after(c.events)
	}
	c.mu.Unlock()
}

// synced notes that a Sync made the first size bytes of file durable.
func (c *counter) synced(file fault.ID, size int64) {
	g := c.grown[file]
	if size > g.size {
		first := g.size / blockSize
		if g.size%blockSize == 0 {
			g.tailEvent = c.events + 1
		}
		g.before, g.event = g.size, g.tailEvent
		if (size-1)/blockSize != first {
			g.tailEvent = c.events + 1
		}
		c.last = file
	}
	g.cut = g.cut || size < g.size
	g.size = size
	c.grown[file] = g
}

// fileID returns the ID that a fault disk gave to the file of info.
func fileID(info xfs.FileInfo) (fault.ID, error) {
	id, ok := info.Sys().(fault.ID)
	if !ok {
		return 0, errors.Newf("crash: %s is not a file of a fault disk", info.Name())
	}
	return id, nil
}

// countedFS passes each call to inner, an FS of a fault disk, and counts the calls that
// change the disk.
type countedFS struct {
	inner xfs.FS
	c     *counter
}

var _ xfs.FS = (*countedFS)(nil)

func (f *countedFS) Open(name string, flag int) (file xfs.File, err error) {
	changed := false
	f.c.begin()
	defer func() { f.c.end(err, changed) }()
	existed, err := f.inner.Exists(name)
	if err != nil {
		return nil, err
	}
	if file, err = f.inner.Open(name, flag); err != nil {
		return nil, err
	}
	changed = !existed || flag&os.O_TRUNC != 0
	return &countedFile{File: file, c: f.c}, nil
}

func (f *countedFS) Sub(name string) (sub xfs.FS, err error) {
	changed := false
	f.c.begin()
	defer func() { f.c.end(err, changed) }()
	existed, err := f.inner.Exists(name)
	if err != nil {
		return nil, err
	}
	if sub, err = f.inner.Sub(name); err != nil {
		return nil, err
	}
	changed = !existed
	return &countedFS{inner: sub, c: f.c}, nil
}

func (f *countedFS) List(name string) ([]xfs.FileInfo, error) {
	return f.inner.List(name)
}

func (f *countedFS) Exists(name string) (bool, error) { return f.inner.Exists(name) }

func (f *countedFS) Stat(name string) (xfs.FileInfo, error) {
	return f.inner.Stat(name)
}

func (f *countedFS) Remove(name string) (err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.inner.Remove(name)
}

func (f *countedFS) Rename(oldPath, newPath string) (err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.inner.Rename(oldPath, newPath)
}

func (f *countedFS) SyncDir(name string) (err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.inner.SyncDir(name)
}

// countedFile passes each call to File and counts the calls that change the disk.
type countedFile struct {
	xfs.File
	c *counter
}

func (f *countedFile) Write(p []byte) (n int, err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.File.Write(p)
}

func (f *countedFile) WriteAt(p []byte, off int64) (n int, err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.File.WriteAt(p, off)
}

func (f *countedFile) Truncate(size int64) (err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	return f.File.Truncate(size)
}

func (f *countedFile) Sync() (err error) {
	f.c.begin()
	defer func() { f.c.end(err, true) }()
	if err = f.File.Sync(); err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	id, err := fileID(info)
	if err != nil {
		return err
	}
	f.c.synced(id, info.Size())
	return nil
}
