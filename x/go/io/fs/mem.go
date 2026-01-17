// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs

import (
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cockroachdb/errors/oserror"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs/internal/invariants"
)

const sep = "/"

// NewMem returns a new memory-backed FS implementation.
func NewMem() *MemFS { return &MemFS{root: newRootMemNode()} }

// MemFS implements FS.
type MemFS struct {
	mu   sync.Mutex
	root *memNode
}

var _ FS = &MemFS{}

// Open implements FS.Open.
func (y *MemFS) Open(fullName string, flag int) (File, error) {
	var ret *memFile
	if err := y.walk(fullName, func(dir *memNode, frag string, final bool) error {
		if !final {
			return nil
		}
		if frag == "" {
			return errors.New("memfs: empty file name")
		}
		if n := dir.children[frag]; n != nil {
			// file exists
			if flag&os.O_CREATE != 0 && flag&os.O_EXCL != 0 {
				return errors.New("memfs: file already exists with O_EXCL")
			}

			if flag&os.O_TRUNC != 0 {
				n.mu.Lock()
				n.mu.data = n.mu.data[:0]
				n.mu.modTime = time.Now()
				n.mu.Unlock()
			}

			writeOffset := 0
			if flag&os.O_APPEND != 0 {
				writeOffset = len(n.mu.data)
			}

			ret = &memFile{
				n:     n,
				fs:    y,
				read:  flag&os.O_WRONLY == 0,
				write: flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0,
				wpos:  writeOffset,
			}
			return nil
		}
		// file does not exist
		if flag&os.O_CREATE == 0 {
			return &os.PathError{Op: "open", Path: fullName, Err: oserror.ErrNotExist}
		}

		node := &memNode{name: frag}
		dir.children[frag] = node
		ret = &memFile{
			n:     node,
			fs:    y,
			read:  flag&os.O_WRONLY == 0,
			write: flag&os.O_WRONLY != 0 || flag&os.O_RDWR != 0,
		}
		return nil
	}); err != nil {
		return nil, err
	}
	ret.n.refs.Add(1)
	return ret, nil
}

// Sub implements FS.Sub.
func (y *MemFS) Sub(name string) (FS, error) {
	if err := y.walk(name, func(dir *memNode, frag string, final bool) error {
		if frag == "" {
			if final {
				return nil
			}
			return errors.New("memfs: empty file name")
		}
		child := dir.children[frag]
		if child == nil {
			dir.children[frag] = &memNode{
				name:     frag,
				children: make(map[string]*memNode),
				isDir:    true,
			}
			return nil
		}
		if !child.isDir {
			return &os.PathError{
				Op:   "open",
				Path: name,
				Err:  errors.New("not a directory"),
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &subFS{dir: name, FS: y}, nil
}

// Exists implements FS.Exists.
func (y *MemFS) Exists(name string) (bool, error) {
	_, err := y.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Remove implements FS.Remove
func (y *MemFS) Remove(fullName string) error {
	if err := y.walk(fullName, func(dir *memNode, frag string, final bool) error {
		if !final {
			return nil
		}
		if frag == "" {
			return errors.New("memfs: empty file name")
		}
		if _, ok := dir.children[frag]; !ok {
			return nil
		}
		delete(dir.children, frag)
		return nil
	}); err != nil &&
		// Match os.RemoveAll which returns a nil error even if the parent directories
		// don't exist.
		!oserror.IsNotExist(err) {
		return err
	}
	return nil
}

// Rename implements FS.Rename.
func (y *MemFS) Rename(oldName, newName string) error {
	var n *memNode
	if err := y.walk(oldName, func(dir *memNode, frag string, final bool) error {
		if !final {
			return nil
		}
		if frag == "" {
			return errors.New("memfs: empty file name")
		}
		n = dir.children[frag]
		delete(dir.children, frag)
		return nil
	}); err != nil {
		return err
	}
	if n == nil {
		return &os.PathError{Op: "open", Path: oldName, Err: oserror.ErrNotExist}
	}
	return y.walk(newName, func(dir *memNode, frag string, final bool) error {
		if !final {
			return nil
		}
		if frag == "" {
			return errors.New("memfs: empty file name")
		}
		dir.children[frag] = n
		n.name = frag
		return nil
	})
}

// List returns a list of files in the directory sorted by file name.
func (y *MemFS) List(dirname string) ([]os.FileInfo, error) {
	if !strings.HasSuffix(dirname, sep) {
		dirname += sep
	}
	var ret []os.FileInfo
	if err := y.walk(dirname, func(dir *memNode, frag string, final bool) error {
		if !final {
			return nil
		}
		if frag != "" {
			panic("unreachable")
		}
		ret = make([]os.FileInfo, 0, len(dir.children))
		for _, s := range dir.children {
			ret = append(ret, s)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].Name() < ret[j].Name() })
	return ret, nil
}

// Stat implements FS.Stat.
func (y *MemFS) Stat(name string) (os.FileInfo, error) {
	f, err := y.Open(name, 0)
	if err != nil {
		var pe *os.PathError
		if errors.As(err, &pe) {
			pe.Op = "stat"
		}
		return nil, err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	var stats os.FileInfo
	stats, err = f.Stat()
	return stats, err
}

// memNode holds a file's data or a directory's children, and implements os.FileInfo.
type memNode struct {
	name  string
	isDir bool
	refs  atomic.Int32

	// Mutable state.
	//
	// - For a file: data, syncedDate, modTime. A file is only being mutated by a single
	//   goroutine, but there can be concurrent readers e.g. DB.Checkpoint() which can
	//   read WAL or MANIFEST files that are being written to. Additionally Sync() calls
	//   can be concurrent with writing.
	// - For a directory: children and syncedChildren. Concurrent writes are possible,
	//   and these are protected using MemFS.mu.
	mu struct {
		sync.Mutex
		data       []byte
		syncedData []byte
		modTime    time.Time
	}

	children       map[string]*memNode
	syncedChildren map[string]*memNode
}

func newRootMemNode() *memNode {
	return &memNode{name: sep, children: make(map[string]*memNode), isDir: true}
}

func (f *memNode) IsDir() bool { return f.isDir }

func (f *memNode) ModTime() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mu.modTime
}

const standardPerm = UserRWX | GroupRX | OtherRX

func (f *memNode) Mode() os.FileMode {
	if f.isDir {
		return os.ModeDir | standardPerm
	}
	return standardPerm
}

func (f *memNode) Name() string { return f.name }

func (f *memNode) Size() int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return int64(len(f.mu.data))
}

func (f *memNode) Sys() any { return nil }

// walk walks the directory tree for the fullname, calling f at each step. If f returns
// an error, the walk will be aborted and return that same error.
//
// Each walk is atomic: y's mutex is held for the entire operation, including all calls
// to f.
//
// dir is the directory at that step, frag is the name fragment, and final is whether it
// is the final step. For example, walking "/foo/bar/x" will result in 3 calls to f:
//   - "/", "foo", false
//   - "/foo/", "bar", false
//   - "/foo/bar/", "x", true
//
// Similarly, walking "/y/z/", with a trailing slash, will result in 3 calls to f:
//   - "/", "y", false
//   - "/y/", "z", false
//   - "/y/z/", "", true
func (y *MemFS) walk(fullname string, f func(*memNode, string, bool) error) error {
	y.mu.Lock()
	defer y.mu.Unlock()

	// For memfs, the current working directory is the same as the root directory, so we
	// strip off any leading "/"s to make fullname a relative path, and the walk starts
	// at y.root.
	for len(fullname) > 0 && fullname[0] == sep[0] {
		fullname = fullname[1:]
	}
	dir := y.root

	for {
		frag, remaining := fullname, ""
		i := strings.IndexRune(fullname, rune(sep[0]))
		final := i < 0
		if !final {
			frag, remaining = fullname[:i], fullname[i+1:]
			for len(remaining) > 0 && remaining[0] == sep[0] {
				remaining = remaining[1:]
			}
		}
		if err := f(dir, frag, final); err != nil {
			return err
		}
		if final {
			break
		}
		child := dir.children[frag]
		if child == nil {
			return &os.PathError{Op: "open", Path: fullname, Err: oserror.ErrNotExist}
		}
		if !child.isDir {
			return &os.PathError{
				Op:   "open",
				Path: fullname,
				Err:  errors.New("memfs: not a directory"),
			}
		}
		dir, fullname = child, remaining
	}
	return nil
}

// memFile is a reader or writer of a node's data, and implements File.
type memFile struct {
	n           *memNode
	fs          *MemFS // nil for a standalone memFile
	rpos        int
	wpos        int
	read, write bool
}

var _ File = (*memFile)(nil)

func (f *memFile) Close() error {
	if n := f.n.refs.Add(-1); n < 0 {
		panic(fmt.Sprintf("memfs: close of unopened file: %d", n))
	}
	f.n = nil
	return nil
}

func (f *memFile) Read(p []byte) (int, error) {
	if !f.read {
		return 0, errors.New("memfs: file was not opened for reading")
	}
	if f.n.isDir {
		return 0, errors.New("memfs: cannot read a directory")
	}
	f.n.mu.Lock()
	defer f.n.mu.Unlock()
	if f.rpos >= len(f.n.mu.data) {
		return 0, io.EOF
	}
	n := copy(p, f.n.mu.data[f.rpos:])
	f.rpos += n
	return n, nil
}

func (f *memFile) ReadAt(p []byte, off int64) (int, error) {
	if !f.read {
		return 0, errors.New("memfs: file was not opened for reading")
	}
	if f.n.isDir {
		return 0, errors.New("memfs: cannot read a directory")
	}
	f.n.mu.Lock()
	defer f.n.mu.Unlock()
	if off >= int64(len(f.n.mu.data)) {
		return 0, io.EOF
	}
	n := copy(p, f.n.mu.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func (f *memFile) Write(p []byte) (int, error) {
	if !f.write {
		return 0, invariants.ErrAccessDenied
	}
	if f.n.isDir {
		return 0, errors.New("memfs: cannot write a directory")
	}
	f.n.mu.Lock()
	defer f.n.mu.Unlock()
	f.n.mu.modTime = time.Now()
	if f.wpos+len(p) <= len(f.n.mu.data) {
		n := copy(f.n.mu.data[f.wpos:f.wpos+len(p)], p)
		if n != len(p) {
			panic("stuff")
		}
	} else if f.wpos > len(f.n.mu.data) {
		zeros := make([]byte, f.wpos-len(f.n.mu.data))
		f.n.mu.data = append(append(f.n.mu.data, zeros...), p...)
	} else {
		f.n.mu.data = append(f.n.mu.data[:f.wpos], p...)
	}
	f.wpos += len(p)

	if invariants.Enabled {
		// Mutate the input buffer to flush out bugs in Pebble which expect the input
		// buffer to be unmodified.
		for i := range p {
			p[i] ^= 0xff
		}
	}
	return len(p), nil
}

func (f *memFile) WriteAt(p []byte, ofs int64) (int, error) {
	if !f.write {
		return 0, invariants.ErrAccessDenied
	}
	if f.n.isDir {
		return 0, errors.New("memfs: cannot write a directory")
	}
	f.n.mu.Lock()
	defer f.n.mu.Unlock()
	f.n.mu.modTime = time.Now()

	for len(f.n.mu.data) < int(ofs)+len(p) {
		f.n.mu.data = append(f.n.mu.data, 0)
	}

	n := copy(f.n.mu.data[int(ofs):int(ofs)+len(p)], p)
	if n != len(p) {
		panic(fmt.Sprintf(
			"expected to copy %d bytes into memfile, only copied %d",
			len(p),
			n,
		))
	}

	return len(p), nil
}

// Truncate resizes the file to the given size in bytes. It is important to note that
// the file offset is NOT changed: i.e. if the file handle was used to write 10 bytes
// and the file is then truncated to 5 bytes, the file offset will still be at 10 bytes,
// rendering the next read to EOF and write to leave null bytes between. This is the
// default behavior of the os.File implementation, and the memFS implementation has been
// adapted to match this behavior.
func (f *memFile) Truncate(size int64) error {
	if !f.write {
		return errors.New(
			"memfs: file was not created for writing (truncate requires write fd)",
		)
	}
	if f.n.isDir {
		return errors.New("memfs: cannot truncate a directory")
	}
	if size < 0 {
		return errors.New("memfs: truncate size must be non-negative")
	}

	f.n.mu.Lock()
	defer f.n.mu.Unlock()
	f.n.mu.modTime = time.Now()

	if size > int64(len(f.n.mu.data)) {
		f.n.mu.data = append(f.n.mu.data, make([]byte, size-int64(len(f.n.mu.data)))...)
	} else {
		f.n.mu.data = f.n.mu.data[:size]
	}

	return nil
}

func (f *memFile) Stat() (os.FileInfo, error) { return f.n, nil }

func (f *memFile) Sync() error {
	if f.fs == nil {
		return nil
	}
	f.fs.mu.Lock()
	defer f.fs.mu.Unlock()
	if f.n.isDir {
		f.n.syncedChildren = make(map[string]*memNode)
		maps.Copy(f.n.syncedChildren, f.n.children)
		return nil
	}
	f.n.mu.Lock()
	f.n.mu.syncedData = slices.Clone(f.n.mu.data)
	f.n.mu.Unlock()
	return nil
}
