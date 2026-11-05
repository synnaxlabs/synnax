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
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// Restored describes a backup that Restore put back.
type Restored struct {
	// Backup is the backup that Restore put back.
	Backup Backup
	// OrphanedDir holds the Cesium directories of the channels that the backup has no
	// metadata for. It is empty when there were none.
	OrphanedDir string
}

// Restore puts the named backup back in the data directory at dirname and records the
// backup's version, which is the version of the Core to run on the data. It moves the
// Cesium directory of each channel that the backup has no metadata for to
// Restored.OrphanedDir, and keeps the telemetry of every other channel.
//
// Restore holds the lock on the data directory, so it fails while a Core runs on it. It
// returns query.ErrNotFound when there is no complete backup with the name. A failed
// Restore can run again.
func Restore(dirname, name string) (res Restored, err error) {
	fs, err := openDataDir(dirname)
	if err != nil {
		return Restored{}, err
	}
	lock, err := acquireLock(dirname, vfs.Default)
	if err != nil {
		return Restored{}, err
	}
	defer func() { err = errors.Combine(err, lock.Close()) }()
	backups, err := listBackups(fs)
	if err != nil {
		return Restored{}, err
	}
	b, ok := lo.Find(backups, func(b Backup) bool { return b.Name == name })
	if !ok {
		return Restored{}, errors.Wrapf(query.ErrNotFound, "backup %s", name)
	}
	src := path.Join(backupsDir, name)
	orphaned, err := moveOrphans(fs, src)
	if err != nil {
		return Restored{}, err
	}
	if err = replaceKV(fs, src); err != nil {
		return Restored{}, err
	}
	keys, err := channelDirs(fs, path.Join(src, cesiumDir))
	if err != nil {
		return Restored{}, err
	}
	for _, key := range keys {
		if err = copyFileInto(
			fs,
			path.Join(src, cesiumDir, key, metaFile),
			path.Join(cesiumDir, key),
		); err != nil {
			return Restored{}, err
		}
	}
	if err = restoreDriverKey(fs, src); err != nil {
		return Restored{}, err
	}
	// A data directory that no Core recorded a version for has no version file.
	if b.Version == UnknownVersion {
		err = fs.Remove(versionFile)
	} else {
		err = writeVersion(fs, b.Version)
	}
	if err != nil {
		return Restored{}, err
	}
	res = Restored{Backup: b}
	if orphaned != "" {
		res.OrphanedDir = filepath.Join(dirname, filepath.FromSlash(orphaned))
	}
	return res, nil
}

// moveOrphans moves the Cesium directory of each channel that the backup at src has no
// metadata for to a new directory in the backups directory, and returns the path of
// that directory, or an empty string when there were no such channels.
func moveOrphans(fs xfs.FS, src string) (string, error) {
	live, err := channelDirs(fs, cesiumDir)
	if err != nil {
		return "", err
	}
	backedUp, err := channelDirs(fs, path.Join(src, cesiumDir))
	if err != nil {
		return "", err
	}
	known := set.New(backedUp...)
	dir := ""
	for _, key := range live {
		if known.Contains(key) {
			continue
		}
		if dir == "" {
			dir = path.Join(backupsDir, fmt.Sprintf("orphaned-%d", time.Now().Unix()))
			if _, err = fs.Sub(dir); err != nil {
				return "", err
			}
		}
		if err = fs.Rename(path.Join(cesiumDir, key), path.Join(dir, key)); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// replaceKV replaces the key-value store with the copy in the backup at src. The live
// store stays in place until the copy is complete.
func replaceKV(fs xfs.FS, src string) error {
	staged, retired := kvDir+".restore", kvDir+".old"
	if err := fs.Remove(staged); err != nil {
		return err
	}
	if err := fs.Remove(retired); err != nil {
		return err
	}
	if err := copyDir(fs, path.Join(src, kvDir), staged); err != nil {
		return err
	}
	if exists, err := fs.Exists(kvDir); err != nil {
		return err
	} else if exists {
		if err = fs.Rename(kvDir, retired); err != nil {
			return err
		}
	}
	if err := fs.Rename(staged, kvDir); err != nil {
		return err
	}
	return fs.Remove(retired)
}

func restoreDriverKey(fs xfs.FS, src string) error {
	key := path.Join(src, DriverKeyFile)
	exists, err := fs.Exists(key)
	if err != nil {
		return err
	}
	if !exists {
		return fs.Remove(DriverKeyFile)
	}
	return copyFile(fs, key, DriverKeyFile)
}
