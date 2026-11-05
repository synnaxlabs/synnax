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
	"strings"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"go.uber.org/zap"
	"golang.org/x/mod/semver"
)

// versionFile is the file in the data directory that holds the version of the last Core
// that opened it.
const versionFile = "VERSION"

// UnknownVersion is the Backup.Version of data that no Core recorded a version for.
const UnknownVersion = "unknown"

// toSemver returns version in the form that the semver package parses.
func toSemver(version string) string { return "v" + version }

// isDev reports whether version belongs to a dev build, which neither checks nor
// records the version. A dev version need not be valid semver, since its pre-release
// can hold a commit hash with a leading zero.
func isDev(version string) bool { return strings.HasPrefix(version, "0.0.") }

func checkVersion(cfg LayerConfig, rootFS xfs.FS) error {
	if isDev(cfg.Version) {
		return nil
	}
	fs, err := rootFS.Sub(cfg.Dirname)
	if err != nil {
		return err
	}
	recorded, found, err := readVersion(fs)
	if err != nil {
		return err
	}
	if found {
		switch semver.Compare(toSemver(cfg.Version), toSemver(recorded)) {
		case 0:
			return nil
		case -1:
			return errors.Newf(
				"Data was last opened by Core %s. Run Core %s or newer, or restore a "+
					"backup with `synnax restore`.",
				recorded,
				recorded,
			)
		}
	} else {
		recorded = UnknownVersion
	}
	hasMetadata, err := fs.Exists(kvDir)
	if err != nil {
		return err
	}
	if hasMetadata {
		cfg.L.Info(
			"backing up metadata before the first start of a new version",
			zap.String("from", recorded),
			zap.String("to", cfg.Version),
		)
		b, err := backup(fs, recorded)
		if err != nil {
			return errors.Wrap(err, "failed to back up metadata")
		}
		cfg.L.Info("backed up metadata", zap.String("backup", b.Name))
		if err = prune(fs); err != nil {
			return errors.Wrap(err, "failed to prune old backups")
		}
	}
	return writeVersion(fs, cfg.Version)
}

func readVersion(fs xfs.FS) (string, bool, error) {
	b, found, err := readFile(fs, versionFile)
	if err != nil || !found {
		return "", false, err
	}
	version := strings.TrimSpace(string(b))
	if !semver.IsValid(toSemver(version)) {
		return "", false, errors.Newf("invalid version %q in %s", version, versionFile)
	}
	return version, true, nil
}

// writeVersion replaces the version file, so an interrupted write leaves the old file
// in place.
func writeVersion(fs xfs.FS, version string) error {
	tmp := versionFile + ".tmp"
	if err := writeFile(fs, tmp, []byte(version)); err != nil {
		return err
	}
	return fs.Rename(tmp, versionFile)
}
