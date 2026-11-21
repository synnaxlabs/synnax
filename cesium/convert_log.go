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
	"github.com/synnaxlabs/cesium/internal/convert"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// convert reads the state of the old layout in the root, and writes the channel file of
// each of its channels for database db. Then it syncs each old file that the state
// names, each channel file, and each channel directory. It changes no old file.
func (e *engine) convert(o *options, db uint64) (*state.Snapshot, error) {
	intents, err := liveIntents(o.fs)
	if err != nil {
		return nil, err
	}
	snapshot, channels, err := convert.Read(convert.Config{
		Instrumentation: o.Instrumentation,
		FS:              o.fs,
		Codec:           o.metaCodec,
		Intents:         intents,
	})
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(channels))
	dirs := make([]string, 0, len(channels))
	for _, ch := range channels {
		name, err := writeChannel(o.fs, o.metaCodec, db, ch)
		if err != nil {
			return nil, err
		}
		names, dirs = append(names, name), append(dirs, keyToDirName(ch.Key))
	}
	for _, ch := range snapshot.Channels {
		for _, seg := range ch.Segments {
			names = append(names, segmentName(ch.Key, seg.ID, seg.File))
		}
	}
	return snapshot, e.budget.SyncFiles(o.fs, names, dirs)
}

// liveIntents returns the keys of the live delete intents in the intents file of the
// old layout, or none when the root holds no such file.
func liveIntents(fs xfs.FS) ([]uint32, error) {
	ok, err := fs.Exists(intentsFile)
	if err != nil || !ok {
		return nil, err
	}
	in, _, err := openIntents(fs)
	if err != nil {
		return nil, err
	}
	return in.keys.Slice(), in.file.Close()
}
