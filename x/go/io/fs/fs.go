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
	"fmt"
	"github.com/cockroachdb/pebble/vfs"
	"io"
	"os"
	"path"
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
	List() ([]os.FileInfo, error)
	Exists(name string) (bool, error)
}

type defaultFS struct {
	dir  string
	perm os.FileMode
}

var Default FS = &defaultFS{perm: defaultPerm}

func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(path.Join(d.dir, name), flag, d.perm)
}

func (d *defaultFS) Sub(name string) (FS, error) {
	return OSDirFS(path.Join(d.dir, name))
}

func (d *defaultFS) Exists(name string) (bool, error) {
	_, err := os.Stat(path.Join(d.dir, name))
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (d *defaultFS) List() ([]os.FileInfo, error) {
	entries, err := os.ReadDir(d.dir)
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

func OSDirFS(dir string) (FS, error) {
	err := os.MkdirAll(dir, defaultPerm)
	return &defaultFS{dir: dir, perm: defaultPerm}, err
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
	if err := m.FS.MkdirAll(name, m.perm); err != nil {
		return nil, err
	}
	ret := &memFS{
		FS: vfs.NewMem(),
	}
	ret.PathJoin(name)
	return ret, nil
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

func (m *memFS) List() ([]os.FileInfo, error) {
	entries, err := m.FS.List("")
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		infos[i], err = m.FS.Stat(e)
		fmt.Println(e)
		if err != nil {
			return nil, err
		}
	}
	return infos, nil
}
