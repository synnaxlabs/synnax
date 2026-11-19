// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package fault provides an FS in memory that models a disk, for crash and fault tests.
package fault

import (
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// FS is a disk in memory. It is safe for concurrent use, and each call is atomic.
type FS struct {
	d *disk
	// root is the cleaned absolute path of the directory that names resolve against.
	root string
}

var _ xfs.FS = (*FS)(nil)

// New returns an empty FS.
func New() *FS {
	return &FS{
		d:    &disk{root: &node{name: "/", dir: true, entries: map[string]*node{}}},
		root: "/",
	}
}

func (f *FS) abs(name string) string { return path.Join(f.root, name) }

// Open implements fs.FS.
func (f *FS) Open(name string, flag int) (xfs.File, error) {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.lookup("open", p)
	switch {
	case err == nil:
		if flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0 {
			return nil, pathError("open", p, os.ErrExist)
		}
		if flag&os.O_TRUNC != 0 && !n.dir {
			d.resize(n, 0)
		}
	case errors.Is(err, os.ErrNotExist) && flag&os.O_CREATE != 0:
		dir, base, err := d.parent("open", p)
		if err != nil {
			return nil, err
		}
		n = &node{}
		d.bind(dir, base, n)
		d.touch(n)
	default:
		return nil, err
	}
	return &file{
		d:      d,
		n:      n,
		path:   p,
		read:   flag&os.O_WRONLY == 0,
		write:  flag&(os.O_WRONLY|os.O_RDWR) != 0,
		append: flag&os.O_APPEND != 0,
	}, nil
}

// Sub implements fs.FS.
func (f *FS) Sub(name string) (xfs.FS, error) {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n := d.root
	for frag := range strings.SplitSeq(strings.TrimPrefix(p, "/"), "/") {
		if frag == "" {
			break
		}
		child, ok := n.entries[frag]
		if !ok {
			child = &node{dir: true, entries: map[string]*node{}}
			d.bind(n, frag, child)
			d.touch(child)
		} else if !child.dir {
			return nil, pathError("mkdir", p, syscall.ENOTDIR)
		}
		n = child
	}
	return &FS{d: d, root: p}, nil
}

// List implements fs.FS.
func (f *FS) List(name string) ([]xfs.FileInfo, error) {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.lookupDir("open", p)
	if err != nil {
		return nil, err
	}
	infos := make([]xfs.FileInfo, 0, len(n.entries))
	for _, name := range slices.Sorted(maps.Keys(n.entries)) {
		infos = append(infos, n.entries[name].info())
	}
	return infos, nil
}

// Exists implements fs.FS.
func (f *FS) Exists(name string) (bool, error) {
	_, err := f.Stat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Remove implements fs.FS.
func (f *FS) Remove(name string) error {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	dir, base, err := d.parent("remove", p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, ok := dir.entries[base]; ok {
		d.unbind(dir, base)
	}
	return nil
}

// Rename implements fs.FS.
func (f *FS) Rename(oldPath, newPath string) error {
	from, to, d := f.abs(oldPath), f.abs(newPath), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	toDir, toBase, err := d.parent("rename", to)
	if err != nil {
		return err
	}
	fromDir, fromBase, err := d.parent("rename", from)
	if err != nil {
		return err
	}
	n, ok := fromDir.entries[fromBase]
	if !ok {
		return pathError("rename", from, os.ErrNotExist)
	}
	if from == to {
		return nil
	}
	if strings.HasPrefix(to, from+"/") {
		return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EINVAL}
	}
	d.unbind(fromDir, fromBase)
	d.bind(toDir, toBase, n)
	return nil
}

// Stat implements fs.FS.
func (f *FS) Stat(name string) (xfs.FileInfo, error) {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.lookup("stat", p)
	if err != nil {
		return nil, err
	}
	return n.info(), nil
}

// SyncDir implements fs.FS.
func (f *FS) SyncDir(name string) error {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.lookupDir("sync", p)
	return err
}
