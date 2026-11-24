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
	"context"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// The suffixes of the files that a repair keeps aside.
const (
	damagedSuffix     = ".damaged"
	unrecoveredSuffix = ".unrecovered"
)

// write commits the repair of rep to the root of r: the log of a generation above every
// one in a name, with one Restart frame, then the head of that generation with the
// state of e. The header sector of the head is the commit point. Then it renames each
// damaged log and each unrecovered segment file aside, and syncs their directories.
func (e *engine) write(
	ctx context.Context,
	o *options,
	r *recovery,
	rep *Report,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g, err := nextGeneration(e.fs)
	if err != nil {
		return err
	}
	r.gen = g - 1
	prev, err := r.next(false)
	if err != nil {
		return err
	}
	if err = r.restartFrame(prev); err != nil {
		return err
	}
	c, err := checkpoint.New(checkpoint.Config{
		FS:       e.fs,
		DB:       r.db,
		LogSize:  log.FileSize,
		Prepared: func(uint64, error) {},
		Executor: o.executor,
		Parts:    r.parts,
		PartSize: o.partSize,
		Cache:    e.cache,
	})
	if err != nil {
		return err
	}
	ran := make(chan error, 1)
	go func() { ran <- c.Run(ctx) }()
	c.Head(g, e.state.ReplaySnapshot())
	if err = errors.Join(c.Close(), <-ran); err != nil {
		return err
	}
	rep.Repaired = true
	return e.setAside(rep)
}

// setAside renames each damaged log of rep with the suffix .damaged, and each
// unrecovered segment file with the suffix .unrecovered, then syncs the root and each
// channel directory that it changed.
func (e *engine) setAside(rep *Report) error {
	gens := set.New[uint64]()
	for _, d := range rep.Damage {
		gens.Add(d.Generation)
	}
	for _, g := range slices.Sorted(maps.Keys(gens)) {
		if err := e.fs.Rename(log.Name(g), log.Name(g)+damagedSuffix); err != nil {
			return err
		}
	}
	if err := e.fs.SyncDir(""); err != nil {
		return err
	}
	dirs := set.New[ChannelKey]()
	for _, f := range rep.Unrecovered {
		name := segmentName(f.Key, f.Segment, 0)
		if err := e.fs.Rename(name, name+unrecoveredSuffix); err != nil {
			return err
		}
		dirs.Add(f.Key)
	}
	for _, key := range slices.Sorted(maps.Keys(dirs)) {
		if err := e.fs.SyncDir(strconv.FormatUint(uint64(key), 10)); err != nil {
			return err
		}
	}
	return nil
}

// nextGeneration returns one above the generation of every head and log in the root
// fs, with any suffix.
func nextGeneration(fs xfs.FS) (uint64, error) {
	infos, err := fs.List("")
	if err != nil {
		return 0, err
	}
	var g uint64
	for _, info := range infos {
		hex, rest, ok := strings.Cut(info.Name(), ".")
		if !ok ||
			(!strings.HasPrefix(rest, "head") && !strings.HasPrefix(rest, "log")) {
			continue
		}
		if n, err := strconv.ParseUint(hex, 16, 64); err == nil {
			g = max(g, n)
		}
	}
	return g + 1, nil
}

// restartFrame writes one frame of a Restart entry at the start of the frames of the
// log of r.gen, which links to prev, and syncs it.
func (r *recovery) restartFrame(prev uint32) error {
	var en log.Entry
	en.Add(&log.Restart{})
	frame := log.Frame{
		Count:    1,
		Sequence: r.pos.Sequence,
		PrevCRC:  prev,
		Entries:  en.Bytes(),
	}
	b := frame.Append(nil, r.db, r.gen, log.FramesStart)
	f, err := r.fs.Open(log.Name(r.gen), os.O_RDWR)
	if err != nil {
		return err
	}
	if _, err = f.WriteAt(b, log.FramesStart); err == nil {
		err = f.Sync()
	}
	return errors.Join(err, f.Close())
}
