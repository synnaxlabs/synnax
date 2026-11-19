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
	"github.com/synnaxlabs/x/set"
)

// disk is the state that an FS shares with every FS from its Sub.
type disk struct {
	root *node
	// pending holds the changes to directory entries that are not durable, in order.
	pending []*change
	// calls counts the calls of each kind at each path and offset. The count is a part
	// of each draw, so a second call gets its own draw.
	calls map[call]uint64
	// spoiled holds each place whose last read got a read fault.
	spoiled set.Set[place]
	// open counts the handles on each node. A removed file holds its space until it has
	// none.
	open map[*node]int
	cfg  Config
	// clock counts the changes to the disk. It gives each one a later mod time.
	clock int64
	// mu guards every field of the disk, its nodes, and its open files.
	mu sync.Mutex
}

type node struct {
	modTime time.Time
	// entries and data are what a call sees. durableEntries and durable are what a
	// crash keeps.
	entries, durableEntries map[string]*node
	data, durable           []byte
	// dirty holds each block whose bytes or length changed since the last Sync.
	dirty blocks
	// syncErrors counts the failed Syncs of the file. Each handle reports a count that
	// it has not seen, once.
	syncErrors uint64
	dir        bool
}

func newDir() *node {
	return &node{
		dir:            true,
		entries:        map[string]*node{},
		durableEntries: map[string]*node{},
	}
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

func (n *node) info(name string) info {
	return info{name: name, size: int64(len(n.data)), modTime: n.modTime, dir: n.dir}
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
	if err != nil {
		return nil, "", err
	}
	return n, base, nil
}

// change is one change to directory entries that is not durable. A create or a remove
// has one half, and a rename has two.
type change struct {
	halves []half
	// key fixes the draw that decides if the change survives a crash.
	key uint64
}

// half is the change to one entry: name in dir held from before, and holds to after. A
// nil node is no entry.
type half struct {
	dir, from, to *node
	name          string
}

// place is an offset in the file at a path.
type place struct {
	path string
	off  int64
}

type call struct {
	path string
	kind uint64
	off  int64
}

func newDisk(cfg Config, root *node, clock int64) *disk {
	return &disk{
		cfg:     cfg,
		root:    root,
		clock:   clock,
		calls:   map[call]uint64{},
		open:    map[*node]int{},
		spoiled: set.Set[place]{},
	}
}

// draw is a draw with the run key of the disk.
func (d *disk) draw(parts ...uint64) float64 { return draw(d.cfg.RunKey, parts...) }

// next counts a call of kind at the path p and the offset off, and returns how many
// came before it.
func (d *disk) next(kind uint64, p string, off int64) uint64 {
	c := call{path: p, kind: kind, off: off}
	n := d.calls[c]
	d.calls[c] = n + 1
	return n
}

// set binds name in dir to n, or to no entry when n is nil.
func (d *disk) set(dir *node, name string, n *node) half {
	h := half{dir: dir, name: name, from: dir.entries[name], to: n}
	if n == nil {
		delete(dir.entries, name)
	} else {
		dir.entries[name] = n
	}
	d.touch(dir)
	return h
}

// record adds the change at the path p to the changes that are not durable.
func (d *disk) record(p string, halves ...half) {
	key := mix(hashPath(p) ^ d.next(drawEntry, p, 0))
	d.pending = append(d.pending, &change{halves: halves, key: key})
}
