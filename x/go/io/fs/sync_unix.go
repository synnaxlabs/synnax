// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package fs

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/synnaxlabs/x/errors"
)

func syncDir(name string) (err error) {
	// O_DIRECTORY fails a name that is not a directory before the open can block on it
	// or need read permission.
	dir, err := os.OpenFile(filepath.Clean(name), os.O_RDONLY|syscall.O_DIRECTORY, 0)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, dir.Close()) }()
	return dir.Sync()
}
