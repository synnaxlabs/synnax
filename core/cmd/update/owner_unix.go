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
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
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
	owner := info.Sys().(*syscall.Stat_t)
	if owner.Uid == 0 {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info, err = d.Info(); err != nil {
			return err
		}
		if info.Sys().(*syscall.Stat_t).Uid != 0 {
			return nil
		}
		return os.Lchown(path, int(owner.Uid), int(owner.Gid))
	})
}
