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
	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble/vfs"
	"io"
	"os"
	"path"
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
	Open(pth string, flag int) (File, error)
	Sub(pth string) (FS, error)
	List(pth string) ([]os.FileInfo, error)
	Exists(pth string) (bool, error)
	Remove(pth string) error
	Rename(pth string, newPth string) error
	Stat(pth string) (os.FileInfo, error)
}

type subFS struct {
	dir string
	FS
}

func (s *subFS) Open(name string, flag int) (File, error) {
	return s.FS.Open(path.Join(s.dir, name), flag)
}

func (s *subFS) Sub(name string) (FS, error) {
	return s.FS.Sub(path.Join(s.dir, name))
}

func (s *subFS) Exists(name string) (bool, error) {
	return s.FS.Exists(path.Join(s.dir, name))
}

func (s *subFS) List(name string) ([]os.FileInfo, error) {
	return s.FS.List(path.Join(s.dir, name))
}

func (s *subFS) Remove(name string) error {
	return s.FS.Remove(path.Join(s.dir, name))
}

func (s *subFS) Rename(oldName string, newName string) error {
	return s.FS.Rename(path.Join(s.dir, oldName), path.Join(s.dir, newName))
}

func (s *subFS) Stat(name string) (os.FileInfo, error) {
	return s.FS.Stat(path.Join(s.dir, name))
}

type defaultFS struct {
	perm os.FileMode
}

var Default FS = &defaultFS{perm: defaultPerm}

func (d *defaultFS) Open(pth string, flag int) (File, error) {
	return os.OpenFile(pth, flag, d.perm)
}

func (d *defaultFS) Sub(pth string) (FS, error) {
	if err := os.MkdirAll(pth, d.perm); err != nil {
		return nil, err
	}
	return &subFS{dir: pth, FS: d}, nil
}

func (d *defaultFS) Exists(pth string) (bool, error) {
	_, err := os.Stat(pth)
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (d *defaultFS) List(pth string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(pth)
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

func (d *defaultFS) Remove(pth string) error {
	if e, err := d.Exists(pth); err != nil || !e {
		return errors.New("Invalid file path")
	}
	return os.RemoveAll(pth)
}

func (d *defaultFS) Rename(pth string, newPth string) error {
	return os.Rename(pth, newPth)
}

func (d *defaultFS) Stat(pth string) (os.FileInfo, error) {
	return os.Stat(pth)
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

func (m *memFS) Sub(pth string) (FS, error) {
	if err := m.FS.MkdirAll(path.Clean(pth), m.perm); err != nil {
		return nil, err
	}
	return &subFS{dir: pth, FS: m}, nil
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

func (m *memFS) List(pth string) ([]os.FileInfo, error) {
	entries, err := m.FS.List(pth)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		infos[i], err = m.FS.Stat(path.Join(pth, e))
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name() < infos[j].Name()
	})

	return infos, nil
}

func (m *memFS) Remove(pth string) error {
	if e, err := m.Exists(pth); err != nil || !e {
		return errors.New("Invalid file path")
	}
	return m.RemoveAll(pth)
}

func (m *memFS) Rename(pth string, newPth string) error {
	return m.FS.Rename(pth, newPth)
}

func (m *memFS) Stat(pth string) (os.FileInfo, error) {
	return m.FS.Stat(pth)
}
