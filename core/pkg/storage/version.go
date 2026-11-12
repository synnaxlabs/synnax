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

// ErrRefused is returned when this Core version does not open the data directory: a
// newer Core opened it, a restore on it did not finish, or an update to this version
// failed. Each open fails the same way until a person acts.
var ErrRefused = errors.New("data refused")

// UnknownVersion is the Backup.Version of data that no Core recorded a version for.
const UnknownVersion = "unknown"

// checkVersion compares current, the version of the Core that opens the data, with the
// recorded version. It refuses data that a newer Core opened. For data that an older
// Core opened, it records current. Unless backupDisabled, it first backs up the
// metadata, marks the update, and returns it. It returns the update also when it fails
// after the mark, so that the caller undoes it.
func (b *Backups) checkVersion(current string, backupDisabled bool) (*Update, error) {
	if version.IsDev(current) {
		return nil, nil
	}
	recorded, err := b.readVersion()
	if err != nil {
		return nil, err
	}
	if recorded != UnknownVersion {
		switch version.Compare(current, recorded) {
		case 0:
			return nil, nil
		case -1:
			return nil, errors.Mark(errors.Newf(
				"Data was last opened by Core %s. Run Core %s or newer, or undo the "+
					"update with `synnax update undo`.",
				recorded,
				recorded,
			), ErrRefused)
		}
	}
	hasMetadata, err := b.fs.Exists(kvDir)
	if err != nil {
		return nil, err
	}
	var update *Update
	if hasMetadata && backupDisabled {
		b.cfg.L.Warn(
			"skipping the backup before the first start of a new version, so a "+
				"failed update cannot be undone",
			zap.String("from", recorded),
			zap.String("to", current),
		)
	} else if hasMetadata {
		b.cfg.L.Info(
			"backing up metadata before the first start of a new version",
			zap.String("from", recorded),
			zap.String("to", current),
		)
		backup, err := b.create(recorded)
		if err != nil {
			return nil, errors.Wrap(err, "failed to back up metadata")
		}
		b.cfg.L.Info("backed up metadata", zap.String("backup", backup.Name))
		// UpdateState relies on the marker going before the version.
		if err = xfs.WriteFile(
			b.fs,
			updateMarkerFile,
			[]byte(backup.Name+"\n"+current),
		); err != nil {
			return nil, err
		}
		update = &Update{From: recorded, To: current, backup: backup.Name, backups: b}
	}
	return update, xfs.WriteFile(b.fs, versionFile, []byte(current))
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
