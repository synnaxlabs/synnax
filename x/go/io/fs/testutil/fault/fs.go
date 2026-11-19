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
	"github.com/synnaxlabs/x/validate"
)

// FS is a disk in memory. Only a Sync makes the bytes of a file durable, and only a
// SyncDir makes the entries of a directory durable. A rename between two directories is
// one change: a SyncDir of either makes it durable in both. A node in a directory that
// is not durable itself is lost. Crash returns what a power loss keeps. It is safe for
// concurrent use, and each call is atomic.
type FS struct {
	d *disk
	// root is the cleaned absolute path of the directory that names resolve against.
	root string
}

var _ xfs.FS = (*FS)(nil)

// Config sets the faults of an FS. The zero value is a disk with no faults.
type Config struct {
	// RunKey fixes every random choice of the FS.
	RunKey uint64
	// SyncErrorRate is the chance, from 0 to 1, that a Sync or a SyncDir with changes
	// to make durable fails with EIO. A failed Sync drops its changes as Linux does:
	// its blocks are clean again, so a later Sync writes one of them only after a new
	// write to that block. Each handle that is open on the file gets the error from its
	// next Sync, once. A failed SyncDir keeps its changes for the next SyncDir.
	SyncErrorRate float64
	// EvictionRate is the chance, from 0 to 1, that a block dropped by a failed Sync
	// reads its durable bytes from then on. Otherwise the block reads its new bytes
	// until a crash.
	EvictionRate float64
	// ReadCorruptionRate is the chance, from 0 to 1, that a read returns one flipped
	// bit. The bytes on the disk do not change. A read fault of either kind hits the
	// first 4 KiB block of a read, and only when no block of the read gave a fault
	// before. So the next read of the same bytes is good.
	ReadCorruptionRate float64
	// MisdirectedReadRate is the chance, from 0 to 1, that a read gets one 4 KiB block
	// from the wrong place: another block of the file, or a block of another file. The
	// bytes on the disk do not change. The wrong block is one that the disk holds at
	// the read, so a write to another file can change the bytes that come back, but not
	// the read and the block that the fault hits.
	MisdirectedReadRate float64
	// Capacity is the size of the disk in bytes, which the lengths of its files fill. A
	// removed file fills it until its last handle closes. Zero is no limit. A write
	// past it writes the bytes that fit and fails with ENOSPC.
	Capacity int64
}

func (c Config) validate() error {
	v := validate.New("fault.Config")
	validateRate(v, "sync_error_rate", c.SyncErrorRate)
	validateRate(v, "eviction_rate", c.EvictionRate)
	validateRate(v, "read_corruption_rate", c.ReadCorruptionRate)
	validateRate(v, "misdirected_read_rate", c.MisdirectedReadRate)
	v.GreaterThanEq("capacity", c.Capacity, 0)
	return v.Error()
}

// New returns an empty FS with the faults of cfg. The error matches
// validate.ErrValidation when a rate of cfg is not from 0 to 1, or its capacity is
// under zero.
func New(cfg Config) (*FS, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &FS{d: newDisk(cfg, newDir(), 0, 0), root: "/"}, nil
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
		if n.dir && flag&(os.O_WRONLY|os.O_RDWR) != 0 {
			return nil, pathError("open", p, syscall.EISDIR)
		}
		if flag&os.O_TRUNC != 0 && !n.dir {
			d.resize(n, 0)
		}
	case errors.Is(err, os.ErrNotExist) && flag&os.O_CREATE != 0:
		dir, base, err := d.parent("open", p)
		if err != nil {
			return nil, err
		}
		d.ids++
		n = &node{id: d.ids}
		d.record(p, d.set(dir, base, n))
		d.touch(n)
	default:
		return nil, err
	}
	d.open[n]++
	return &file{
		d:          d,
		n:          n,
		path:       p,
		readable:   flag&os.O_WRONLY == 0,
		writable:   flag&(os.O_WRONLY|os.O_RDWR) != 0,
		appending:  flag&os.O_APPEND != 0,
		syncErrors: n.syncErrors,
	}, nil
}

// Sub implements fs.FS.
func (f *FS) Sub(name string) (xfs.FS, error) {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n, at := d.root, ""
	for frag := range strings.SplitSeq(strings.TrimPrefix(p, "/"), "/") {
		if frag == "" {
			break
		}
		at += "/" + frag
		child, ok := n.entries[frag]
		if !ok {
			child = newDir()
			d.record(at, d.set(n, frag, child))
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
		infos = append(infos, n.entries[name].info(name))
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
		d.record(p, d.set(dir, base, nil))
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
	d.record(from, d.set(fromDir, fromBase, nil), d.set(toDir, toBase, n))
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
	return n.info(path.Base(p)), nil
}

// SyncDir implements fs.FS.
func (f *FS) SyncDir(name string) error {
	p, d := f.abs(name), f.d
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.lookupDir("sync", p)
	if err != nil {
		return err
	}
	return d.syncDir(n, p)
}
