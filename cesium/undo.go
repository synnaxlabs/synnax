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
	"fmt"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// checkIdentity compares the identity that the caller holds with the database of r,
// when snapshot holds its state. It refuses when the caller gives no identity, and
// undoes the database when the caller holds another. It returns the snapshot that the
// open goes on from, which is nil after an undo.
func checkIdentity(
	o *options,
	r *recovery,
	snapshot *state.Snapshot,
) (*state.Snapshot, error) {
	if snapshot == nil {
		return nil, nil
	}
	if !o.identified {
		return nil, errors.Newf(
			"the root holds database %016x, and the caller gave no identity: pass the "+
				"identity from DB.Identity, or 0 when none is held",
			r.db,
		)
	}
	if r.db == o.identity {
		return snapshot, nil
	}
	return nil, undo(o, r)
}

// undo moves aside the files of the pointer log layout in the root of r, whose database
// identity the caller does not hold: an undo restored the metadata from before the
// conversion, or the caller stopped before it stored the identity. Each file stays in
// its directory and gets the identity and the suffix .undone, so a later undo renames
// onto no earlier name. The heads go last since they mark the layout, and a crash
// before then leaves a root that Open undoes again.
func undo(o *options, r *recovery) error {
	suffix := fmt.Sprintf(".%016x.undone", r.db)
	infos, err := o.fs.List("")
	if err != nil {
		return err
	}
	var moved int
	for _, info := range infos {
		if !info.IsDir() {
			continue
		}
		if _, err = strconv.ParseUint(info.Name(), 10, 32); err != nil {
			continue
		}
		n, err := undoDir(o, info.Name(), suffix)
		if err != nil {
			return err
		}
		moved += n
	}
	var heads []string
	for _, info := range infos {
		name := info.Name()
		if _, ok := generationOf(name, ".head"); ok {
			heads = append(heads, name)
			continue
		}
		_, isLog := generationOf(name, ".log")
		if _, isPart := generationOf(name, ".part"); !isLog && !isPart {
			continue
		}
		if err = o.fs.Rename(name, name+suffix); err != nil {
			return err
		}
		moved++
	}
	if err = o.fs.SyncDir(""); err != nil {
		return err
	}
	for _, name := range heads {
		if err = o.fs.Rename(name, name+suffix); err != nil {
			return err
		}
	}
	if err = o.fs.SyncDir(""); err != nil {
		return err
	}
	o.L.Warn(
		"the database identity of the heads is not the one held: an undo restored "+
			"the metadata from before the conversion. Moved the files of the pointer "+
			"log layout aside, and the open starts again from the old files",
		zap.Uint64("held", o.identity),
		zap.Uint64("heads", r.db),
		zap.String("suffix", suffix),
		zap.Int("files", moved+len(heads)),
	)
	r.heads, r.logs = nil, nil
	return nil
}

// undoDir renames each segment and check file and the channel file of the channel
// directory dir with suffix, and syncs dir when it renamed one. It returns the count of
// files renamed. The check of a channel file holds the database identity, so a channel
// file that stayed would read as torn, and Sweep would remove it.
func undoDir(o *options, dir, suffix string) (int, error) {
	infos, err := o.fs.List(dir)
	if err != nil {
		return 0, err
	}
	var n int
	for _, info := range infos {
		name := dir + "/" + info.Name()
		if !strings.HasSuffix(name, ".seg") && !strings.HasSuffix(name, ".chk") &&
			info.Name() != channelFile {
			continue
		}
		if err = o.fs.Rename(name, name+suffix); err != nil {
			return n, err
		}
		n++
	}
	if n == 0 {
		return 0, nil
	}
	return n, o.fs.SyncDir(dir)
}
