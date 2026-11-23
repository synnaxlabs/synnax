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
	_, err := writeFile(fs, Name(generation), b)
	return err
}

// writeFile writes the checkpoint file b to name in the root fs, as Write does a head.
// On an error, it returns true when no file of the attempt is left: the create failed,
// or the remove and the sync of the root after it succeeded.
func writeFile(fs xfs.FS, name string, b []byte) (gone bool, err error) {
	f, err := fs.Open(name, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		return true, err
	}
	if err = errors.Join(write(fs, f, b), f.Close()); err == nil {
		return false, nil
	}
	if rmErr := fs.Remove(name); rmErr != nil {
		return false, errors.Join(err, rmErr)
	}
	if syncErr := fs.SyncDir(""); syncErr != nil {
		return false, errors.Join(err, syncErr)
	}
	return true, err
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
// for a format this binary does not know, and an error that wraps place.ErrCorrupt for
// any other check that fails.
func Read(fs xfs.FS, generation uint64) (Header, *state.Snapshot, error) {
	h, bd, err := headFormat.read(fs, Name(generation), true)
	if err != nil {
		return h, nil, err
	}
	h.Parts = bd.parts
	return h, &state.Snapshot{
		Sequence: h.Sequence,
		Limits:   h.Limits,
		Channels: bd.channels,
		Intents:  bd.intents,
	}, nil
}

// readPart reads and checks the part p that a head names in the root fs of database db,
// as Read does a head. It returns the channels of the part, and an error that wraps
// place.ErrCorrupt when the part is not the one that the head names.
func readPart(fs xfs.FS, db uint64, p Part) ([]*state.Channel, error) {
	h, bd, err := partFormat.read(fs, partName(p.ID), true)
	if errors.Is(err, ErrIncomplete) {
		err = errors.Wrap(place.ErrCorrupt, err.Error())
	}
	if err != nil {
		return nil, err
	}
	if h.DB != db || h.Generation != p.ID || h.Sequence != p.Generation ||
		h.Length != p.Length || h.CRC != p.CRC {
		return nil, partFormat.corrupt("part %d is not the part that its head names",
			p.ID)
	}
	return bd.channels, nil
}

// read reads and checks the file name, as Read does a head. It parses every record when
// whole is true, and else only the sections and the part records.
func (f *format) read(
	fs xfs.FS,
	name string,
	whole bool,
) (h Header, bd body, err error) {
	file, err := fs.Open(name, os.O_RDONLY)
	if err != nil {
		return h, bd, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return h, bd, err
	}
	b := make([]byte, info.Size())
	reread := func(lo, hi uint64) error {
		if lo == hi {
			return nil
		}
		_, err := file.ReadAt(b[lo:hi], int64(lo))
		if errors.Is(err, io.EOF) {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	if err = reread(0, uint64(len(b))); err != nil {
		return h, bd, err
	}
	h, err = f.decodeSector(b)
	if errors.Is(err, place.ErrCorrupt) || errors.Is(err, ErrIncomplete) {
		if err = reread(0, min(SectorSize, uint64(len(b)))); err != nil {
			return h, bd, err
		}
		h, err = f.decodeSector(b)
	}
	if err != nil {
		return h, bd, err
	}
	if err = f.checkChunks(b, h, reread); err != nil {
		return h, bd, err
	}
	if whole {
		bd, err = f.parse(b, h)
		return h, bd, err
	}
	secs, err := f.sectionsOf(b, h)
	if err == nil {
		bd.parts, err = f.parseParts(b, secs)
	}
	return h, bd, err
}

// checkChunks checks the chunk table and each chunk of b, the file of h. It reads a
// range that fails its check again with reread, and checks it once more.
func (f *format) checkChunks(
	b []byte,
	h Header,
	reread func(lo, hi uint64) error,
) error {
	err := again(f.checkChunkTable(b, h), func() error {
		if err := reread(h.chunks, h.Length); err != nil {
			return err
		}
		return f.checkChunkTable(b, h)
	})
	for i := uint64(0); err == nil && i < chunkCount(h); i++ {
		lo, hi := chunkRange(h, i)
		err = again(f.checkChunk(b, h, i), func() error {
			if err := reread(lo, hi); err != nil {
				return err
			}
			return f.checkChunk(b, h, i)
		})
	}
	return err
}

// readParts reads and checks the head of generation in the root fs, as Read does, and
// returns the parts that it names. It parses no channel record.
func readParts(fs xfs.FS, generation uint64) (set.Set[uint64], error) {
	_, bd, err := headFormat.read(fs, Name(generation), false)
	if err != nil {
		return nil, err
	}
	ids := set.New[uint64]()
	for _, p := range bd.parts {
		ids.Add(p.ID)
	}
	return ids, nil
}

// again returns err, or the result of retry when err is a failed check.
func again(err error, retry func() error) error {
	if errors.Is(err, place.ErrCorrupt) {
		return retry()
	}
	return err
}

// prepareLog makes the file of the log of generation next in the root fs when it is
// missing or not size bytes long, and removes a leftover head of generation next. It
// syncs a file of size bytes that it finds, and then the root, as it does a file that
// it makes.
func prepareLog(fs xfs.FS, next uint64, size int64) error {
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

// clean removes from the root fs what the switch to generation live no longer needs:
// every head and log older than the newest older head that passes its checks and whose
// logs all exist from its generation to live, and each part that neither of the two
// heads names. liveParts holds the identities of the parts that the head of live names,
// or is nil to read them from the head when the root holds a part. It tries every
// removal, then syncs the root, and returns the errors of the removals that failed,
// which the next switch tries again.
func clean(fs xfs.FS, live uint64, liveParts set.Set[uint64]) error {
	entries, err := fs.List("")
	if err != nil {
		return err
	}
	heads, logs, parts := set.Set[uint64]{}, set.Set[uint64]{}, set.Set[uint64]{}
	for _, e := range entries {
		if g, ok := generationOf(e.Name(), ".head"); ok {
			heads.Add(g)
		} else if g, ok = generationOf(e.Name(), ".log"); ok {
			logs.Add(g)
		} else if id, ok := generationOf(e.Name(), ".part"); ok {
			parts.Add(id)
		}
	}
	keep, named := live, set.New[uint64]()
	for g := live - 1; g >= 1 && logs.Contains(g); g-- {
		if !heads.Contains(g) {
			continue
		}
		if older, err := readParts(fs, g); err == nil {
			keep, named = g, older
			break
		}
	}
	var errs []error
	if len(parts) > 0 && liveParts == nil {
		if liveParts, err = readParts(fs, live); err != nil {
			return err
		}
	}
	named.Add(liveParts.Slice()...)
	for _, id := range slices.Sorted(maps.Keys(parts)) {
		if !named.Contains(id) {
			errs = append(errs, fs.Remove(partName(id)))
		}
	}
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
