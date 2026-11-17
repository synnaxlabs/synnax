// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"path"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/meta"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// MetaFiles returns the paths in fs of the metadata files of every channel of the
// database in fs. A copy of these files is a backup of the database without telemetry.
// No DB may be open on fs.
func MetaFiles(fs xfs.FS) ([]string, error) {
	keys, err := channelKeys(fs)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(keys))
	for _, key := range keys {
		file := path.Join(keyToDirName(key), meta.File)
		exists, err := fs.Exists(file)
		if err != nil {
			return nil, err
		}
		if exists {
			files = append(files, file)
		}
	}
	return files, nil
}

// RestoreMeta puts the metadata files in backup, at the paths that MetaFiles returns,
// back into the database in fs, and keeps the telemetry. It moves the directory of each
// channel that backup has no metadata for into the directory orphans in fs, which it
// creates when needed. No DB may be open on fs.
func RestoreMeta(fs, backup xfs.FS, orphans string) error {
	live, err := channelKeys(fs)
	if err != nil {
		return err
	}
	backedUp, err := channelKeys(backup)
	if err != nil {
		return err
	}
	known := set.New(backedUp...)
	unknown := lo.Reject(live, func(key ChannelKey, _ int) bool {
		return known.Contains(key)
	})
	if len(unknown) > 0 {
		if _, err = fs.Sub(orphans); err != nil {
			return err
		}
	}
	for _, key := range unknown {
		dir := keyToDirName(key)
		if err = fs.Rename(dir, path.Join(orphans, dir)); err != nil {
			return err
		}
	}
	for _, key := range backedUp {
		if err = copyMeta(backup, fs, keyToDirName(key)); err != nil {
			return err
		}
	}
	return nil
}

// channelKeys returns the keys of the channel directories in fs. It skips the
// directories of deleted channels and every other entry.
func channelKeys(fs xfs.FS) ([]ChannelKey, error) {
	infos, err := fs.List("")
	if err != nil {
		return nil, err
	}
	var keys []ChannelKey
	for _, info := range infos {
		if key, err := dirNameToKey(info.Name()); info.IsDir() && err == nil {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// copyMeta copies the metadata file of the channel directory dir from src to dst, when
// src has one.
func copyMeta(src, dst xfs.FS, dir string) error {
	file := path.Join(dir, meta.File)
	if exists, err := src.Exists(file); err != nil || !exists {
		return err
	}
	if _, err := dst.Sub(dir); err != nil {
		return err
	}
	return xfs.CopyFile(src, dst, file)
}
