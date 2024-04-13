// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fs

import (
	"github.com/cockroachdb/pebble/vfs"
	"io"
	"os"
	goPath "path"
	"sort"
)

type File interface {
	io.Closer
	io.Reader
	io.ReaderAt
	io.Writer
	io.WriterAt

	Stat() (os.FileInfo, error)
	Sync() error
}

const defaultPerm = 0755

type FS interface {
	Open(name string, flag int) (File, error)
	Sub(name string) (FS, error)
	List(name string) ([]os.FileInfo, error)
	Exists(name string) (bool, error)
	Remove(name string) error
	Rename(name string, newPath string) error
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

type defaultFS struct {
	perm os.FileMode
}

var Default FS = &defaultFS{perm: defaultPerm}

func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(name, flag, d.perm)
}

func (d *defaultFS) Sub(name string) (FS, error) {
	if err := os.MkdirAll(name, d.perm); err != nil {
		return nil, err
	}
	return &subFS{dir: name, FS: d}, nil
}

func (d *defaultFS) Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (d *defaultFS) List(name string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		infos[i], err = e.Info()
		if err != nil {
			return nil, err
		}
	}
	return infos, nil
}

func (d *defaultFS) Remove(name string) error {
	return os.RemoveAll(name)
}

func (d *defaultFS) Rename(name string, newName string) error {
	return os.Rename(name, newName)
}

func (d *defaultFS) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func NewMem() FS {
	return &memFS{
		FS:   vfs.NewMem(),
		perm: defaultPerm,
	}
}

type memFS struct {
	vfs.FS
	perm os.FileMode
}

func (m *memFS) Open(name string, flag int) (File, error) {
	if flag&os.O_CREATE != 0 {
		// create
		if flag&os.O_EXCL == 0 {
			return m.FS.Create(name)
		} else {
			if e, err := m.Exists(name); err != nil || e {
				if err != nil {
					return nil, err
				} else {
					return nil, nil
				}
			} else {
				return m.FS.Create(name)
			}

		}
	} else if flag&os.O_RDWR != 0 || flag&os.O_WRONLY != 0 {
		// not readonly
		return m.FS.OpenReadWrite(name)
	} else {
		// readonly
		return m.FS.Open(name)
	}
}

func (m *memFS) Sub(name string) (FS, error) {
	if err := m.FS.MkdirAll(goPath.Clean(name), m.perm); err != nil {
		return nil, err
	}
	return &subFS{dir: name, FS: m}, nil
}

func (m *memFS) Exists(name string) (bool, error) {
	_, err := m.FS.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (m *memFS) List(name string) ([]os.FileInfo, error) {
	entries, err := m.FS.List(name)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		infos[i], err = m.FS.Stat(goPath.Join(name, e))
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})

	return infos, nil
}

func (m *memFS) Remove(name string) error {
	return m.RemoveAll(name)
}

func (m *memFS) Rename(name string, newName string) error {
	return m.FS.Rename(name, newName)
}

func (m *memFS) Stat(name string) (os.FileInfo, error) {
	return m.FS.Stat(name)
}
