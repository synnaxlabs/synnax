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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
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
// returns query.ErrNotFound when there is no complete backup with the name, and an
// error without changing the data when the backup lacks a part. A failed Restore can
// run again, and OpenLayer does not open the data until one finishes.
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
	src, err := fs.Sub(path.Join(backupsDir, name))
	if err != nil {
		return Restored{}, err
	}
	for _, dir := range []string{kvDir, cesiumDir} {
		if exists, err := src.Exists(dir); err != nil {
			return Restored{}, err
		} else if !exists {
			return Restored{}, errors.Newf("backup %s has no %s directory", name, dir)
		}
	}
	if err = writeFile(fs, RestoreMarkerFile, []byte(name)); err != nil {
		return Restored{}, err
	}
	orphaned, err := restoreTS(fs, src)
	if err != nil {
		return Restored{}, err
	}
	if err = replaceKV(fs, src); err != nil {
		return Restored{}, err
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
	if err = fs.Remove(RestoreMarkerFile); err != nil {
		return Restored{}, err
	}
	res = Restored{Backup: b}
	if orphaned != "" {
		res.OrphanedDir = filepath.Join(dirname, filepath.FromSlash(orphaned))
	}
	return res, nil
}

// checkRestore returns an error when a Restore on the data directory did not finish.
func checkRestore(cfg LayerConfig, rootFS xfs.FS) error {
	fs, err := rootFS.Sub(cfg.Dirname)
	if err != nil {
		return err
	}
	name, found, err := readFile(fs, RestoreMarkerFile)
	if err != nil || !found {
		return err
	}
	return errors.Newf(
		"The restore of backup %s did not finish. Run `synnax restore %s` again.",
		name,
		name,
	)
}

// restoreTS puts back the Cesium metadata of the backup src. It returns the path of the
// directory that holds the channels the backup does not know, or an empty string when
// there are none.
func restoreTS(fs, src xfs.FS) (string, error) {
	ts, err := fs.Sub(cesiumDir)
	if err != nil {
		return "", err
	}
	tsBackup, err := src.Sub(cesiumDir)
	if err != nil {
		return "", err
	}
	dir := path.Join(backupsDir, fmt.Sprintf("orphaned-%d", time.Now().Unix()))
	orphans, err := fs.Sub(dir)
	if err != nil {
		return "", err
	}
	moved, err := cesium.RestoreMeta(ts, tsBackup, orphans)
	if err != nil {
		return "", err
	}
	if len(moved) > 0 {
		return dir, nil
	}
	// A restore that failed earlier in the same second can have moved channels here.
	if infos, err := fs.List(dir); err != nil {
		return "", err
	} else if len(infos) > 0 {
		return dir, nil
	}
	return "", fs.Remove(dir)
}

// replaceKV replaces the key-value store with the copy in the backup src. The live
// store stays in place until the copy is complete.
func replaceKV(fs, src xfs.FS) error {
	staged, retired := kvDir+".restore", kvDir+".old"
	if err := fs.Remove(staged); err != nil {
		return err
	}
	if err := fs.Remove(retired); err != nil {
		return err
	}
	kv, err := src.Sub(kvDir)
	if err != nil {
		return err
	}
	stagedKV, err := fs.Sub(staged)
	if err != nil {
		return err
	}
	if err = copyDir(kv, stagedKV, ""); err != nil {
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

func restoreDriverKey(fs, src xfs.FS) error {
	exists, err := src.Exists(DriverKeyFile)
	if err != nil {
		return err
	}
	if !exists {
		return fs.Remove(DriverKeyFile)
	}
	return xfs.CopyFile(src, fs, DriverKeyFile)
}
