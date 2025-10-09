// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pebblekv

import (
	"github.com/cockroachdb/errors/oserror"
	pebblev1 "github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/vfs"
	vfsv1 "github.com/cockroachdb/pebble/vfs"
	"github.com/synnaxlabs/x/errors"
)

func RequiresMigration(dirname string, fs vfs.FS) (bool, error) {
	dbDesc, err := pebble.Peek(dirname, fs)
	if err != nil {
		return false, errors.Skip(err, oserror.ErrNotExist)
	}
	return !dbDesc.FormatMajorVersion.IsSupported() &&
		uint64(dbDesc.FormatMajorVersion) < uint64(pebble.FormatNewest), nil
}

func Migrate(dirname string) error {
	// Check if the database requires migration using v2
	dbDesc, err := pebble.Peek(dirname, vfs.Default)
	if err != nil {
		return err
	}

	// Only use v1 for truly old formats (< v2's minimum supported version)
	if uint64(dbDesc.FormatMajorVersion) < uint64(pebble.FormatMinSupported) {
		oldDB, err := pebblev1.Open(dirname, &pebblev1.Options{FS: vfsv1.Default})
		if err != nil {
			return err
		}
		if err := oldDB.RatchetFormatMajorVersion(pebblev1.FormatNewest); err != nil {
			return err
		}
		if err := oldDB.Close(); err != nil {
			return err
		}
	}

	// For all other formats, open with v2 (handles both supported formats and auto-migration)
	db, err := pebble.Open(dirname, &pebble.Options{
		FS:                 vfs.Default,
		FormatMajorVersion: pebble.FormatNewest,
	})
	if err != nil {
		return err
	}
	return db.Close()
}
