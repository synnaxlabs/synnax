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
	"os"
	"path"
	"path/filepath"
)

type defaultFS struct{ dirPerm, filePerm os.FileMode }

var Default = NewDefault(defaultPerm, defaultPerm)

// NewDefault returns an FS on the OS filesystem that creates directories with dirPerm
// and files with filePerm. The process umask can remove bits from both. The modes of
// files and directories that already exist do not change.
func NewDefault(dirPerm, filePerm os.FileMode) FS {
	return &defaultFS{dirPerm: dirPerm, filePerm: filePerm}
}

// Open implements FS.
func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(filepath.Clean(name), flag, d.filePerm)
}

// Sub implements FS.
func (d *defaultFS) Sub(name string) (FS, error) {
	if err := os.MkdirAll(name, d.dirPerm); err != nil {
		return nil, err
	}
	return &subFS{dir: name, FS: d}, nil
}

// Exists implements FS.
func (d *defaultFS) Exists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		err = nil
	}
	return false, err
}

// List implements FS.
func (d *defaultFS) List(dirName string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := os.Stat(path.Join(dirName, e.Name()))
		// An entry removed since ReadDir, or a symlink to nothing, has nothing to list.
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// Remove implements FS.
func (d *defaultFS) Remove(name string) error { return os.RemoveAll(name) }

// Rename implements FS.
func (d *defaultFS) Rename(name, newName string) error {
	return os.Rename(name, newName)
}

// Stat implements FS.
func (d *defaultFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }

// SyncDir implements FS.
func (d *defaultFS) SyncDir(name string) error { return syncDir(name) }
