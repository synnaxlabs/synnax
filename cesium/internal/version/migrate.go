// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

import (
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"strconv"
)

var migrations = map[string]func(fs xfs.FS) error{
	"01": migrate01,
}

func Migrate(fs xfs.FS, oldVersion Version, newVersion Version) error {
	migrate, ok := migrations[strconv.Itoa(int(oldVersion))+strconv.Itoa(int(newVersion))]
	if !ok {
		return errors.Newf("migration from version %d to version %d not found", oldVersion, newVersion)
	}
	err := migrate(fs)
	if err != nil {
		return errors.Wrap(err, "version migration error")
	}

	return nil
}

// migrate01 is a migration from unversioned to v1 (which is the same as unversioned)
func migrate01(_ xfs.FS) error {
	return nil
}
