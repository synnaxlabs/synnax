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
	"github.com/synnaxlabs/x/set"
)

type file struct {
	d    *disk
	n    *node
	path string
	// syncErrors is the count of failed Syncs of the file that the handle has seen.
	syncErrors uint64
	// rpos and wpos are the offsets of the next Read and the next Write.
	rpos, wpos                            int64
	readable, writable, appending, closed bool
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
	if f.d.open[f.n]--; f.d.open[f.n] == 0 {
		delete(f.d.open, f.n)
	}
	return nil
}

// Read implements fs.File.
func (f *file) Read(p []byte) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("read", f.readable, 0); err != nil {
		return 0, err
	}
	if f.rpos >= int64(len(f.n.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.n.data[f.rpos:])
	f.d.spoil(p[:n], f.n, f.path, f.rpos)
	f.rpos += int64(n)
	return n, nil
}

// ReadAt implements fs.File.
func (f *file) ReadAt(p []byte, off int64) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("read", f.readable, off); err != nil {
		return 0, err
	}
	if off >= int64(len(f.n.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.n.data[off:])
	f.d.spoil(p[:n], f.n, f.path, off)
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Write implements fs.File.
func (f *file) Write(p []byte) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("write", f.writable, 0); err != nil {
		return 0, err
	}
	if f.appending {
		f.wpos = int64(len(f.n.data))
	}
	n, err := f.d.write(f.n, p, f.wpos, f.path)
	f.wpos += int64(n)
	return n, err
}

// WriteAt implements fs.File.
func (f *file) WriteAt(p []byte, off int64) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("write", f.writable, off); err != nil {
		return 0, err
	}
	return f.d.write(f.n, p, off, f.path)
}

// Truncate implements fs.File.
func (f *file) Truncate(size int64) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("truncate", f.writable, size); err != nil {
		return err
	}
	if f.d.fit(f.n, size) < size {
		return pathError("truncate", f.path, syscall.ENOSPC)
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
		return f.d.syncDir(f.n, f.path)
	}
	f.d.sync(f.n, f.path)
	if f.syncErrors != f.n.syncErrors {
		f.syncErrors = f.n.syncErrors
		return pathError("sync", f.path, syscall.EIO)
	}
	return nil
}

// write copies p into the file n at off, and fills a gap before off with zeros. On a
// full disk it copies the bytes that fit and fails with ENOSPC.
func (d *disk) write(n *node, p []byte, off int64, path string) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	end := off + int64(len(p))
	fit := d.fit(n, end)
	if fit <= off {
		return 0, pathError("write", path, syscall.ENOSPC)
	}
	if fit > int64(len(n.data)) {
		d.resize(n, fit)
	}
	written := copy(n.data[off:fit], p)
	n.dirty.mark(off, fit)
	d.touch(n)
	if fit < end {
		return written, pathError("write", path, syscall.ENOSPC)
	}
	return written, nil
}

// fit returns size, or the longest length under size that the disk has space to give
// the file n.
func (d *disk) fit(n *node, size int64) int64 {
	if d.cfg.Capacity == 0 || size <= int64(len(n.data)) {
		return size
	}
	return min(size, int64(len(n.data))+max(d.cfg.Capacity-d.used(), 0))
}

// used returns the bytes that the disk holds: each file under the root, and each
// removed file that is still open.
func (d *disk) used() int64 {
	seen := set.Set[*node]{}
	total := d.usedUnder(d.root, seen)
	for n := range d.open {
		total += d.usedUnder(n, seen)
	}
	return total
}

// usedUnder returns the length of every file under n that is not in seen, and adds each
// node to seen.
func (d *disk) usedUnder(n *node, seen set.Set[*node]) int64 {
	if seen.Contains(n) {
		return 0
	}
	seen.Add(n)
	total := int64(len(n.data))
	for _, child := range n.entries {
		total += d.usedUnder(child, seen)
	}
	return total
}

// spoil applies the read faults to p, the bytes that a read of the file n at path and
// off returns. A read gets a fault only when no block that it covers gave one before,
// and the fault hits the first block of the read. So the next read of the same bytes is
// good.
func (d *disk) spoil(p []byte, n *node, path string, off int64) {
	if len(p) == 0 || d.cfg.ReadCorruptionRate == 0 && d.cfg.MisdirectedReadRate == 0 {
		return
	}
	first, end := off/blockSize, off+int64(len(p))
	for block := first; block*blockSize < end; block++ {
		if d.spoiled.Contains(place{path: path, block: block}) {
			return
		}
	}
	part := p[:min(end, (first+1)*blockSize)-off]
	misdirected := d.misdirect(part, n, path, off)
	if d.corrupt(part, path, off) || misdirected {
		d.spoiled.Add(place{path: path, block: first})
	}
}

// corrupt flips one bit of p, the bytes of one block that a read of path returns from
// off, at the read corruption rate. It reports if it did.
func (d *disk) corrupt(p []byte, path string, off int64) bool {
	file, at, nth := hashPath(path), uint64(off), d.next(drawRead, path, off)
	if d.draw(drawRead, file, at, nth) >= d.cfg.ReadCorruptionRate {
		return false
	}
	bit := int(d.draw(drawReadBit, file, at, nth) * float64(len(p)*8))
	p[bit/8] ^= 1 << (bit % 8)
	return true
}

// resize sets the length of the file n to size, and fills new bytes with zeros.
func (d *disk) resize(n *node, size int64) {
	old := int64(len(n.data))
	n.data = resized(n.data, size)
	n.dirty.mark(min(old, size), max(old, size))
	d.touch(n)
}
