// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pebblefs runs Pebble on an fs.FS, so Pebble and another store share one disk,
// and the fault disk and the crash harness see the calls of both.
package pebblefs

import (
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// errNoLinks is the error of Link: an fs.FS has no hard links. Pebble links a file
// only through vfs.LinkOrCopy, which then copies.
var errNoLinks = errors.New("pebblefs: no hard links")

// space is the free space that GetDiskUsage reports. A full disk shows as ENOSPC from a
// write.
const space = 1 << 40

type fs struct {
	inner xfs.FS
	// locked holds the names of the locks that are held.
	locked set.Set[string]
	// dirs holds the open directory handles, so a rename can move them.
	dirs set.Set[*dir]
	mu   sync.Mutex
}

var _ vfs.FS = (*fs)(nil)

// New returns a vfs.FS of Pebble whose calls go to inner, with the semantics of
// vfs.Default on Linux. Paths use slashes. A lock is held in memory, so it excludes
// only
// the opens on the returned FS, and a directory handle follows only the renames made
// through it.
func New(inner xfs.FS) vfs.FS {
	return &fs{inner: inner, locked: set.Set[string]{}, dirs: set.Set[*dir]{}}
}

func (f *fs) open(name string, flag int, opts []vfs.OpenOption) (vfs.File, error) {
	inner, err := f.inner.Open(name, flag)
	if err != nil {
		return nil, err
	}
	out := &file{inner: inner}
	for _, opt := range opts {
		opt.Apply(out)
	}
	return out, nil
}

// Create removes a file at name before it creates one, so a handle on the old file
// keeps the old bytes.
func (f *fs) Create(name string, _ vfs.DiskWriteCategory) (vfs.File, error) {
	const flag = os.O_CREATE | os.O_EXCL | os.O_RDWR
	out, err := f.open(name, flag, nil)
	if !errors.Is(err, os.ErrExist) {
		return out, err
	}
	if err = f.Remove(name); err != nil {
		return nil, err
	}
	return f.open(name, flag, nil)
}

func (f *fs) Link(oldname, newname string) error {
	return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: errNoLinks}
}

func (f *fs) Open(name string, opts ...vfs.OpenOption) (vfs.File, error) {
	return f.open(name, os.O_RDONLY, opts)
}

func (f *fs) OpenReadWrite(
	name string,
	_ vfs.DiskWriteCategory,
	opts ...vfs.OpenOption,
) (vfs.File, error) {
	return f.open(name, os.O_CREATE|os.O_RDWR, opts)
}

func (f *fs) OpenDir(name string) (vfs.File, error) {
	info, err := f.inner.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOTDIR}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	d := &dir{fs: f, name: path.Clean(name)}
	f.dirs.Add(d)
	return d, nil
}

// Remove removes the file or the empty directory name, as os.Remove does.
func (f *fs) Remove(name string) error {
	info, err := f.inner.Stat(name)
	if err != nil {
		return err
	}
	if info.IsDir() {
		entries, err := f.inner.List(name)
		if err != nil {
			return err
		}
		if len(entries) > 0 {
			return &os.PathError{Op: "remove", Path: name, Err: syscall.ENOTEMPTY}
		}
	}
	return f.inner.Remove(name)
}

func (f *fs) RemoveAll(name string) error { return f.inner.Remove(name) }

// Rename fails onto a directory, as os.Rename does.
func (f *fs) Rename(oldname, newname string) error {
	info, err := f.inner.Stat(newname)
	if err == nil && info.IsDir() && path.Clean(oldname) != path.Clean(newname) {
		return &os.LinkError{
			Op:  "rename",
			Old: oldname,
			New: newname,
			Err: syscall.EEXIST,
		}
	}
	if err = f.inner.Rename(oldname, newname); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	from, to := path.Clean(oldname), path.Clean(newname)
	for d := range f.dirs {
		if d.name == from {
			d.name = to
		} else if rest, ok := strings.CutPrefix(d.name, from+"/"); ok {
			d.name = path.Join(to, rest)
		}
	}
	return nil
}

func (f *fs) ReuseForWrite(
	oldname, newname string,
	category vfs.DiskWriteCategory,
) (vfs.File, error) {
	if err := f.Rename(oldname, newname); err != nil {
		return nil, err
	}
	return f.OpenReadWrite(newname, category)
}

func (f *fs) MkdirAll(dir string, _ os.FileMode) error {
	_, err := f.inner.Sub(dir)
	return err
}

func (f *fs) Lock(name string) (io.Closer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.locked.Contains(name) {
		return nil, &os.PathError{Op: "lock", Path: name, Err: syscall.EAGAIN}
	}
	lock, err := f.inner.Open(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	if err = lock.Close(); err != nil {
		return nil, err
	}
	f.locked.Add(name)
	return &unlock{fs: f, name: name}, nil
}

func (f *fs) List(dir string) ([]string, error) {
	infos, err := f.inner.List(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name()
	}
	return names, nil
}

func (f *fs) Stat(name string) (vfs.FileInfo, error) {
	info, err := f.inner.Stat(name)
	if err != nil {
		return nil, err
	}
	return fileInfo{FileInfo: info}, nil
}

func (f *fs) PathBase(p string) string { return path.Base(p) }

func (f *fs) PathJoin(elem ...string) string { return path.Join(elem...) }

func (f *fs) PathDir(p string) string { return path.Dir(p) }

func (f *fs) GetDiskUsage(string) (vfs.DiskUsage, error) {
	return vfs.DiskUsage{AvailBytes: space, TotalBytes: space}, nil
}

func (f *fs) Unwrap() vfs.FS { return nil }

type unlock struct {
	fs   *fs
	name string
}

func (u *unlock) Close() error {
	u.fs.mu.Lock()
	defer u.fs.mu.Unlock()
	u.fs.locked.Remove(u.name)
	return nil
}

// fileInfo gives the info of an fs.FS the device of Pebble: one device for all files.
type fileInfo struct{ xfs.FileInfo }

func (fileInfo) DeviceID() vfs.DeviceID { return vfs.DeviceID{} }

// file is a handle with one offset for Read and Write, as an os.File has.
type file struct {
	inner xfs.File
	off   int64
}

var _ vfs.File = (*file)(nil)

func (f *file) Read(p []byte) (int, error) {
	n, err := f.inner.ReadAt(p, f.off)
	f.off += int64(n)
	if n > 0 && errors.Is(err, io.EOF) {
		err = nil
	}
	return n, err
}

func (f *file) ReadAt(
	p []byte,
	off int64,
) (int, error) {
	return f.inner.ReadAt(p, off)
}

func (f *file) Write(p []byte) (int, error) {
	n, err := f.inner.WriteAt(p, f.off)
	f.off += int64(n)
	return n, err
}

func (f *file) WriteAt(
	p []byte,
	off int64,
) (int, error) {
	return f.inner.WriteAt(p, off)
}

func (f *file) Close() error { return f.inner.Close() }

func (f *file) Sync() error { return f.inner.Sync() }

func (f *file) Preallocate(int64, int64) error { return nil }

func (f *file) Stat() (vfs.FileInfo, error) {
	info, err := f.inner.Stat()
	if err != nil {
		return nil, err
	}
	return fileInfo{FileInfo: info}, nil
}

func (f *file) SyncTo(int64) (bool, error) { return true, f.Sync() }

func (f *file) SyncData() error { return f.Sync() }

func (f *file) Prefetch(int64, int64) error { return nil }

func (f *file) Fd() uintptr { return vfs.InvalidFd }

// dir is a directory opened for a Sync, which makes its entries durable. Its name
// follows a rename through the fs, under the lock of the fs.
type dir struct {
	fs   *fs
	name string
}

var _ vfs.File = (*dir)(nil)

func (d *dir) path() string {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()
	return d.name
}

func (d *dir) Close() error {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()
	d.fs.dirs.Remove(d)
	return nil
}

func (d *dir) Read([]byte) (int, error) { return 0, d.isDir("read") }

func (d *dir) ReadAt([]byte, int64) (int, error) { return 0, d.isDir("read") }

func (d *dir) Write([]byte) (int, error) { return 0, d.isDir("write") }

func (d *dir) WriteAt([]byte, int64) (int, error) { return 0, d.isDir("write") }

func (d *dir) isDir(op string) error {
	return &os.PathError{Op: op, Path: d.path(), Err: syscall.EISDIR}
}

func (d *dir) Preallocate(int64, int64) error { return nil }

func (d *dir) Stat() (vfs.FileInfo, error) {
	info, err := d.fs.inner.Stat(d.path())
	if err != nil {
		return nil, err
	}
	return fileInfo{FileInfo: info}, nil
}

func (d *dir) Sync() error { return d.fs.inner.SyncDir(d.path()) }

// SyncTo does nothing on a directory, as on vfs.Default.
func (d *dir) SyncTo(int64) (bool, error) { return false, nil }

func (d *dir) SyncData() error { return d.Sync() }

func (d *dir) Prefetch(int64, int64) error { return nil }

func (d *dir) Fd() uintptr { return vfs.InvalidFd }
