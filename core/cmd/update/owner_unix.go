// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build unix

package update

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/synnaxlabs/x/errors"
)

// matchOwner gives the owner of dir each file in it that root owns, when root runs the
// command. A Core service that runs as another user can then open the files that the
// command wrote.
func matchOwner(dir string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	owner := status(info)
	if owner.Uid == 0 {
		return nil
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	// The root stops a symlink that the owner of dir adds during the walk from sending
	// a chown outside dir.
	walkErr := fs.WalkDir(
		root.FS(),
		".",
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() && !d.Type().IsRegular() {
				return nil
			}
			return chownRootOwned(root, path, owner)
		},
	)
	return errors.Combine(walkErr, root.Close())
}

// chownRootOwned gives owner the regular file or directory at path when root owns it.
// It checks and changes the inode it opened, so the owner of dir cannot swap in a hard
// link to a root-owned file outside dir between the check and the chown. Special files
// and files with more than one link keep their owner.
func chownRootOwned(root *os.Root, path string, owner *syscall.Stat_t) error {
	f, err := root.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		return errors.Combine(err, f.Close())
	}
	st := status(info)
	sole := info.IsDir() || (info.Mode().IsRegular() && st.Nlink == 1)
	if st.Uid != 0 || !sole {
		return f.Close()
	}
	return errors.Combine(f.Chown(int(owner.Uid), int(owner.Gid)), f.Close())
}

func status(info fs.FileInfo) *syscall.Stat_t {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		panic(fmt.Sprintf("file info of %s is a %T", info.Name(), info.Sys()))
	}
	return st
}
