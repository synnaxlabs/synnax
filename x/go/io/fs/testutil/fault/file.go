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
	"io"
	"os"
	"path"
	"syscall"

	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/internal/invariants"
)

type file struct {
	d    *disk
	n    *node
	path string
	// rpos and wpos are the offsets of the next Read and the next Write.
	rpos, wpos                  int64
	read, write, append, closed bool
}

var _ xfs.File = (*file)(nil)

// check returns the error that stops a call on f. The call needs allowed to be true and
// takes the offset or size off.
func (f *file) check(op string, allowed bool, off int64) error {
	switch {
	case f.closed:
		return pathError(op, f.path, os.ErrClosed)
	case !allowed:
		return pathError(op, f.path, invariants.ErrAccessDenied)
	case off < 0:
		return pathError(op, f.path, syscall.EINVAL)
	case f.n.dir:
		return pathError(op, f.path, syscall.EISDIR)
	}
	return nil
}

// Close implements fs.File.
func (f *file) Close() error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if f.closed {
		return pathError("close", f.path, os.ErrClosed)
	}
	f.closed = true
	return nil
}

// Read implements fs.File.
func (f *file) Read(p []byte) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("read", f.read, 0); err != nil {
		return 0, err
	}
	if f.rpos >= int64(len(f.n.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.n.data[f.rpos:])
	f.rpos += int64(n)
	return n, nil
}

// ReadAt implements fs.File.
func (f *file) ReadAt(p []byte, off int64) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("read", f.read, off); err != nil {
		return 0, err
	}
	if off >= int64(len(f.n.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.n.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Write implements fs.File.
func (f *file) Write(p []byte) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("write", f.write, 0); err != nil {
		return 0, err
	}
	if f.append {
		f.wpos = int64(len(f.n.data))
	}
	f.d.write(f.n, p, f.wpos)
	f.wpos += int64(len(p))
	return len(p), nil
}

// WriteAt implements fs.File.
func (f *file) WriteAt(p []byte, off int64) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("write", f.write, off); err != nil {
		return 0, err
	}
	f.d.write(f.n, p, off)
	return len(p), nil
}

// Truncate implements fs.File.
func (f *file) Truncate(size int64) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("truncate", f.write, size); err != nil {
		return err
	}
	f.d.resize(f.n, size)
	return nil
}

// Stat implements fs.File.
func (f *file) Stat() (xfs.FileInfo, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if f.closed {
		return nil, pathError("stat", f.path, os.ErrClosed)
	}
	return f.n.info(path.Base(f.path)), nil
}

// Sync implements fs.File.
func (f *file) Sync() error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if f.closed {
		return pathError("sync", f.path, os.ErrClosed)
	}
	if f.n.dir {
		f.d.syncDir(f.n)
	} else {
		f.d.sync(f.n)
	}
	return nil
}

// write copies p into the file n at off, and fills a gap before off with zeros.
func (d *disk) write(n *node, p []byte, off int64) {
	if end := off + int64(len(p)); end > int64(len(n.data)) {
		d.resize(n, end)
	}
	copy(n.data[off:], p)
	n.dirty.mark(off, off+int64(len(p)))
	d.touch(n)
}

// resize sets the length of the file n to size, and fills new bytes with zeros.
func (d *disk) resize(n *node, size int64) {
	old := int64(len(n.data))
	n.data = resized(n.data, size)
	n.dirty.mark(min(old, size), max(old, size))
	d.touch(n)
}
