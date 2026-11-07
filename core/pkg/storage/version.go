// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage

import (
	"os"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"go.uber.org/zap"
)

// versionFile is the file in the data directory that holds the version of the last Core
// that opened it.
const versionFile = "VERSION"

// UnknownVersion is the Backup.Version of data that no Core recorded a version for.
const UnknownVersion = "unknown"

// checkVersion compares current, the version of the Core that opens the data, with the
// recorded version. It refuses data that a newer Core opened, and backs up data that an
// older Core opened before it records current.
func (b *Backups) checkVersion(current string) error {
	if version.IsDev(current) {
		return nil
	}
	recorded, err := b.readVersion()
	if err != nil {
		return err
	}
	if recorded != UnknownVersion {
		switch version.Compare(current, recorded) {
		case 0:
			return nil
		case -1:
			return errors.Newf(
				"Data was last opened by Core %s. Run Core %s or newer, or undo the "+
					"update with `synnax update undo`.",
				recorded,
				recorded,
			)
		}
	}
	hasMetadata, err := b.fs.Exists(kvDir)
	if err != nil {
		return err
	}
	if hasMetadata {
		b.cfg.L.Info(
			"backing up metadata before the first start of a new version",
			zap.String("from", recorded),
			zap.String("to", current),
		)
		backup, err := b.create(recorded)
		if err != nil {
			return errors.Wrap(err, "failed to back up metadata")
		}
		b.cfg.L.Info("backed up metadata", zap.String("backup", backup.Name))
	}
	return xfs.WriteFile(b.fs, versionFile, []byte(current))
}

// readVersion returns the recorded version, or UnknownVersion when there is none.
func (b *Backups) readVersion() (string, error) {
	data, err := xfs.ReadFile(b.fs, versionFile)
	if errors.Is(err, os.ErrNotExist) {
		return UnknownVersion, nil
	}
	if err != nil {
		return "", err
	}
	recorded := strings.TrimSpace(string(data))
	if !version.Valid(recorded) {
		return "", errors.Newf("invalid version %q in %s", recorded, versionFile)
	}
	return recorded, nil
}
