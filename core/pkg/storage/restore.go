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
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/query"
)

// restoreMarkerFile is the file in the data directory that marks a Restore that did not
// finish. It holds the name of the backup.
const restoreMarkerFile = "RESTORING"

// orphansDir is where a Restore stages, inside the Cesium directory, the channels that
// the backup does not know.
const orphansDir = "orphaned"

// Restored describes a backup that Restore put back.
type Restored struct {
	// Backup is the backup that Restore put back.
	Backup Backup
	// OrphanedDir holds the Cesium directories of the channels that the backup has no
	// metadata for. It is empty when there were none.
	OrphanedDir string
}

// Restore puts the named backup back in the data directory and records the backup's
// version, which is the version of the Core to run on the data. It moves the Cesium
// directory of each channel that the backup has no metadata for to
// Restored.OrphanedDir, and keeps the telemetry of every other channel. It also ends an
// update that did not commit.
//
// Restore holds the lock on the data directory, so it fails while a Core runs on it. It
// returns query.ErrNotFound when there is no complete backup with the name, and an
// error without changing the data when the backup lacks a part. A failed Restore can
// run again, and OpenLayer does not open the data until one finishes.
func (b *Backups) Restore(name string) (Restored, error) {
	return b.locked(func() (Restored, error) { return b.restore(name) })
}

// locked runs f while it holds the lock on the data directory.
func (b *Backups) locked(f func() (Restored, error)) (res Restored, err error) {
	lock, err := acquireLock(b.cfg.Dirname, b.cfg.LockFS)
	if err != nil {
		return Restored{}, err
	}
	defer func() {
		if err = errors.Combine(err, lock.Close()); err != nil {
			res = Restored{}
		}
	}()
	return f()
}

// restore is Restore for a caller that holds the lock on the data directory.
func (b *Backups) restore(name string) (res Restored, err error) {
	backups, err := b.List()
	if err != nil {
		return Restored{}, err
	}
	backup, ok := lo.Find(backups, func(bk Backup) bool { return bk.Name == name })
	if !ok {
		return Restored{}, errors.Wrapf(query.ErrNotFound, "backup %s", name)
	}
	src, err := b.extract(name)
	defer func() { err = errors.Combine(err, b.fs.Remove(extractedDir(name))) }()
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
	if err = xfs.WriteFile(b.fs, restoreMarkerFile, []byte(name)); err != nil {
		return Restored{}, err
	}
	orphaned, err := b.restoreTS(src)
	if err != nil {
		return Restored{}, err
	}
	if err = b.replaceKV(src); err != nil {
		return Restored{}, err
	}
	// A data directory that no Core recorded a version for has no version file.
	if backup.Version == UnknownVersion {
		err = b.fs.Remove(versionFile)
	} else {
		err = xfs.WriteFile(b.fs, versionFile, []byte(backup.Version))
	}
	if err != nil {
		return Restored{}, err
	}
	if err = b.fs.Remove(updateMarkerFile); err != nil {
		return Restored{}, err
	}
	if err = b.fs.Remove(restoreMarkerFile); err != nil {
		return Restored{}, err
	}
	res = Restored{Backup: backup}
	if orphaned != "" {
		res.OrphanedDir = filepath.Join(b.cfg.Dirname, filepath.FromSlash(orphaned))
	}
	return res, nil
}

// ErrUndone is returned when an earlier undo already restored the newest backup.
var ErrUndone = errors.New("update already undone")

// Undo restores the newest backup, as Restore does. It returns an error when there is
// no backup, and ErrUndone when an earlier undo already restored the newest backup.
func (b *Backups) Undo() (Restored, error) {
	return b.locked(func() (Restored, error) {
		backups, err := b.List()
		if err != nil {
			return Restored{}, err
		}
		if len(backups) == 0 {
			return Restored{}, errors.New(
				"there is no update to undo: the data has no backups",
			)
		}
		newest := backups[len(backups)-1]
		recorded, err := b.readVersion()
		if err != nil {
			return Restored{}, err
		}
		if recorded == newest.Version {
			return Restored{}, errors.Mark(errors.Newf(
				"The last update is already undone. To restore backup %s again, "+
					"run `synnax update undo %s`.",
				newest.Name,
				newest.Name,
			), ErrUndone)
		}
		return b.restore(newest.Name)
	})
}

// extract unpacks the archive of the named backup into an empty directory and returns
// it.
func (b *Backups) extract(name string) (xfs.FS, error) {
	dir := extractedDir(name)
	if err := b.fs.Remove(dir); err != nil {
		return nil, err
	}
	dst, err := b.fs.Sub(dir)
	if err != nil {
		return nil, err
	}
	archive := path.Join(backupsDir, name+archiveExt)
	if err = extractArchive(b.fs, archive, dst); err != nil {
		return nil, err
	}
	return dst, nil
}

// extractedDir returns the directory that extract unpacks the named backup into.
func extractedDir(name string) string {
	return path.Join(backupsDir, name+".restore")
}

// checkRestore returns an error when a Restore did not finish.
func (b *Backups) checkRestore() error {
	name, err := xfs.ReadFile(b.fs, restoreMarkerFile)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.Mark(errors.Newf(
		"The restore of backup %s did not finish. Run `synnax update undo %s` again.",
		name,
		name,
	), ErrRefused)
}

// restoreTS puts back the Cesium metadata of the backup src. It returns the path of the
// directory that holds the channels the backup does not know, or an empty string when
// there are none.
func (b *Backups) restoreTS(src xfs.FS) (string, error) {
	ts, err := b.fs.Sub(cesiumDir)
	if err != nil {
		return "", err
	}
	tsBackup, err := src.Sub(cesiumDir)
	if err != nil {
		return "", err
	}
	if err = cesium.RestoreMeta(ts, tsBackup, orphansDir); err != nil {
		return "", err
	}
	staged := path.Join(cesiumDir, orphansDir)
	if exists, err := b.fs.Exists(staged); err != nil || !exists {
		return "", err
	}
	dir := path.Join(backupsDir, fmt.Sprintf("orphaned-%d", time.Now().Unix()))
	if err := b.fs.Rename(staged, dir); err != nil {
		return "", err
	}
	return dir, nil
}

// replaceKV replaces the key-value store with the copy in the backup src. The live
// store stays in place until the copy is complete.
func (b *Backups) replaceKV(src xfs.FS) error {
	staged, retired := kvDir+".restore", kvDir+".old"
	if err := b.fs.Remove(staged); err != nil {
		return err
	}
	if err := b.fs.Remove(retired); err != nil {
		return err
	}
	kv, err := src.Sub(kvDir)
	if err != nil {
		return err
	}
	stagedKV, err := b.fs.Sub(staged)
	if err != nil {
		return err
	}
	if err = xfs.CopyDir(kv, stagedKV, ""); err != nil {
		return err
	}
	if exists, err := b.fs.Exists(kvDir); err != nil {
		return err
	} else if exists {
		if err = b.fs.Rename(kvDir, retired); err != nil {
			return err
		}
	}
	if err := b.fs.Rename(staged, kvDir); err != nil {
		return err
	}
	return b.fs.Remove(retired)
}
