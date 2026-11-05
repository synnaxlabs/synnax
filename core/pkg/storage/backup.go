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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble/v2/vfs"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const (
	kvDir         = "kv"
	cesiumDir     = "cesium"
	backupsDir    = "backups"
	partialSuffix = ".partial"
	keptBackups   = 3
)

// backupName matches a complete backup: the Unix seconds of the backup, padded so that
// names sort by time, and the version of the data.
var backupName = regexp.MustCompile(`^(\d{10})-(.+)$`)

// Backup is a copy of the metadata in a data directory: the key-value store and the
// metadata of each Cesium channel. It holds no telemetry.
type Backup struct {
	// Name is the name of the backup directory in the backups directory.
	Name string
	// Time is when the backup was taken, to the second.
	Time time.Time
	// Version is the version of the Core that last opened the data before the backup,
	// or UnknownVersion when no Core recorded one.
	Version string
}

// BackupsConfig is the configuration for the metadata backups of a data directory.
type BackupsConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Dirname is the data directory. It must exist.
	//
	// [REQUIRED]
	Dirname string
	// FS is the file system that holds Dirname.
	//
	// [OPTIONAL] - Defaults to xfs.Default.
	FS xfs.FS
	// LockFS is the file system that locks Dirname, so that a restore and a Core never
	// run on the data at once.
	//
	// [OPTIONAL] - Defaults to vfs.Default.
	LockFS vfs.FS
}

var (
	_ config.Config[BackupsConfig] = BackupsConfig{}
	// DefaultBackupsConfig is the default configuration for NewBackups.
	DefaultBackupsConfig = BackupsConfig{FS: xfs.Default, LockFS: vfs.Default}
)

// Override implements config.Config.
func (cfg BackupsConfig) Override(other BackupsConfig) BackupsConfig {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Dirname = override.String(cfg.Dirname, other.Dirname)
	cfg.FS = override.Nil(cfg.FS, other.FS)
	cfg.LockFS = override.Nil(cfg.LockFS, other.LockFS)
	return cfg
}

// Validate implements config.Config.
func (cfg BackupsConfig) Validate() error {
	v := validate.New("storage.backups")
	v.NotEmptyString("dirname", cfg.Dirname)
	v.NotNil("fs", cfg.FS)
	v.NotNil("lock_fs", cfg.LockFS)
	return v.Error()
}

// Backups are the metadata backups of a data directory. A Core takes one before a new
// version first opens the data, and keeps the newest three.
type Backups struct {
	cfg BackupsConfig
	fs  xfs.FS
}

// NewBackups returns the backups of the data directory in the configuration. It returns
// an error that matches fs.ErrNotExist when the data directory does not exist.
func NewBackups(cfgs ...BackupsConfig) (*Backups, error) {
	cfg, err := config.New(DefaultBackupsConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	if _, err = cfg.FS.Stat(cfg.Dirname); err != nil {
		return nil, err
	}
	fs, err := cfg.FS.Sub(cfg.Dirname)
	if err != nil {
		return nil, err
	}
	return &Backups{cfg: cfg, fs: fs}, nil
}

// List returns the complete backups, oldest first.
func (b *Backups) List() ([]Backup, error) {
	if exists, err := b.fs.Exists(backupsDir); err != nil || !exists {
		return nil, err
	}
	infos, err := b.fs.List(backupsDir)
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

// create copies the metadata to a new backup labeled with version, then removes all but
// the newest keptBackups backups.
func (b *Backups) create(version string) (Backup, error) {
	if err := b.removePartial(); err != nil {
		return Backup{}, err
	}
	now := time.Now()
	backup := Backup{
		Name:    fmt.Sprintf("%010d-%s", now.Unix(), version),
		Time:    time.Unix(now.Unix(), 0),
		Version: version,
	}
	partialDir := path.Join(backupsDir, backup.Name+partialSuffix)
	partial, err := b.fs.Sub(partialDir)
	if err != nil {
		return Backup{}, err
	}
	if err = xfs.CopyDir(b.fs, partial, kvDir); err != nil {
		return Backup{}, err
	}
	ts, err := b.fs.Sub(cesiumDir)
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
	if err = b.fs.Rename(partialDir, path.Join(backupsDir, backup.Name)); err != nil {
		return Backup{}, err
	}
	return backup, b.prune()
}

// removePartial removes the backups that a failed or interrupted create left.
func (b *Backups) removePartial() error {
	if exists, err := b.fs.Exists(backupsDir); err != nil || !exists {
		return err
	}
	infos, err := b.fs.List(backupsDir)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if strings.HasSuffix(info.Name(), partialSuffix) {
			if err = b.fs.Remove(path.Join(backupsDir, info.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// prune removes all but the newest keptBackups backups.
func (b *Backups) prune() error {
	backups, err := b.List()
	if err != nil {
		return err
	}
	for _, old := range backups[:max(len(backups)-keptBackups, 0)] {
		if err = b.fs.Remove(path.Join(backupsDir, old.Name)); err != nil {
			return err
		}
	}
	return nil
}
