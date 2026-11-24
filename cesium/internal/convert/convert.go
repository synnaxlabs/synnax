// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package convert reads a root of the old layout and builds the state of its first
// head. It reads only numbered .domain files, the file that an old compaction renamed a
// missing one to, and the metadata files. It writes nothing, and uses the bytes of an
// old file only when two reads of them agree.
package convert

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path"
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
	"golang.org/x/sync/errgroup"
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
// channel with storage. It skips a virtual channel, and a directory with no meta.json,
// whose key it logs. It cuts, and logs, a pointer that ends past its file, the newest
// pointer of an index after its last stamp inside its time range, and data samples
// with no stamps. It returns an error that names the file when an old file has no two
// agreeing reads in three.
func Read(cfg Config) (*Plan, error) {
	r := &reader{
		Config: cfg,
		ptrs:   make(map[uint32][]pointer),
		clocks: make(map[uint32]*clock),
		segs:   make(map[uint32]map[uint16][]state.Pointer),
		next:   1,
	}
	if err := r.read(); err != nil {
		return nil, err
	}
	return &Plan{Snapshot: r.snapshot(), Channels: r.channels}, nil
}

// Plan is what Read builds from a root of the old layout.
type Plan struct {
	// Snapshot is the state of the first head.
	Snapshot *state.Snapshot
	// Channels are the channels with storage, in rising order of key.
	Channels []channel.Channel
}

// reader holds what Read has read so far.
type reader struct {
	Config
	channels []channel.Channel
	// ptrs holds the old pointers of each channel, in rising order of start.
	ptrs   map[uint32][]pointer
	clocks map[uint32]*clock
	// segs holds the pointers of each key, by old file number.
	segs map[uint32]map[uint16][]state.Pointer
	// next is the next alignment number.
	next uint64
}

// read reads the channels, then numbers the stretches of each index, then splits the
// pointers of each data key.
func (r *reader) read() error {
	if err := r.list(); err != nil {
		return err
	}
	if err := r.indexes(); err != nil {
		return err
	}
	if r.next > uint64(telem.LeadingDomain) {
		return errors.Newf(
			"the old layout has %d stretches, above %d",
			r.next-1,
			telem.LeadingDomain-1,
		)
	}
	return r.datas()
}

// each runs f for each index below n, readWidth at a time, and returns the error of the
// lowest index that failed.
func each(n int, f func(i int) error) error {
	errs := make([]error, n)
	var g errgroup.Group
	g.SetLimit(readWidth)
	for i := range n {
		g.Go(func() error {
			errs[i] = f(i)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// readWidth is the most channel directories that Read reads at once. On a cold disk
// each small old file is a random read, and many at once keep the disk busy.
const readWidth = 32

// dir is what Read reads of one channel directory.
type dir struct {
	key uint32
	ch  channel.Channel
	// kept is true for a channel with storage, and missing for a directory with no
	// meta.json.
	kept, missing bool
	ptrs          []pointer
}

// list reads the channel and the old pointers of each directory that a key names,
// readWidth directories at a time.
func (r *reader) list() error {
	infos, err := r.FS.List("")
	if err != nil {
		return err
	}
	var dirs []dir
	for _, info := range infos {
		if key, err := strconv.ParseUint(
			info.Name(),
			10,
			32,
		); err == nil &&
			info.IsDir() {
			dirs = append(dirs, dir{key: uint32(key)})
		}
	}
	if err = each(
		len(dirs),
		func(i int) error { return r.readDir(&dirs[i]) },
	); err != nil {
		return err
	}
	for _, d := range dirs {
		switch {
		case d.missing:
			r.L.Warn(
				"skipped a channel directory with no meta.json",
				zap.Uint32("key", d.key),
			)
		case d.kept:
			r.channels, r.ptrs[d.key] = append(r.channels, d.ch), d.ptrs
		}
	}
	slices.SortFunc(r.channels, func(a, b channel.Channel) int {
		return cmp.Compare(a.Key, b.Key)
	})
	return nil
}

// readDir reads the channel of d and its old pointers.
func (r *reader) readDir(d *dir) (err error) {
	name := path.Join(strconv.FormatUint(uint64(d.key), 10), meta.File)
	ok, err := r.FS.Exists(name)
	if d.missing = !ok; !ok || err != nil {
		return err
	}
	if d.ch, d.kept, err = r.channel(d.key, name); !d.kept || err != nil {
		return err
	}
	var density telem.Density
	switch {
	case d.ch.IsIndex:
		density = stampSize
	case !d.ch.DataType.IsVariable():
		density = d.ch.DataType.Density()
	}
	d.ptrs, err = r.pointers(d.key, density, d.ch.IsIndex)
	return err
}

// channel decodes the meta.json file name of key and migrates it as the old open does.
// It returns false for a channel that the old open ignores, and for a virtual channel.
func (r *reader) channel(key uint32, name string) (channel.Channel, bool, error) {
	b, err := r.readFile(name)
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
// samples. cut is true when the pointer ended past its file, so it can hold fewer
// samples than its range has stamps. temp is true when its old file is missing, so it
// reads the file that an old compaction renamed aside.
type pointer struct {
	start, end   telem.TimeStamp
	file         uint16
	offset, size uint32
	cut, temp    bool
}

// tempSuffix ends the name that an old compaction gives a data file while it moves the
// compacted copy into place.
const tempSuffix = "_temp"

// name returns the name of the file that p reads in the directory of key.
func (p pointer) name(key uint32) string {
	if p.temp {
		return fileName(key, p.file) + tempSuffix
	}
	return fileName(key, p.file)
}

// pointers reads the pointers of key, in rising order of start. A pointer that ends
// past its file keeps the whole samples inside the file. One that keeps none goes, or
// for an index stays as a mark with no stamps over its time range. A density of zero is
// a variable type. It returns an error when a pointer holds no whole number of samples
// of density.
func (r *reader) pointers(
	key uint32,
	density telem.Density,
	index bool,
) (_ []pointer, err error) {
	name := fmt.Sprintf("%d/index.domain", key)
	b, err := r.readFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	h := &handle{fs: r.FS, l: r.L}
	defer func() { err = errors.Join(err, h.close()) }()
	ptrs := make([]pointer, 0, len(b)/pointerSize)
	files := make(map[uint16]oldFile)
	for i := 0; i+pointerSize <= len(b); i += pointerSize {
		p := decodePointer(b[i:])
		if p.size == 0 {
			continue
		}
		file := fileName(key, p.file)
		f, ok := files[p.file]
		if !ok {
			if f, err = r.oldFile(file); err != nil {
				return nil, err
			}
			files[p.file] = f
		}
		if p.temp = f.temp; p.temp {
			file += tempSuffix
		}
		size := f.size
		if density != 0 && int64(p.size)%int64(density) != 0 {
			return nil, errors.Newf(
				"a pointer of key %d at %d of %s holds %d bytes, not whole samples",
				key,
				p.offset,
				file,
				p.size,
			)
		}
		if int64(p.offset)+int64(p.size) > size {
			if p, err = h.cutPastFile(key, p, size, density); err != nil {
				return nil, err
			}
			if p.size == 0 && !index {
				continue
			}
		}
		ptrs = append(ptrs, p)
	}
	slices.SortFunc(
		ptrs,
		func(a, b pointer) int { return cmp.Compare(a.start, b.start) },
	)
	return ptrs, nil
}

// decodePointer decodes the pointer at the start of b, an index.domain record.
func decodePointer(b []byte) pointer {
	return pointer{
		start:  telem.TimeStamp(binary.LittleEndian.Uint64(b)),
		end:    telem.TimeStamp(binary.LittleEndian.Uint64(b[8:])),
		file:   binary.LittleEndian.Uint16(b[16:]),
		offset: binary.LittleEndian.Uint32(b[18:]),
		size:   binary.LittleEndian.Uint32(b[22:]),
	}
}

// oldFile is an old data file that pointers name.
type oldFile struct {
	size int64
	temp bool
}

// oldFile returns the old data file name. When the file is missing and the file that
// an old compaction renamed it to exists, that file holds the bytes of name. The old
// compaction renames name aside before it moves the compacted copy into place, and
// never syncs. A missing file has length zero.
func (r *reader) oldFile(name string) (oldFile, error) {
	info, err := r.FS.Stat(name)
	if err == nil {
		return oldFile{size: info.Size()}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return oldFile{}, err
	}
	info, err = r.FS.Stat(name + tempSuffix)
	if errors.Is(err, os.ErrNotExist) {
		return oldFile{}, nil
	}
	if err != nil {
		return oldFile{}, err
	}
	return oldFile{size: info.Size(), temp: true}, nil
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

// readFile returns the bytes of the file name once two whole reads of it agree. It is
// safe for concurrent use.
func (r *reader) readFile(name string) ([]byte, error) {
	info, err := r.FS.Stat(name)
	if err != nil {
		return nil, err
	}
	f, err := r.FS.Open(name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	b, err := agreed(f, name, 0, int(info.Size()))
	return b, errors.Join(err, f.Close())
}

// handle reads old files of fs, and keeps the last file it read open until the next one
// or close. It logs each cut to l. A handle serves one goroutine.
type handle struct {
	fs   xfs.FS
	l    *alamos.Logger
	f    xfs.File
	name string
}

// cutPastFile cuts p, a pointer of key that ends past size, the length of its file, to
// the whole samples of density inside the file, and logs the cut. A cut to no sample
// leaves p.size zero.
func (h *handle) cutPastFile(
	key uint32,
	p pointer,
	size int64,
	density telem.Density,
) (pointer, error) {
	file := p.name(key)
	kept, err := h.inside(file, int64(p.offset), size, density)
	if err != nil {
		return p, err
	}
	h.l.Warn(
		"cut a pointer that ends past its file to the whole samples inside it",
		zap.Uint32("key", key),
		zap.String("file", file),
		zap.Int64("end", int64(p.offset)+int64(p.size)),
		zap.Int64("length", size),
		zap.Int64("kept", kept),
	)
	p.size, p.cut = uint32(kept), true
	return p, nil
}

// inside returns the bytes of the whole samples of density from off of the file name
// that end by size, its length. A density of zero is a variable type, whose length
// prefixes it walks.
func (h *handle) inside(
	name string,
	off, size int64,
	density telem.Density,
) (int64, error) {
	switch {
	case off >= size:
		return 0, nil
	case density != 0:
		return (size - off) / int64(density) * int64(density), nil
	}
	end := uint32(min(size, math.MaxUint32))
	_, kept, err := h.walk(name, uint32(off), end, math.MaxUint32)
	return int64(kept), err
}

// agreed returns n bytes at off of the file name as agreed does.
func (h *handle) agreed(name string, off int64, n int) ([]byte, error) {
	if h.name != name {
		if err := h.close(); err != nil {
			return nil, err
		}
		f, err := h.fs.Open(name, os.O_RDONLY)
		if err != nil {
			return nil, err
		}
		h.f, h.name = f, name
	}
	return agreed(h.f, name, off, n)
}

// close closes the file that h keeps open.
func (h *handle) close() error {
	if h.f == nil {
		return nil
	}
	f := h.f
	h.f, h.name = nil, ""
	return f.Close()
}

// agreed reads n bytes at off of f, the file name, and returns them once two reads
// agree. With no two agreeing reads in three, it returns an error that wraps
// place.ErrCorrupt and names the file.
func agreed(f xfs.File, name string, off int64, n int) ([]byte, error) {
	reads := make([][]byte, 0, 3)
	for range 3 {
		b := make([]byte, n)
		if got, err := f.ReadAt(b, off); got < n {
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
