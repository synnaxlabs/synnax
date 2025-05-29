// Copyright 2025 Synnax Labs, Inc.
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
	"os"
	goPath "path"
)

type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Writer
	io.WriterAt

	// Truncate resizes the file to the given size in bytes. It is important to note that
	// the file offset is NOT changed: i.e. if the file handle was used to write 10 bytes
	// and the file is then truncated to 5 bytes, the file offset will still be at 10 bytes,
	// rendering the next read to EOF and write to leave null bytes between.
	// This is the default behavior of the os.File implementation, and the memFS
	// implementation has been adapted to match this behaviour.
	//
	// In addition, the Windows implementation of FS allows a file without write
	// permissions to be truncated whereas Unix and our memFS implementation do not.
	Truncate(size int64) error
	Stat() (os.FileInfo, error)
	Sync() error
}

const defaultPerm = 0755

type FS interface {
	// Open opens a file according to the provided flag. The provided flag can be
	// OR-ed in out of the flags in os, e.g. os.O_CREATE|os.O_WRONLY.
	Open(name string, flag int) (File, error)
	// Sub returns a new FS rooted at the given directory.
	Sub(name string) (FS, error)
	// List returns a list of files in the directory SORTED by file name.
	List(name string) ([]os.FileInfo, error)
	// Exists returns true if the file exists, false otherwise.
	Exists(name string) (bool, error)
	// Remove removes a file or directory recursively. It returns nil if the name does
	// not exist.
	Remove(name string) error
	// Rename renames a file or directory. It returns an error if the target does not exist.
	Rename(name string, newPath string) error
	// Stat returns a FileInfo interface.
	Stat(name string) (os.FileInfo, error)
}

type subFS struct {
	dir string
	FS
}

func (s *subFS) Open(name string, flag int) (File, error) {
	return s.FS.Open(goPath.Join(s.dir, name), flag)
}

func (s *subFS) Sub(name string) (FS, error) {
	return s.FS.Sub(goPath.Join(s.dir, name))
}

func (s *subFS) Exists(name string) (bool, error) {
	return s.FS.Exists(goPath.Join(s.dir, name))
}

func (s *subFS) List(name string) ([]os.FileInfo, error) {
	return s.FS.List(goPath.Join(s.dir, name))
}

func (s *subFS) Remove(name string) error {
	return s.FS.Remove(goPath.Join(s.dir, name))
}

func (s *subFS) Rename(oldName string, newName string) error {
	return s.FS.Rename(goPath.Join(s.dir, oldName), goPath.Join(s.dir, newName))
}

func (s *subFS) Stat(name string) (os.FileInfo, error) {
	return s.FS.Stat(goPath.Join(s.dir, name))
}
