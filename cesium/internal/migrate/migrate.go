// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
	"strconv"
)

type migration = func(ch core.Channel, fs xfs.FS) (core.Channel, error)

var migrations = map[string]migration{
	"01": migrate01,
	"02": migrate02,
}

func Migrate(
	ch core.Channel,
	fs xfs.FS,
	newVersion version.Version,
) error {
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
func migrate01(ch core.Channel, _ xfs.FS) (core.Channel, error) {
	return ch, nil
}

func migrate02(ch core.Channel, fs xfs.FS) (core.Channel, error) {
	err = meta.Validate(ch)
	var fieldErr validate.FieldError
	if errors.As(err, &fieldErr) {
		if fieldErr.Field == "index" && fieldErr.Message == "non-indexed channel must have an index" {
			if err := fs.Remove("."); err != nil {
				return err
			}
		}
	}
	return err
}
