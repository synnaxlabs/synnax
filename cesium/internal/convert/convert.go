// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package convert reads a root of the old layout and builds the state of its first
// head. It reads only numbered .domain files and the metadata files, writes nothing,
// and uses the bytes of an old file only when two reads of them agree.
package convert

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/migrate"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// Config is the configuration of Read.
type Config struct {
	alamos.Instrumentation
	// FS is the root of the database.
	FS xfs.FS
	// Codec decodes the meta.json file of each channel.
	Codec encoding.Decoder
	// Intents are the keys of the live delete intents of the old layout.
	Intents []uint32
}

// Read builds the state of a root of the old layout. Each old data file that a pointer
// names is a sealed version 0 segment, and each old index pointer takes one alignment
// number. Read also returns the channel of each directory whose meta.json names a
// channel with storage, in rising order of key. It skips a virtual channel, and a
// directory with no meta.json, whose key it logs. It returns an error that names the
// file when an old file has no two agreeing reads in three, when a pointer ends past
// its file, or when a data pointer holds samples with no stamps.
func Read(cfg Config) (*state.Snapshot, []channel.Channel, error) {
	r := &reader{
		Config: cfg,
		clocks: make(map[uint32]*clock),
		segs:   make(map[uint32]map[uint16][]state.Pointer),
		next:   1,
	}
	err := r.read()
	if r.open != nil {
		err = errors.Join(err, r.open.Close())
	}
	if err != nil {
		return nil, nil, err
	}
	return r.snapshot(), r.channels, nil
}

// reader holds what Read has read so far.
type reader struct {
	Config
	channels []channel.Channel
	clocks   map[uint32]*clock
	// segs holds the pointers of each key, by old file number.
	segs map[uint32]map[uint16][]state.Pointer
	// next is the next alignment number.
	next uint64
	// open is the file of the last read, and openName its name.
	open     xfs.File
	openName string
}

// read reads the channels, then numbers the stretches of each index, then splits the
// pointers of each data key.
func (r *reader) read() error {
	if err := r.list(); err != nil {
		return err
	}
	for _, ch := range r.channels {
		if ch.IsIndex {
			if err := r.index(ch); err != nil {
				return err
			}
		}
	}
	if r.next > uint64(telem.LeadingDomain) {
		return errors.Newf(
			"the old layout has %d stretches, above %d",
			r.next-1,
			telem.LeadingDomain-1,
		)
	}
	for _, ch := range r.channels {
		if !ch.IsIndex {
			if err := r.data(ch); err != nil {
				return err
			}
		}
	}
	return nil
}

// list reads the meta.json file of each directory that a key names.
func (r *reader) list() error {
	infos, err := r.FS.List("")
	if err != nil {
		return err
	}
	for _, info := range infos {
		key, err := strconv.ParseUint(info.Name(), 10, 32)
		if err != nil || !info.IsDir() {
			continue
		}
		name := info.Name() + "/" + meta.File
		ok, err := r.FS.Exists(name)
		if err != nil {
			return err
		}
		if !ok {
			r.L.Warn(
				"skipped a channel directory with no meta.json",
				zap.Uint64("key", key),
			)
			continue
		}
		ch, ok, err := r.channel(uint32(key), name)
		if err != nil {
			return err
		}
		if ok {
			r.channels = append(r.channels, ch)
		}
	}
	slices.SortFunc(r.channels, func(a, b channel.Channel) int {
		return cmp.Compare(a.Key, b.Key)
	})
	return nil
}

// channel decodes the meta.json file name of key and migrates it as the old open does.
// It returns false for a channel that the old open ignores, and for a virtual channel.
func (r *reader) channel(key uint32, name string) (channel.Channel, bool, error) {
	info, err := r.FS.Stat(name)
	if err != nil {
		return channel.Channel{}, false, err
	}
	b, err := r.agreed(name, 0, int(info.Size()))
	if err != nil {
		return channel.Channel{}, false, err
	}
	var ch channel.Channel
	if err = r.Codec.Decode(b, &ch); err != nil {
		return channel.Channel{}, false, errors.Wrapf(err, "decode of %s", name)
	}
	sub, err := r.FS.Sub(strconv.FormatUint(uint64(key), 10))
	if err != nil {
		return channel.Channel{}, false, err
	}
	s := migrate.Migrate(migrate.DBState{Channel: ch, FS: sub})
	if s.ShouldIgnoreChannel || s.Channel.Virtual {
		return channel.Channel{}, false, nil
	}
	switch err = s.Channel.Validate(); {
	case err != nil:
		return channel.Channel{}, false, errors.Wrapf(err, "%s", name)
	case s.Channel.Key != key:
		return channel.Channel{}, false, errors.Newf("%s holds key %d", name, ch.Key)
	case telem.CodeOf(s.Channel.DataType) == telem.DataTypeCodeUnknown:
		return channel.Channel{}, false, errors.Newf(
			"%s has data type %s, which has no type code",
			name,
			s.Channel.DataType,
		)
	}
	return s.Channel, true, nil
}

// pointerSize is the size of a pointer in an old index.domain file.
const pointerSize = 26

// pointer is a pointer of an old index.domain file. Its time range does not hold
// stamps: end is exclusive, and the old writer can start or end a range between
// samples.
type pointer struct {
	start, end   telem.TimeStamp
	file         uint16
	offset, size uint32
}

// pointers reads the pointers of key, in rising order of start. It returns an error
// when a pointer ends past its file or holds no whole number of samples of density.
func (r *reader) pointers(key uint32, density telem.Density) ([]pointer, error) {
	name := fmt.Sprintf("%d/index.domain", key)
	info, err := r.FS.Stat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b, err := r.agreed(name, 0, int(info.Size()))
	if err != nil {
		return nil, err
	}
	ptrs := make([]pointer, 0, len(b)/pointerSize)
	sizes := make(map[uint16]int64)
	for i := 0; i+pointerSize <= len(b); i += pointerSize {
		p := pointer{
			start:  telem.TimeStamp(binary.LittleEndian.Uint64(b[i:])),
			end:    telem.TimeStamp(binary.LittleEndian.Uint64(b[i+8:])),
			file:   binary.LittleEndian.Uint16(b[i+16:]),
			offset: binary.LittleEndian.Uint32(b[i+18:]),
			size:   binary.LittleEndian.Uint32(b[i+22:]),
		}
		if p.size == 0 {
			continue
		}
		file := fileName(key, p.file)
		size, ok := sizes[p.file]
		if !ok {
			if size, err = r.fileSize(file); err != nil {
				return nil, err
			}
			sizes[p.file] = size
		}
		switch {
		case int64(p.offset)+int64(p.size) > size:
			return nil, errors.Newf(
				"a pointer of key %d ends at %d, past the length %d of %s",
				key,
				int64(p.offset)+int64(p.size),
				size,
				file,
			)
		case density != 0 && int64(p.size)%int64(density) != 0:
			return nil, errors.Newf(
				"a pointer of key %d at %d of %s holds %d bytes, not whole samples",
				key,
				p.offset,
				file,
				p.size,
			)
		}
		ptrs = append(ptrs, p)
	}
	slices.SortFunc(
		ptrs,
		func(a, b pointer) int { return cmp.Compare(a.start, b.start) },
	)
	return ptrs, nil
}

// fileSize returns the length of name, or zero when it does not exist.
func (r *reader) fileSize(name string) (int64, error) {
	info, err := r.FS.Stat(name)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// clockOf returns the key of the index that times the samples of ch.
func clockOf(ch channel.Channel) uint32 {
	if ch.IsIndex {
		return ch.Key
	}
	return ch.Index
}

// fileName returns the name of old data file n of key.
func fileName(
	key uint32,
	n uint16,
) string {
	return fmt.Sprintf("%d/%d.domain", key, n)
}

// agreed reads n bytes at off of the file name, and returns them once two reads agree.
// With no two agreeing reads in three, it returns an error that wraps place.ErrCorrupt
// and names the file.
func (r *reader) agreed(name string, off int64, n int) ([]byte, error) {
	if r.openName != name {
		if r.open != nil {
			if err := r.open.Close(); err != nil {
				return nil, err
			}
			r.open = nil
		}
		f, err := r.FS.Open(name, os.O_RDONLY)
		if err != nil {
			return nil, err
		}
		r.open, r.openName = f, name
	}
	reads := make([][]byte, 0, 3)
	for range 3 {
		b := make([]byte, n)
		if got, err := r.open.ReadAt(b, off); got < n {
			return nil, errors.Wrapf(
				cmp.Or(err, io.ErrUnexpectedEOF),
				"read of %d bytes at %d of %s",
				n,
				off,
				name,
			)
		}
		for _, prev := range reads {
			if bytes.Equal(b, prev) {
				return b, nil
			}
		}
		reads = append(reads, b)
	}
	return nil, errors.Wrapf(
		place.ErrCorrupt,
		"%d bytes at %d of %s have no two agreeing reads in three",
		n,
		off,
		name,
	)
}

// add appends ps to the pointers of old file n of key.
func (r *reader) add(key uint32, n uint16, ps ...state.Pointer) {
	files := r.segs[key]
	if files == nil {
		files = make(map[uint16][]state.Pointer)
		r.segs[key] = files
	}
	files[n] = append(files[n], ps...)
}

// snapshot returns the state of what r read: the segments of each key in rising order
// of old file number, from identity 1.
func (r *reader) snapshot() *state.Snapshot {
	s := &state.Snapshot{
		Intents: slices.Compact(slices.Sorted(slices.Values(r.Intents))),
	}
	var id uint64
	for _, ch := range r.channels {
		files := r.segs[ch.Key]
		if len(files) == 0 {
			continue
		}
		c := &state.Channel{Key: ch.Key}
		for _, n := range slices.Sorted(maps.Keys(files)) {
			ptrs := files[n]
			slices.SortFunc(ptrs, func(a, b state.Pointer) int {
				return cmp.Compare(a.Offset, b.Offset)
			})
			last := ptrs[len(ptrs)-1]
			id++
			c.Segments = append(c.Segments, state.Segment{
				ID:       id,
				Clock:    clockOf(ch),
				File:     n,
				DataType: uint8(telem.CodeOf(ch.DataType)),
				Sealed:   true,
				Length:   last.Offset + last.Length,
				Pointers: ptrs,
			})
		}
		s.Channels = append(s.Channels, c)
	}
	s.Limits = state.Limits{Segments: id + 1, Alignments: r.next}
	return s
}
