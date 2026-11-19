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
	f.d.corrupt(p[:n], f.path, f.rpos)
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
	f.d.corrupt(p[:n], f.path, off)
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
	n, err := f.d.write(f.n, p, f.wpos, f.path)
	f.wpos += int64(n)
	return n, err
}

// WriteAt implements fs.File.
func (f *file) WriteAt(p []byte, off int64) (int, error) {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("write", f.write, off); err != nil {
		return 0, err
	}
	return f.d.write(f.n, p, off, f.path)
}

// Truncate implements fs.File.
func (f *file) Truncate(size int64) error {
	f.d.mu.Lock()
	defer f.d.mu.Unlock()
	if err := f.check("truncate", f.write, size); err != nil {
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
	return f.d.sync(f.n, f.path)
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
	free := max(d.cfg.Capacity-d.used(d.root, set.Set[*node]{}), 0)
	return min(size, int64(len(n.data))+free)
}

// used returns the length of every file under the directory n that is not in seen, and
// adds each node to seen.
func (d *disk) used(n *node, seen set.Set[*node]) int64 {
	if seen.Contains(n) {
		return 0
	}
	seen.Add(n)
	total := int64(len(n.data))
	for _, child := range n.entries {
		total += d.used(child, seen)
	}
	return total
}

// corrupt flips one bit of p, the bytes that a read of path at off returns, at the read
// corruption rate.
func (d *disk) corrupt(p []byte, path string, off int64) {
	if d.cfg.ReadCorruptionRate == 0 || len(p) == 0 {
		return
	}
	file, nth := hashPath(path), d.next(drawRead, path, off)
	if draw(d.cfg.RunKey, drawRead, file, uint64(off), nth) < d.cfg.ReadCorruptionRate {
		bit := draw(
			d.cfg.RunKey,
			drawReadBit,
			file,
			uint64(off),
			nth,
		) * float64(
			len(p)*8,
		)
		p[int(bit)/8] ^= 1 << (int(bit) % 8)
	}
}

// resize sets the length of the file n to size, and fills new bytes with zeros.
func (d *disk) resize(n *node, size int64) {
	old := int64(len(n.data))
	n.data = resized(n.data, size)
	n.dirty.mark(min(old, size), max(old, size))
	d.touch(n)
}
