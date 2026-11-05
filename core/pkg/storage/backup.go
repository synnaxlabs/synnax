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
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

const (
	kvDir         = "kv"
	cesiumDir     = "cesium"
	metaFile      = "meta.json"
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
	partial := path.Join(backupsDir, b.Name+partialSuffix)
	if err := copyDir(fs, kvDir, path.Join(partial, kvDir)); err != nil {
		return Backup{}, err
	}
	keys, err := channelDirs(fs, cesiumDir)
	if err != nil {
		return Backup{}, err
	}
	for _, key := range keys {
		if err = copyFileInto(
			fs,
			path.Join(cesiumDir, key, metaFile),
			path.Join(partial, cesiumDir, key),
		); err != nil {
			return Backup{}, err
		}
	}
	if err = copyFileInto(fs, DriverKeyFile, partial); err != nil {
		return Backup{}, err
	}
	if err = fs.Rename(partial, path.Join(backupsDir, b.Name)); err != nil {
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

// channelDirs returns the names of the channel directories in the Cesium directory dir,
// or nothing when dir does not exist.
func channelDirs(fs xfs.FS, dir string) ([]string, error) {
	if exists, err := fs.Exists(dir); err != nil || !exists {
		return nil, err
	}
	infos, err := fs.List(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, info := range infos {
		_, err := strconv.ParseUint(info.Name(), 10, 32)
		if info.IsDir() && err == nil {
			names = append(names, info.Name())
		}
	}
	return names, nil
}

func copyDir(fs xfs.FS, from, to string) error {
	if _, err := fs.Sub(to); err != nil {
		return err
	}
	infos, err := fs.List(from)
	if err != nil {
		return err
	}
	for _, info := range infos {
		src, dst := path.Join(from, info.Name()), path.Join(to, info.Name())
		if info.IsDir() {
			err = copyDir(fs, src, dst)
		} else {
			err = copyFile(fs, src, dst)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// copyFileInto copies the file at src, when it exists, into the directory dir.
func copyFileInto(fs xfs.FS, src, dir string) error {
	if exists, err := fs.Exists(src); err != nil || !exists {
		return err
	}
	if _, err := fs.Sub(dir); err != nil {
		return err
	}
	return copyFile(fs, src, path.Join(dir, path.Base(src)))
}

func copyFile(fs xfs.FS, from, to string) (err error) {
	src, err := fs.Open(from, os.O_RDONLY)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, src.Close()) }()
	dst, err := fs.Open(to, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return err
	}
	defer func() { err = errors.Combine(err, dst.Close()) }()
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}
	return dst.Sync()
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
