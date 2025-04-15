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
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
	"strconv"
)

var migrations = map[string]func(fs xfs.FS) error{
	"01": migrate01,
	"02": migrate02,
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

func migrate02(fs xfs.FS) error {
	dirs, err := fs.List(".")
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		subFS, err := fs.Sub(dir.Name())
		if err != nil {
			return err
		}
		_, err = meta.Read(subFS, &binary.JSONCodec{})
		if err == nil {
			continue
		}
		var fieldErr validate.FieldError
		if errors.As(err, &fieldErr) {
			if fieldErr.Field == "index" && fieldErr.Message == "non-indexed channel must have an index" {
				if err := fs.Remove(dir.Name()); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	return nil
}
