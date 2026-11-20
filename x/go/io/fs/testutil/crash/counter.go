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
	// files holds, for each file, the Syncs that changed its durable bytes.
	files map[fault.ID]*syncs
	// unit holds the ranges that the newest Sync with a changed block made durable, in
	// the file last. A Sync that changes no durable byte gives the engine nothing new
	// to confirm, so it does not count.
	unit []fault.Span
	last fault.ID
	// lossFrom holds, for each range of unit, the number among the events of the first
	// Sync that changed the first block of the range. A stale block of the range can
	// take each commit from that Sync on.
	lossFrom []int
	// events is the count of calls that changed the disk.
	events int
	// faults is the count of calls that the disk failed with a sync error or with no
	// space.
	faults int
	mu     sync.Mutex
}

// blockSize is the unit that a disk loses or keeps as a whole.
const blockSize = 4096

// syncs is what the counter knows of the Syncs of one file.
type syncs struct {
	// first holds, for each block, the number of the first Sync that changed it.
	first map[int64]int
	// count is the count of Syncs that changed a durable byte of the file.
	count uint64
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

// synced notes a Sync of the file of stat.
func (c *counter) synced(stat fault.Stat) {
	file, ok := c.files[stat.ID]
	if !ok {
		file = &syncs{first: map[int64]int{}}
		c.files[stat.ID] = file
	}
	if stat.Syncs == file.count {
		return
	}
	file.count = stat.Syncs
	for _, s := range stat.Synced {
		for block := s.From / blockSize; block*blockSize < s.To; block++ {
			if _, ok := file.first[block]; !ok {
				file.first[block] = c.events + 1
			}
		}
	}
	c.unit, c.last, c.lossFrom = stat.Synced, stat.ID, c.lossFrom[:0]
	for _, s := range stat.Synced {
		c.lossFrom = append(c.lossFrom, file.first[s.From/blockSize])
	}
}

// statOf returns what a fault disk says of the file of info.
func statOf(info xfs.FileInfo) (fault.Stat, error) {
	stat, ok := info.Sys().(fault.Stat)
	if !ok {
		return stat, errors.Newf("crash: %s is not a file of a fault disk", info.Name())
	}
	return stat, nil
}

// Writeback makes each write and each directory change on the disk of fs durable, as
// fault.FS.Writeback does, in one event. fs is an FS that Run gave the engine of the
// workload, or a Sub of it.
func Writeback(fs xfs.FS) error {
	counted, ok := fs.(*countedFS)
	if !ok {
		return errors.New("crash: writeback needs the FS of the workload")
	}
	disk, ok := counted.inner.(*fault.FS)
	if !ok {
		return errors.New("crash: writeback needs a fault disk")
	}
	counted.c.begin()
	defer counted.c.end(nil, true)
	disk.Writeback()
	return nil
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
	stat, err := statOf(info)
	if err != nil {
		return err
	}
	f.c.synced(stat)
	return nil
}
