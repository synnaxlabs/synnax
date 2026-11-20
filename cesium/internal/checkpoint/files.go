// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/set"
)

// Write writes head bytes b of generation, from Encode, to the root fs: it creates the
// file, sets its size, writes and syncs the body, syncs the root, then writes and syncs
// the header sector. That last sync is the commit point. On an error, Write removes the
// file with no further sync of it, and syncs the root.
func Write(fs xfs.FS, generation uint64, b []byte) error {
	name := Name(generation)
	f, err := fs.Open(name, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		return err
	}
	err = errors.Join(write(fs, f, b), f.Close())
	if err == nil {
		return nil
	}
	if rmErr := fs.Remove(name); rmErr != nil {
		return errors.Join(err, rmErr)
	}
	return errors.Join(err, fs.SyncDir(""))
}

func write(fs xfs.FS, f xfs.File, b []byte) error {
	if err := f.Truncate(int64(len(b))); err != nil {
		return err
	}
	if _, err := f.WriteAt(b[BodyStart:], BodyStart); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := fs.SyncDir(""); err != nil {
		return err
	}
	if _, err := f.WriteAt(b[:SectorSize], 0); err != nil {
		return err
	}
	return f.Sync()
}

// Read reads and checks the head of generation in the root fs. A header sector, chunk
// table, or chunk that fails its check is read a second time, and only a second failure
// of the same range decides. It reads no byte between the header sector and the body,
// which no checksum covers. It returns ErrIncomplete for a zero header sector, an error
// that wraps ErrUnknown for a format this binary does not know, and an error that wraps
// place.ErrCorrupt for any other check that fails.
func Read(fs xfs.FS, generation uint64) (h Header, s *state.Snapshot, err error) {
	f, err := fs.Open(Name(generation), os.O_RDONLY)
	if err != nil {
		return h, nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return h, nil, err
	}
	b := make([]byte, info.Size())
	reread := func(lo, hi uint64) error {
		if lo == hi {
			return nil
		}
		_, err := f.ReadAt(b[lo:hi], int64(lo))
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	if err = reread(0, uint64(len(b))); err != nil {
		return h, nil, err
	}
	h, err = decodeSector(b)
	if errors.Is(err, place.ErrCorrupt) || errors.Is(err, ErrIncomplete) {
		if err = reread(0, min(SectorSize, uint64(len(b)))); err != nil {
			return h, nil, err
		}
		h, err = decodeSector(b)
	}
	if err != nil {
		return h, nil, err
	}
	err = again(checkChunkTable(b, h), func() error {
		if err := reread(h.chunks, h.Length); err != nil {
			return err
		}
		return checkChunkTable(b, h)
	})
	for i := uint64(0); err == nil && i < chunkCount(h); i++ {
		lo, hi := chunkRange(h, i)
		err = again(checkChunk(b, h, i), func() error {
			if err := reread(lo, hi); err != nil {
				return err
			}
			return checkChunk(b, h, i)
		})
	}
	if err != nil {
		return h, nil, err
	}
	s, err = parse(b, h)
	return h, s, err
}

// again returns err, or the result of retry when err is a failed check.
func again(err error, retry func() error) error {
	if errors.Is(err, place.ErrCorrupt) {
		return retry()
	}
	return err
}

// Prepare makes the file of the log of generation next in the root fs when it is
// missing or not size bytes long, and removes a leftover head of generation next. It
// syncs a file of size bytes that it finds, and then the root, as it does a file that
// it makes.
func Prepare(fs xfs.FS, next uint64, size int64) error {
	if err := fs.Remove(Name(next)); err != nil {
		return err
	}
	info, err := fs.Stat(log.Name(next))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err != nil || info.Size() != size {
		return log.Prepare(fs, next, size)
	}
	f, err := fs.Open(log.Name(next), os.O_RDWR)
	if err != nil {
		return err
	}
	if err = errors.Join(f.Sync(), f.Close()); err != nil {
		return err
	}
	return fs.SyncDir("")
}

// Clean removes from the root fs what the switch to generation live no longer needs:
// every head and log older than the newest older head that Read takes and whose logs
// all exist from its generation to live. It tries every removal, then syncs the root,
// and returns the
// errors of the removals that failed, which the next switch tries again.
func Clean(fs xfs.FS, live uint64) error {
	entries, err := fs.List("")
	if err != nil {
		return err
	}
	heads, logs := set.Set[uint64]{}, set.Set[uint64]{}
	for _, e := range entries {
		if g, ok := generationOf(e.Name(), ".head"); ok {
			heads.Add(g)
		} else if g, ok = generationOf(e.Name(), ".log"); ok {
			logs.Add(g)
		}
	}
	keep := live
	for g := live - 1; g >= 1 && logs.Contains(g); g-- {
		if !heads.Contains(g) {
			continue
		}
		if _, _, err := Read(fs, g); err == nil {
			keep = g
			break
		}
	}
	var errs []error
	for _, g := range slices.Sorted(maps.Keys(heads)) {
		if g < keep {
			errs = append(errs, fs.Remove(Name(g)))
		}
	}
	for _, g := range slices.Sorted(maps.Keys(logs)) {
		if g < keep {
			errs = append(errs, fs.Remove(log.Name(g)))
		}
	}
	return errors.Join(errors.Join(errs...), fs.SyncDir(""))
}

// generationOf returns the generation of a file name of 16 hex digits and suffix.
func generationOf(name, suffix string) (uint64, bool) {
	hex, ok := strings.CutSuffix(name, suffix)
	if !ok || len(hex) != 16 {
		return 0, false
	}
	g, err := strconv.ParseUint(hex, 16, 64)
	return g, err == nil
}
