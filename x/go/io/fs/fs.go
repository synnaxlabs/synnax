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
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/synnaxlabs/x/errors"
)

type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Writer
	io.WriterAt

	// Truncate resizes the file to the given size in bytes. It is important to note
	// that the file offset is NOT changed: i.e. if the file handle was used to write 10
	// bytes and the file is then truncated to 5 bytes, the file offset will still be at
	// 10 bytes, rendering the next read to EOF and write to leave null bytes between.
	// This is the default behavior of the os.File implementation, and the memFS
	// implementation has been adapted to match this behavior.
	//
	// In addition, the Windows implementation of FS allows a file without write
	// permissions to be truncated whereas Unix and our memFS implementation do not.
	Truncate(int64) error
	Stat() (FileInfo, error)
	Sync() error
	// Chmod changes the permission bits of the file to those of mode.
	Chmod(mode fs.FileMode) error
}

type FileInfo = fs.FileInfo

const defaultPerm = UserRWX | GroupRX | OtherRX

type FS interface {
	// Open opens a file according to the provided flag. The provided flag can be OR-ed
	// in out of the flags in os, e.g. os.O_CREATE|os.O_WRONLY.
	Open(name string, flag int) (File, error)
	// Sub returns a new FS rooted at the given directory. It creates the directory and
	// its parents when they do not exist.
	Sub(string) (FS, error)
	// List returns a list of files in the directory SORTED by file name.
	List(string) ([]FileInfo, error)
	// Exists returns true if the file exists, false otherwise.
	Exists(string) (bool, error)
	// Remove removes a file or directory recursively. It returns nil if the name does
	// not exist.
	Remove(string) error
	// Rename renames a file or directory. It returns an error if the target does not
	// exist.
	Rename(oldPath, newPath string) error
	// Stat returns a FileInfo interface.
	Stat(string) (FileInfo, error)
}

type subFS struct {
	FS
	dir string
}

func (s *subFS) Open(name string, flag int) (File, error) {
	return s.FS.Open(path.Join(s.dir, name), flag)
}

func (s *subFS) Sub(name string) (FS, error) { return s.FS.Sub(path.Join(s.dir, name)) }

func (s *subFS) Exists(name string) (bool, error) {
	return s.FS.Exists(path.Join(s.dir, name))
}

func (s *subFS) List(name string) ([]FileInfo, error) {
	return s.FS.List(path.Join(s.dir, name))
}

func (s *subFS) Remove(name string) error { return s.FS.Remove(path.Join(s.dir, name)) }

func (s *subFS) Rename(oldName, newName string) error {
	return s.FS.Rename(path.Join(s.dir, oldName), path.Join(s.dir, newName))
}

func (s *subFS) Stat(name string) (FileInfo, error) {
	return s.FS.Stat(path.Join(s.dir, name))
}

// CopyFile copies the file name in src to the same name in dst, replaces any file
// there, and syncs the copy. The copy gets the permission bits of the source before
// CopyFile writes data to it. The directory of name must exist in dst.
func CopyFile(src, dst FS, name string) (err error) {
	from, err := src.Open(name, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, from.Close()) }()
	info, err := from.Stat()
	if err != nil {
		return err
	}
	to, err := dst.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, to.Close()) }()
	if err = to.Chmod(info.Mode().Perm()); err != nil {
		return err
	}
	if _, err = io.Copy(to, from); err != nil {
		return err
	}
	return to.Sync()
}

// Move moves the file or directory name in src to the same name in dst. src and dst
// must be one FS, or come from Sub calls on one FS. It returns errors.ErrUnsupported
// otherwise.
func Move(src, dst FS, name string) error {
	srcRoot, srcPath := resolve(src, name)
	dstRoot, dstPath := resolve(dst, name)
	if srcRoot != dstRoot {
		return errors.Wrap(
			errors.ErrUnsupported,
			"cannot move between different file systems",
		)
	}
	return srcRoot.Rename(srcPath, dstPath)
}

func resolve(f FS, name string) (FS, string) {
	if sub, ok := f.(*subFS); ok {
		return sub.FS, path.Join(sub.dir, name)
	}
	return f, name
}
