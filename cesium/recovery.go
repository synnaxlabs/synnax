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
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// ErrOldLayout is returned by an open on the pointer log of a root that holds the old
// layout, which only a conversion opens.
var ErrOldLayout = errors.New("the root holds the old storage layout")

// recovery reads a root on the pointer log: the newest head that passes its checks,
// then every log entry after it, in log order.
type recovery struct {
	fs xfs.FS
	// heads and logs are the generations of the head and log files in the listing
	// taken at open.
	heads, logs []uint64
	// db is the identity of the database, and head the header of the head in use.
	db   uint64
	head checkpoint.Header
	// gen is the generation of the newest log, size the size of its file, and fresh is
	// true when it holds no header yet. logHeader is its header.
	gen       uint64
	size      int64
	fresh     bool
	logHeader log.Header
	// first is the sequence of the first entry of the newest log, and pos is where it
	// goes on.
	first uint64
	pos   log.Position
	// clean is true when the last valid frame is a clean mark.
	clean bool
	// last is the last valid frame and its offset, which recovery writes again.
	last    []byte
	lastOff int64
	buf     []byte
	dec     log.Decoder
}

// list sorts the head and log files of the root, and reports whether a channel
// directory of the old layout exists.
func (r *recovery) list() (old bool, err error) {
	infos, err := r.fs.List("")
	if err != nil {
		return false, err
	}
	for _, info := range infos {
		name := info.Name()
		if info.IsDir() {
			if _, err := strconv.ParseUint(name, 10, 32); err != nil {
				continue
			}
			if ok, err := r.fs.Exists(name + "/" + meta.File); err != nil {
				return false, err
			} else if ok {
				old = true
			}
			continue
		}
		if g, ok := generationOf(name, ".head"); ok {
			r.heads = append(r.heads, g)
		} else if g, ok := generationOf(name, ".log"); ok {
			r.logs = append(r.logs, g)
		}
	}
	slices.Sort(r.heads)
	slices.Sort(r.logs)
	return old, nil
}

// generationOf parses the generation of a file named by 16 hex digits and suffix.
func generationOf(name, suffix string) (uint64, bool) {
	hex, ok := strings.CutSuffix(name, suffix)
	if !ok || len(hex) != 16 {
		return 0, false
	}
	g, err := strconv.ParseUint(hex, 16, 64)
	return g, err == nil
}

// pickHead takes the newest head that passes, and removes each newer head that is not
// complete. It returns nil when the root holds no head that passes and may start as a
// new database: no head at all, or only an incomplete head 1 whose log holds no frame.
func (r *recovery) pickHead() (*state.Snapshot, error) {
	var incomplete []uint64
	for i := len(r.heads) - 1; i >= 0; i-- {
		g := r.heads[i]
		h, s, err := checkpoint.Read(r.fs, g)
		if err == nil {
			for _, gen := range incomplete {
				if err := r.fs.Remove(checkpoint.Name(gen)); err != nil {
					return nil, err
				}
			}
			if len(incomplete) > 0 {
				if err := r.fs.SyncDir(""); err != nil {
					return nil, err
				}
			}
			r.db, r.head = h.DB, h
			return s, nil
		}
		switch {
		case errors.Is(err, checkpoint.ErrIncomplete):
			incomplete = append(incomplete, g)
		case errors.Is(err, place.ErrCorrupt):
		default:
			return nil, err
		}
	}
	if len(r.heads) == 0 {
		return nil, nil
	}
	if len(r.heads) == 1 && r.heads[0] == 1 && len(incomplete) == 1 {
		empty, err := r.logEmpty(1)
		if err != nil || empty {
			return nil, err
		}
	}
	return nil, errors.Wrapf(
		place.ErrCorrupt,
		"no head of the database passes its checks: generations %v",
		r.heads,
	)
}

// logEmpty reports whether the log of generation holds no frame, from the identity in
// its own header, since no head names one.
func (r *recovery) logEmpty(generation uint64) (bool, error) {
	b, ok, err := r.read(generation)
	if err != nil || !ok {
		return true, err
	}
	db := binary.LittleEndian.Uint64(b[16:])
	if _, _, ok = r.header(b, db, generation); !ok {
		return true, nil
	}
	_, err = log.DecodeFrame(b[log.FramesStart:], db, generation, log.FramesStart)
	return err != nil, nil
}

// read reads the whole log file of generation into the buffer of r. It returns false
// when no such file exists.
func (r *recovery) read(generation uint64) (b []byte, ok bool, err error) {
	name := log.Name(generation)
	if ok, err = r.fs.Exists(name); err != nil || !ok {
		return nil, false, err
	}
	f, err := r.fs.Open(name, os.O_RDONLY)
	if err != nil {
		return nil, false, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	r.buf = slices.Grow(r.buf[:0], int(info.Size()))[:info.Size()]
	if _, err = f.ReadAt(r.buf, 0); err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	return r.buf, true, nil
}

// reread reads n bytes at off of the log of generation again, into b.
func (r *recovery) reread(generation uint64, b []byte, off int64) error {
	f, err := r.fs.Open(log.Name(generation), os.O_RDONLY)
	if err != nil {
		return err
	}
	_, err = f.ReadAt(b, off)
	if errors.Is(err, io.EOF) {
		err = nil
	}
	return errors.Join(err, f.Close())
}

// header returns the header of the log of generation in b, from the first copy that
// passes, and the crc field of the copy at 0 that the first frame links to.
func (r *recovery) header(b []byte, db, generation uint64) (log.Header, uint32, bool) {
	if len(b) < log.FramesStart {
		return log.Header{}, 0, false
	}
	for _, off := range []uint64{0, log.HeaderCopy} {
		h, err := log.DecodeHeader(b[off:], db, generation, off)
		if err != nil {
			continue
		}
		return h, binary.LittleEndian.Uint32(h.Encode(0)[log.HeaderSize-4:]), true
	}
	return log.Header{}, 0, false
}

// verifiedHeader is header with the second read of the header copies of the rule that
// a failed check is read again before recovery decides.
func (r *recovery) verifiedHeader(
	b []byte,
	db, generation uint64,
) (log.Header, uint32, bool, error) {
	if h, crc, ok := r.header(b, db, generation); ok {
		return h, crc, true, nil
	}
	if len(b) < log.FramesStart {
		return log.Header{}, 0, false, nil
	}
	if err := r.reread(generation, b[:log.FramesStart], 0); err != nil {
		return log.Header{}, 0, false, err
	}
	h, crc, ok := r.header(b, db, generation)
	return h, crc, ok, nil
}

// frameAt returns the frame at off of the log of generation in b when it is valid: its
// magic, bounds, and check pass, and it continues sequence with a link to prev. A frame
// that fails is read a second time before frameAt decides.
func (r *recovery) frameAt(
	b []byte,
	generation uint64,
	off int64,
	sequence uint64,
	prev uint32,
) (log.Frame, bool, error) {
	if off+log.FrameHeaderSize > int64(len(b)) {
		return log.Frame{}, false, nil
	}
	valid := func() (log.Frame, bool) {
		f, err := log.DecodeFrame(b[off:], r.db, generation, uint64(off))
		return f, err == nil && f.Sequence == sequence && f.PrevCRC == prev
	}
	if f, ok := valid(); ok {
		return f, true, nil
	}
	end := min(off+log.MaxFrame, int64(len(b)))
	if err := r.reread(generation, b[off:end], off); err != nil {
		return log.Frame{}, false, err
	}
	f, ok := valid()
	return f, ok, nil
}

// replay applies every entry of the logs after the head to replay, in log order, and
// leaves r at the end of the last valid frame. It refuses with an error that wraps
// place.ErrCorrupt when synced bytes are bad.
func (r *recovery) replay(
	apply func(sequence uint64, records []log.Record) error,
) error {
	r.gen = r.head.Generation
	b, ok, err := r.read(r.gen)
	if err != nil {
		return err
	}
	if !ok {
		return errors.Wrapf(place.ErrCorrupt, "log %d of the head is missing", r.gen)
	}
	r.size = int64(len(b))
	h, prev, ok, err := r.verifiedHeader(b, r.db, r.gen)
	if err != nil {
		return err
	}
	if !ok {
		if r.gen == 1 && r.head.Sequence == 0 && allZero(b[:log.FramesStart]) {
			r.fresh, r.first = true, 1
			r.pos = log.Position{End: log.FramesStart, Sequence: 1}
			return nil
		}
		return errors.Wrapf(place.ErrCorrupt, "no header of log %d passes", r.gen)
	}
	if h.Sequence > r.head.Sequence+1 {
		return errors.Wrapf(
			place.ErrCorrupt,
			"log %d starts at entry %d, after entry %d of its head",
			r.gen,
			h.Sequence,
			r.head.Sequence,
		)
	}
	r.first, r.logHeader = h.Sequence, h
	sequence, end := h.Sequence, int64(log.FramesStart)
	for {
		if end, sequence, prev, err = r.frames(
			b,
			end,
			sequence,
			prev,
			apply,
		); err != nil {
			return err
		}
		next, ok, err := r.read(r.gen + 1)
		if err != nil {
			return err
		}
		if ok {
			nh, crc, passes, err := r.verifiedHeader(next, r.db, r.gen+1)
			if err != nil {
				return err
			}
			if passes && nh.PrevCRC == prev && nh.Sequence == sequence {
				r.gen++
				r.first, r.logHeader, r.size = nh.Sequence, nh, int64(len(next))
				b, prev, end = next, crc, log.FramesStart
				continue
			}
			if passes {
				return errors.Wrapf(
					place.ErrCorrupt,
					"log %d links to another frame than the end of log %d at %d",
					r.gen+1,
					r.gen,
					end,
				)
			}
			if err = r.scan(next, r.gen+1, log.FramesStart, sequence); err != nil {
				return err
			}
			if b, _, err = r.read(r.gen); err != nil {
				return err
			}
		}
		if err = r.scan(b, r.gen, end, sequence); err != nil {
			return err
		}
		r.pos = log.Position{End: end, Sequence: sequence, PrevCRC: prev}
		return nil
	}
}

// frames applies the valid frames of the log of generation r.gen in b from end, and
// returns the end of the last one with the sequence and link that follow it.
func (r *recovery) frames(
	b []byte,
	end int64,
	sequence uint64,
	prev uint32,
	apply func(sequence uint64, records []log.Record) error,
) (int64, uint64, uint32, error) {
	for {
		off := end
		f, ok, err := r.frameAt(b, r.gen, off, sequence, prev)
		if err != nil {
			return 0, 0, 0, err
		}
		if !ok {
			if next := (end + log.BlockSize - 1) &^ (log.BlockSize - 1); next != end {
				off = next
				if f, ok, err = r.frameAt(b, r.gen, off, sequence, prev); err != nil {
					return 0, 0, 0, err
				}
			}
		}
		if !ok {
			return end, sequence, prev, nil
		}
		if err = r.entries(f, apply); err != nil {
			return 0, 0, 0, err
		}
		end = off + int64(f.Size())
		r.last, r.lastOff = append(r.last[:0], b[off:end]...), off
		r.clean = f.Count == 0 && f.Clean
		sequence, prev = f.Sequence+uint64(f.Count), f.CRC
	}
}

// entries applies the entries of f that the head does not cover.
func (r *recovery) entries(
	f log.Frame,
	apply func(sequence uint64, records []log.Record) error,
) error {
	b := f.Entries
	for i := range uint64(f.Count) {
		n, err := r.dec.Reset(b)
		if err != nil {
			return err
		}
		var records []log.Record
		for {
			rec, ok, err := r.dec.Next()
			if err != nil {
				return err
			}
			if !ok {
				break
			}
			records = append(records, log.Clone(rec))
		}
		b = b[n:]
		if sequence := f.Sequence + i; sequence > r.head.Sequence {
			if err = apply(sequence, records); err != nil {
				return err
			}
		}
	}
	return nil
}

// scan applies the torn tail rule to the log of generation in b from off: a frame that
// passes at its own offset there proves that synced bytes are bad. It reads the bytes
// twice.
func (r *recovery) scan(b []byte, generation uint64, off int64, sequence uint64) error {
	if off >= int64(len(b)) {
		return nil
	}
	// found scans tail, the bytes of the log from base on.
	found := func(tail []byte, base int64) (int64, bool) {
		for at := (off + 7) &^ 7; at+log.FrameHeaderSize <= base+int64(len(tail)); at += 8 {
			b := tail[at-base:]
			if !bytes.Equal(b[:4], frameMagic) {
				continue
			}
			if _, err := log.DecodeFrame(b, r.db, generation, uint64(at)); err == nil {
				return at, true
			}
		}
		return 0, false
	}
	at, ok := found(b[off:], off)
	if !ok {
		second := make([]byte, len(b)-int(off))
		if err := r.reread(generation, second, off); err != nil {
			return err
		}
		at, ok = found(second, off)
	}
	if !ok {
		return nil
	}
	return errors.Wrapf(
		place.ErrCorrupt,
		"log %d holds a valid frame at %d after a bad frame before entry %d",
		generation,
		at,
		sequence,
	)
}

// rewrite writes again, with the same bytes, what recovery read and builds on: the last
// valid frame, both header copies of the newest log, and the header sector of the head
// in use. A failed sync before the stop may have left them only in the page cache.
func (r *recovery) rewrite() error {
	if r.fresh {
		return nil
	}
	f, err := r.fs.Open(log.Name(r.gen), os.O_RDWR)
	if err != nil {
		return err
	}
	_, err = f.WriteAt(r.logHeader.Encode(0), 0)
	if err == nil {
		_, err = f.WriteAt(r.logHeader.Encode(log.HeaderCopy), log.HeaderCopy)
	}
	if err == nil && len(r.last) > 0 && r.lastOff >= log.FramesStart {
		_, err = f.WriteAt(r.last, r.lastOff)
	}
	if err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return err
	}
	return r.rewriteHeadSector()
}

func (r *recovery) rewriteHeadSector() error {
	f, err := r.fs.Open(checkpoint.Name(r.head.Generation), os.O_RDWR)
	if err != nil {
		return err
	}
	sector := make([]byte, checkpoint.SectorSize)
	if _, err = f.ReadAt(sector, 0); err == nil {
		if _, err = f.WriteAt(sector, 0); err == nil {
			err = f.Sync()
		}
	}
	return errors.Join(err, f.Close())
}

// restart writes and syncs a frame that holds one Restart at the block boundary after
// the last valid frame, unless that frame is a clean mark, and returns its sequence and
// records. A Restart that would pass three quarters of the log goes in the next log,
// and switched is then true.
func (r *recovery) restart() (
	sequence uint64,
	records []log.Record,
	switched bool,
	err error,
) {
	if r.fresh || r.clean {
		return 0, nil, false, nil
	}
	var e log.Entry
	e.Add(&log.Restart{})
	frame := log.Frame{
		Count:    1,
		Sequence: r.pos.Sequence,
		PrevCRC:  r.pos.PrevCRC,
		Entries:  e.Bytes(),
	}
	start := (r.pos.End + log.BlockSize - 1) &^ (log.BlockSize - 1)
	if start+int64(frame.Size()) > r.size/4*3 {
		if frame.PrevCRC, err = r.next(); err != nil {
			return 0, nil, false, err
		}
		start, switched = log.FramesStart, true
	}
	b := frame.Append(nil, r.db, r.gen, uint64(start))
	f, err := r.fs.Open(log.Name(r.gen), os.O_RDWR)
	if err != nil {
		return 0, nil, false, err
	}
	if _, err = f.WriteAt(b, start); err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return 0, nil, false, err
	}
	sequence = r.pos.Sequence
	r.pos = log.Position{
		End:      start + int64(len(b)),
		Sequence: sequence + 1,
		PrevCRC:  frame.CRC,
	}
	return sequence, []log.Record{&log.Restart{}}, switched, nil
}

// next makes the next log live: its file, prepared when it is missing or not whole,
// then both header copies, synced. It returns the crc field of the copy at 0.
func (r *recovery) next() (uint32, error) {
	gen := r.gen + 1
	size, err := r.fs.Exists(log.Name(gen))
	if err != nil {
		return 0, err
	}
	whole := false
	if size {
		f, err := r.fs.Open(log.Name(gen), os.O_RDONLY)
		if err != nil {
			return 0, err
		}
		info, err := f.Stat()
		if err = errors.Join(err, f.Close()); err != nil {
			return 0, err
		}
		whole = info.Size() == log.FileSize
	}
	if !whole {
		if err = log.Prepare(r.fs, gen, log.FileSize); err != nil {
			return 0, err
		}
	}
	h := log.Header{
		DB:         r.db,
		Generation: gen,
		Sequence:   r.pos.Sequence,
		Size:       log.FileSize,
		PrevCRC:    r.pos.PrevCRC,
	}
	f, err := r.fs.Open(log.Name(gen), os.O_RDWR)
	if err != nil {
		return 0, err
	}
	first := h.Encode(0)
	_, err = f.WriteAt(first, 0)
	if err == nil {
		_, err = f.WriteAt(h.Encode(log.HeaderCopy), log.HeaderCopy)
	}
	if err == nil {
		err = f.Sync()
	}
	if err = errors.Join(err, f.Close()); err != nil {
		return 0, err
	}
	r.gen, r.first, r.logHeader, r.size = gen, h.Sequence, h, log.FileSize
	return binary.LittleEndian.Uint32(first[log.HeaderSize-4:]), nil
}

// frameMagic is the first four bytes of every log frame.
var frameMagic = binary.LittleEndian.AppendUint32(nil, 0x464C5043)

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
