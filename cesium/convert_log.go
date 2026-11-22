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
	"slices"

	"github.com/synnaxlabs/cesium/internal/convert"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"golang.org/x/sync/errgroup"
)

// convert reads the state of the old layout in the root, and writes the channel file of
// each of its channels for database db, syncWidth at a time. Then it syncs each old
// file that the state names, each channel file, and each channel directory. It changes
// no old file.
func (e *engine) convert(o *options, db uint64) (*state.Snapshot, error) {
	intents, err := liveIntents(o.fs)
	if err != nil {
		return nil, err
	}
	plan, err := convert.Read(convert.Config{
		Instrumentation: o.Instrumentation,
		FS:              o.fs,
		Codec:           o.metaCodec,
		Intents:         intents,
	})
	if err != nil {
		return nil, err
	}
	snapshot, channels := plan.Snapshot, plan.Channels
	names, dirs := make([]string, len(channels)), make([]string, len(channels))
	var g errgroup.Group
	g.SetLimit(syncWidth)
	for i, ch := range channels {
		dirs[i] = keyToDirName(ch.Key)
		g.Go(func() (err error) {
			names[i], err = writeChannel(o.fs, o.metaCodec, db, ch)
			return err
		})
	}
	if err = g.Wait(); err != nil {
		return nil, err
	}
	var olds []string
	for _, ch := range snapshot.Channels {
		for _, seg := range ch.Segments {
			olds = append(olds, segmentName(ch.Key, seg.ID, seg.File))
		}
	}
	return snapshot, e.budget.SyncFiles(o.fs, slices.Concat(names, olds), dirs)
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
