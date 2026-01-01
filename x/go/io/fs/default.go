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
)

type defaultFS struct{ perm os.FileMode }

var Default FS = &defaultFS{perm: defaultPerm}

// Open implements FS.
func (d *defaultFS) Open(name string, flag int) (File, error) {
	return os.OpenFile(name, flag, d.perm)
}

// Sub implements FS.
func (d *defaultFS) Sub(name string) (FS, error) {
	if err := os.MkdirAll(name, d.perm); err != nil {
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
	infos := make([]os.FileInfo, len(entries))
	for i, e := range entries {
		if infos[i], err = os.Stat(path.Join(dirName, e.Name())); err != nil {
			return nil, err
		}
	}
	return infos, nil
}

// Remove implements FS.
func (d *defaultFS) Remove(name string) error { return os.RemoveAll(name) }

// Rename implements FS.
func (d *defaultFS) Rename(name string, newName string) error {
	return os.Rename(name, newName)
}

// Stat implements FS.
func (d *defaultFS) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
