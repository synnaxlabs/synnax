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
	"path"
	"strings"
	"sync"

	xfs "github.com/synnaxlabs/x/io/fs"
)

// counter counts the calls that change a disk. It lets one such call run at a time, so
// the disk does not change while after runs.
type counter struct {
	// after runs behind each event, with mu held.
	after func(event int)
	// synced holds the length that the last Sync of each file made durable, by path.
	synced map[string]int64
	// last is the newest Sync that made a file longer. A Sync that makes no new byte
	// durable gives the engine nothing new to confirm, so it does not count.
	last lastSync
	// events is the count of calls that changed the disk.
	events int
	mu     sync.Mutex
}

// lastSync says what one Sync of a file made durable.
type lastSync struct {
	path string
	// before and after are the durable lengths of the file around the Sync.
	before, after int64
	// event is the count of events before the Sync.
	event int
}

func (c *counter) event() {
	c.events++
	if c.after != nil {
		c.after(c.events)
	}
}

// forget drops the synced lengths of the path p and of each path under it.
func (c *counter) forget(p string) {
	for name := range c.synced {
		if name == p || strings.HasPrefix(name, p+"/") {
			delete(c.synced, name)
		}
	}
}

// countedFS passes each call to inner and counts the calls that change the disk.
type countedFS struct {
	inner xfs.FS
	c     *counter
	// dir is the path of inner under the root of the disk.
	dir string
}

var _ xfs.FS = (*countedFS)(nil)

func (f *countedFS) Open(name string, flag int) (xfs.File, error) {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	existed, err := f.inner.Exists(name)
	if err != nil {
		return nil, err
	}
	file, err := f.inner.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if !existed || flag&os.O_TRUNC != 0 {
		f.c.event()
	}
	return &countedFile{File: file, c: f.c, path: path.Join(f.dir, name)}, nil
}

func (f *countedFS) Sub(name string) (xfs.FS, error) {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	existed, err := f.inner.Exists(name)
	if err != nil {
		return nil, err
	}
	sub, err := f.inner.Sub(name)
	if err != nil {
		return nil, err
	}
	if !existed {
		f.c.event()
	}
	return &countedFS{inner: sub, c: f.c, dir: path.Join(f.dir, name)}, nil
}

func (f *countedFS) List(name string) ([]xfs.FileInfo, error) {
	return f.inner.List(name)
}

func (f *countedFS) Exists(name string) (bool, error) { return f.inner.Exists(name) }

func (f *countedFS) Stat(name string) (xfs.FileInfo, error) {
	return f.inner.Stat(name)
}

func (f *countedFS) Remove(name string) error {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	f.c.forget(path.Join(f.dir, name))
	return f.inner.Remove(name)
}

func (f *countedFS) Rename(oldPath, newPath string) error {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	f.c.forget(path.Join(f.dir, oldPath))
	f.c.forget(path.Join(f.dir, newPath))
	return f.inner.Rename(oldPath, newPath)
}

func (f *countedFS) SyncDir(name string) error {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	return f.inner.SyncDir(name)
}

// countedFile passes each call to File and counts the calls that change the disk.
type countedFile struct {
	xfs.File
	c    *counter
	path string
}

func (f *countedFile) Write(p []byte) (int, error) {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	return f.File.Write(p)
}

func (f *countedFile) WriteAt(p []byte, off int64) (int, error) {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	return f.File.WriteAt(p, off)
}

func (f *countedFile) Truncate(size int64) error {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	return f.File.Truncate(size)
}

func (f *countedFile) Sync() error {
	f.c.mu.Lock()
	defer f.c.mu.Unlock()
	defer f.c.event()
	if err := f.File.Sync(); err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if before, after := f.c.synced[f.path], info.Size(); after > before {
		f.c.last = lastSync{
			path:   f.path,
			before: before,
			after:  after,
			event:  f.c.events,
		}
	}
	if !info.IsDir() {
		f.c.synced[f.path] = info.Size()
	}
	return nil
}
