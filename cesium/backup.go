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

	"github.com/synnaxlabs/cesium/internal/meta"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// BackupMeta copies the metadata of every channel of the database in fs to backup,
// without telemetry. No DB may be open on fs.
func BackupMeta(fs, backup xfs.FS) error {
	keys, err := channelKeys(fs)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err = copyMeta(fs, backup, keyToDirName(key)); err != nil {
			return err
		}
	}
	return nil
}

// RestoreMeta puts the metadata that BackupMeta copied to backup back into the database
// in fs, and keeps the telemetry. It moves the directory of each channel that backup
// has no metadata for into the directory orphans in fs, which it creates when needed.
// No DB may be open on fs.
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
	for _, key := range live {
		if known.Contains(key) {
			continue
		}
		if _, err = fs.Sub(orphans); err != nil {
			return err
		}
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
