// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fault

import (
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	xfs "github.com/synnaxlabs/x/io/fs"
)

// disk is the state that an FS shares with every FS from its Sub.
type disk struct {
	root *node
	// clock counts the changes to the disk. It gives each one a later mod time.
	clock int64
	// mu guards every field of the disk, its nodes, and its open files.
	mu sync.Mutex
}

type node struct {
	modTime time.Time
	entries map[string]*node
	name    string
	data    []byte
	dir     bool
}

type info struct {
	modTime time.Time
	name    string
	size    int64
	dir     bool
}

var _ xfs.FileInfo = info{}

func (i info) Name() string       { return i.name }
func (i info) Size() int64        { return i.size }
func (i info) ModTime() time.Time { return i.modTime }
func (i info) IsDir() bool        { return i.dir }
func (i info) Sys() any           { return nil }

func (i info) Mode() os.FileMode {
	mode := xfs.UserRWX | xfs.GroupRX | xfs.OtherRX
	if i.dir {
		mode |= os.ModeDir
	}
	return mode
}

func (n *node) info() info {
	return info{name: n.name, size: int64(len(n.data)), modTime: n.modTime, dir: n.dir}
}

func (d *disk) touch(n *node) {
	d.clock++
	n.modTime = time.Unix(0, d.clock)
}

func pathError(op, p string, err error) error {
	return &os.PathError{Op: op, Path: p, Err: err}
}

// lookup returns the node at the cleaned absolute path p.
func (d *disk) lookup(op, p string) (*node, error) {
	n := d.root
	if p == "/" {
		return n, nil
	}
	for frag := range strings.SplitSeq(p[1:], "/") {
		if !n.dir {
			return nil, pathError(op, p, syscall.ENOTDIR)
		}
		child, ok := n.entries[frag]
		if !ok {
			return nil, pathError(op, p, os.ErrNotExist)
		}
		n = child
	}
	return n, nil
}

// lookupDir is lookup for a path that must be a directory.
func (d *disk) lookupDir(op, p string) (*node, error) {
	n, err := d.lookup(op, p)
	if err != nil {
		return nil, err
	}
	if !n.dir {
		return nil, pathError(op, p, syscall.ENOTDIR)
	}
	return n, nil
}

// parent returns the directory that holds the cleaned absolute path p, and the last
// element of p.
func (d *disk) parent(op, p string) (*node, string, error) {
	if p == "/" {
		return nil, "", pathError(op, p, syscall.EINVAL)
	}
	dir, base := path.Split(p)
	n, err := d.lookupDir(op, path.Clean(dir))
	return n, base, err
}

func (d *disk) bind(dir *node, name string, n *node) {
	dir.entries[name] = n
	n.name = name
	d.touch(dir)
}

func (d *disk) unbind(dir *node, name string) {
	delete(dir.entries, name)
	d.touch(dir)
}
