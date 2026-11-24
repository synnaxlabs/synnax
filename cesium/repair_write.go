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

// write commits the repair of rep to the root of r. First it renames each unrecovered
// segment file aside and syncs its directory, so no orphan pass after a crash removes
// a file that the repair keeps. Then it writes the log of a generation above every head
// and log with no suffix, with one Restart frame, then the head of that generation with
// the state of e. The header sector of the head is the commit point. Last it renames
// each damaged log aside and syncs the root.
func (e *engine) write(
	ctx context.Context,
	o *options,
	r *recovery,
	rep *Report,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := e.setAsideFiles(rep); err != nil {
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
	return e.setAsideLogs(rep)
}

// setAsideLogs renames each damaged log of rep with the suffix .damaged, then syncs the
// root. A rename never replaces a file: a name in use takes the next free .<k> (D-376).
// It sets the name used in rep.
func (e *engine) setAsideLogs(rep *Report) error {
	names := make(map[uint64]string)
	for _, d := range rep.Damage {
		if _, ok := names[d.Generation]; ok {
			continue
		}
		name, err := e.renameAside(log.Name(d.Generation), damagedSuffix)
		if err != nil {
			return err
		}
		names[d.Generation] = name
	}
	for i := range rep.Damage {
		rep.Damage[i].Name = names[rep.Damage[i].Generation]
	}
	return e.fs.SyncDir("")
}

// setAsideFiles renames each unrecovered segment file of rep with the suffix
// .unrecovered, as setAsideLogs does, then syncs each channel directory that it
// changed. It sets the name used in rep.
func (e *engine) setAsideFiles(rep *Report) error {
	dirs := set.New[ChannelKey]()
	for i, f := range rep.Unrecovered {
		name, err := e.renameAside(segmentName(f.Key, f.Segment, 0), unrecoveredSuffix)
		if err != nil {
			return err
		}
		rep.Unrecovered[i].Name = name
		dirs.Add(f.Key)
	}
	for _, key := range slices.Sorted(maps.Keys(dirs)) {
		if err := e.fs.SyncDir(strconv.FormatUint(uint64(key), 10)); err != nil {
			return err
		}
	}
	return nil
}

// renameAside renames name to name with suffix, or with suffix and the next free .<k>
// when that name is in use, and returns the name that it used.
func (e *engine) renameAside(name, suffix string) (string, error) {
	to := name + suffix
	for k := 1; ; k++ {
		used, err := e.fs.Exists(to)
		if err != nil {
			return "", err
		}
		if !used {
			break
		}
		to = name + suffix + "." + strconv.Itoa(k)
	}
	return to, e.fs.Rename(name, to)
}

// nextGeneration returns one above the generation of every head and log in the root
// fs with no suffix (D-376).
func nextGeneration(fs xfs.FS) (uint64, error) {
	infos, err := fs.List("")
	if err != nil {
		return 0, err
	}
	var g uint64
	for _, info := range infos {
		hex, ext, ok := strings.Cut(info.Name(), ".")
		if !ok || (ext != "head" && ext != "log") {
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
