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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

const (
	kvDir         = "kv"
	cesiumDir     = "cesium"
	backupsDir    = "backups"
	partialSuffix = ".partial"
	keptBackups   = 3
)

// DriverKeyFile is the file in the data directory that holds the API key of the
// embedded Driver.
const DriverKeyFile = "embedded-driver-key"

// backupName matches a complete backup: the Unix seconds of the backup, padded so that
// names sort by time, and the version of the data.
var backupName = regexp.MustCompile(`^(\d{10})-(.+)$`)

// Backup is a copy of the metadata in a data directory: the key-value store, the
// metadata of each Cesium channel, and the embedded Driver key. It holds no telemetry.
type Backup struct {
	// Name is the name of the backup directory in the backups directory.
	Name string
	// Time is when the backup was taken, to the second.
	Time time.Time
	// Version is the version of the Core that last opened the data before the backup,
	// or UnknownVersion when no Core recorded one.
	Version string
}

// ListBackups returns the complete backups in the data directory at dirname, oldest
// first.
func ListBackups(dirname string) ([]Backup, error) {
	fs, err := openDataDir(dirname)
	if err != nil {
		return nil, err
	}
	return listBackups(fs)
}

// openDataDir returns a file system rooted at the existing data directory dirname.
func openDataDir(dirname string) (xfs.FS, error) {
	if _, err := xfs.Default.Stat(dirname); err != nil {
		return nil, err
	}
	return xfs.Default.Sub(dirname)
}

func listBackups(fs xfs.FS) ([]Backup, error) {
	if exists, err := fs.Exists(backupsDir); err != nil || !exists {
		return nil, err
	}
	infos, err := fs.List(backupsDir)
	if err != nil {
		return nil, err
	}
	var backups []Backup
	for _, info := range infos {
		match := backupName.FindStringSubmatch(info.Name())
		if !info.IsDir() || match == nil || strings.HasSuffix(match[2], partialSuffix) {
			continue
		}
		seconds, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			return nil, err
		}
		backups = append(backups, Backup{
			Name:    info.Name(),
			Time:    time.Unix(seconds, 0),
			Version: match[2],
		})
	}
	return backups, nil
}

// backup copies the metadata in fs to a new backup labeled with version.
func backup(fs xfs.FS, version string) (Backup, error) {
	now := time.Now()
	b := Backup{
		Name:    fmt.Sprintf("%010d-%s", now.Unix(), version),
		Time:    time.Unix(now.Unix(), 0),
		Version: version,
	}
	partialDir := path.Join(backupsDir, b.Name+partialSuffix)
	partial, err := fs.Sub(partialDir)
	if err != nil {
		return Backup{}, err
	}
	if err = copyDir(fs, partial, kvDir); err != nil {
		return Backup{}, err
	}
	ts, err := fs.Sub(cesiumDir)
	if err != nil {
		return Backup{}, err
	}
	tsBackup, err := partial.Sub(cesiumDir)
	if err != nil {
		return Backup{}, err
	}
	if err = cesium.BackupMeta(ts, tsBackup); err != nil {
		return Backup{}, err
	}
	hasDriverKey, err := fs.Exists(DriverKeyFile)
	if err != nil {
		return Backup{}, err
	}
	if hasDriverKey {
		if err = xfs.CopyFile(fs, partial, DriverKeyFile); err != nil {
			return Backup{}, err
		}
	}
	if err = fs.Rename(partialDir, path.Join(backupsDir, b.Name)); err != nil {
		return Backup{}, err
	}
	return b, nil
}

// prune removes unfinished backups and all but the newest keptBackups complete ones.
func prune(fs xfs.FS) error {
	infos, err := fs.List(backupsDir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if strings.HasSuffix(info.Name(), partialSuffix) {
			if err = fs.Remove(path.Join(backupsDir, info.Name())); err != nil {
				return err
			}
		}
	}
	backups, err := listBackups(fs)
	if err != nil {
		return err
	}
	for _, b := range backups[:max(len(backups)-keptBackups, 0)] {
		if err = fs.Remove(path.Join(backupsDir, b.Name)); err != nil {
			return err
		}
	}
	return nil
}

// copyDir copies the directory dir in src, with everything in it, to dir in dst.
func copyDir(src, dst xfs.FS, dir string) error {
	if _, err := dst.Sub(dir); err != nil {
		return err
	}
	infos, err := src.List(dir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		name := path.Join(dir, info.Name())
		if info.IsDir() {
			err = copyDir(src, dst, name)
		} else {
			err = xfs.CopyFile(src, dst, name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(fs xfs.FS, name string, data []byte) (err error) {
	f, err := fs.Open(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, f.Close()) }()
	if _, err = f.Write(data); err != nil {
		return err
	}
	return f.Sync()
}
