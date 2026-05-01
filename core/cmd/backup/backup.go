// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package backup implements the "synnax backup" command, which copies a Synnax data
// directory to a destination path. It optionally skips channel data files (the
// "*.domain" files written by the Cesium time-series engine) so that only the cluster
// configuration is copied.
package backup

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// CesiumDir is the name of the subdirectory within a Synnax data directory that
// contains channel data managed by the Cesium time-series engine.
const CesiumDir = "cesium"

// DomainExtension is the file extension used by Cesium for channel data files.
const DomainExtension = ".domain"

// LockFile is the name of the lock file written by the storage layer at the root of
// the data directory. It is always excluded from backups because it is process-local
// and would prevent a future Synnax node from acquiring the lock at the destination.
const LockFile = "LOCK"

// Config configures a Backup operation.
type Config struct {
	// Src is the path to the source Synnax data directory.
	Src string
	// Dst is the path where the data directory will be copied.
	Dst string
	// NoData, when true, excludes channel data files (*.domain entries within the
	// cesium/ subtree) from the backup. Channel configuration (meta.json, the kv/
	// store) is still copied.
	NoData bool
	// Overwrite, when true, allows the backup to write into a destination that already
	// exists. When false, Backup returns an error if Dst already exists.
	Overwrite bool
}

// Backup copies the directory at cfg.Src to cfg.Dst. See Config for details on
// behavior.
func Backup(cfg Config) error {
	if cfg.Src == "" {
		return errors.New("src is required")
	}
	if cfg.Dst == "" {
		return errors.New("dst is required")
	}
	absSrc, err := filepath.Abs(cfg.Src)
	if err != nil {
		return errors.Wrapf(err, "resolve src path %q", cfg.Src)
	}
	absDst, err := filepath.Abs(cfg.Dst)
	if err != nil {
		return errors.Wrapf(err, "resolve dst path %q", cfg.Dst)
	}
	if absSrc == absDst {
		return errors.Newf("src and dst must not be the same path: %s", absSrc)
	}
	if isSubpath(absSrc, absDst) {
		return errors.Newf("dst %q is inside src %q", absDst, absSrc)
	}
	srcInfo, err := os.Stat(absSrc)
	if err != nil {
		return errors.Wrapf(err, "stat src %q", absSrc)
	}
	if !srcInfo.IsDir() {
		return errors.Newf("src %q is not a directory", absSrc)
	}
	if _, err = os.Stat(absDst); err == nil {
		if !cfg.Overwrite {
			return errors.Newf(
				"dst %q already exists; pass --overwrite to write into it",
				absDst,
			)
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "stat dst %q", absDst)
	}
	return filepath.WalkDir(absSrc, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.Wrapf(walkErr, "walk %q", path)
		}
		rel, err := filepath.Rel(absSrc, path)
		if err != nil {
			return errors.Wrapf(err, "compute relative path of %q", path)
		}
		if shouldSkip(rel, d.IsDir(), cfg.NoData) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		dstPath := filepath.Join(absDst, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return errors.Wrapf(err, "stat %q", path)
			}
			if err = os.MkdirAll(dstPath, info.Mode().Perm()); err != nil {
				return errors.Wrapf(err, "create directory %q", dstPath)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if err = copyFile(path, dstPath); err != nil {
			return errors.Wrapf(err, "copy %q to %q", path, dstPath)
		}
		return nil
	})
}

// shouldSkip reports whether the entry at the given path (relative to the source
// directory) should be excluded from the backup.
func shouldSkip(rel string, isDir bool, noData bool) bool {
	if rel == "." {
		return false
	}
	if !isDir && rel == LockFile {
		return true
	}
	if noData && !isDir && isCesiumDataFile(rel) {
		return true
	}
	return false
}

// isCesiumDataFile reports whether rel refers to a channel data file inside the
// cesium/ subdirectory. Channel data files are identified by the ".domain" extension
// used by the Cesium time-series engine.
func isCesiumDataFile(rel string) bool {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 2 || parts[0] != CesiumDir {
		return false
	}
	return filepath.Ext(parts[len(parts)-1]) == DomainExtension
}

func copyFile(src, dst string) error {
	srcF, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcF.Close() }()
	info, err := srcF.Stat()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(dst), xfs.UserRWX); err != nil {
		return err
	}
	dstF, err := os.OpenFile(
		dst,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		info.Mode().Perm(),
	)
	if err != nil {
		return err
	}
	if _, err = io.Copy(dstF, srcF); err != nil {
		_ = dstF.Close()
		return err
	}
	return dstF.Close()
}

func isSubpath(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator)) &&
		rel != "."
}
