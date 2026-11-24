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
	"cmp"
	"context"
	"slices"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// Report is what a repair of the pointer log layout found in a root, and what it did.
type Report struct {
	// Damage holds each bad frame, in log order. A root with none opens.
	Damage []Damage
	// Rebuilt holds each range of the bad frames that the repair places, by key and
	// segment. The ranges of one clock write share a fresh alignment.
	Rebuilt []Range
	// Unplaced holds each range in a segment file that the repair does not place, with
	// why, by key and segment.
	Unplaced []Range
	// Tails holds the last block of each segment whose last commit was in a bad frame,
	// which nothing checks, by key and segment.
	Tails []Range
	// Unrecovered holds each segment file that the repair cannot place, which it
	// renames <S>.seg.unrecovered, by key and segment.
	Unrecovered []File
	// Buried holds each sealed segment of the head with no pointer, which leaves the
	// state, by key and segment.
	Buried []Segment
	// Lost holds the records of the bad frames, as far as they decode, in rising order
	// of sequence. A repair never applies them.
	Lost []Record
	// Repaired is true when the repair committed.
	Repaired bool
}

// Damage is a bad frame of a log, which valid frames follow.
type Damage struct {
	// Generation is the generation of the log, and Offset the end of the valid frame
	// before the bad frame, where the bad bytes can start.
	Generation, Offset uint64
	// FirstSequence and LastSequence are the sequences of the entries that the frame
	// held: those between its neighbors.
	FirstSequence, LastSequence uint64
}

// Range is a range of samples of a segment that a repair lists.
type Range struct {
	Key     ChannelKey
	Segment uint64
	// Bytes and Samples are the size of the range, in stream bytes and in samples.
	Bytes   telem.Size
	Samples int64
	// TimeRange is the time range of the samples, or the zero range when the repair
	// places none.
	TimeRange telem.TimeRange
	// Open is true for a range that no later record ends, which can hold samples that
	// a writer wrote and never committed.
	Open bool
	// Reason says why the repair does not place the range, or is empty.
	Reason string
}

// File is a segment file that a repair lists.
type File struct {
	// Key is the key of the directory of the file, and Segment the identity in its
	// name.
	Key     ChannelKey
	Segment uint64
	// Bytes is the size of the file.
	Bytes telem.Size
	// Reason says why the repair lists the file.
	Reason string
}

// Segment is a segment that a repair lists.
type Segment struct {
	Key     ChannelKey
	Segment uint64
}

// Record is a record of a log that a repair lists.
type Record struct {
	// Sequence is the sequence of the entry of the record.
	Sequence uint64
	// Kind names the record, such as Tombstone or Commit, or is empty when none
	// decodes.
	Kind string
	// Key is the key that the record names, or 0.
	Key ChannelKey
	// TimeRange is the time range that the record names, or the zero range.
	TimeRange telem.TimeRange
	// Reason says why the repair lists the record.
	Reason string
}

// damagedReason is the reason of each record of a bad frame.
const damagedReason = "from a damaged frame, so it may be wrong"

// Repair finds the damage of the log of the root dirname on the pointer log layout,
// the ranges of the bad frames that it rebuilds, and the bytes that it cannot place,
// and reports them. opts are those of
// Open: a root with a database needs WithIdentity. Repair refuses a root of the old
// layout, which Open converts, with an error that wraps errors.ErrUnsupported, and
// another identity than the root's with no undo. It writes nothing.
func Repair(
	ctx context.Context,
	dirname string,
	confirm func(Report) bool,
	opts ...Option,
) (Report, error) {
	o, err := newOptions(dirname, opts...)
	if err != nil {
		return Report{}, err
	}
	if err = openFS(o); err != nil {
		return Report{}, err
	}
	r := &recovery{fs: o.fs, readOnly: true}
	old, err := r.list()
	if err != nil {
		return Report{}, err
	}
	if old && len(r.heads) == 0 {
		return Report{}, errors.Wrap(
			errors.ErrUnsupported,
			"the root holds the old layout, which an open converts",
		)
	}
	snapshot, err := r.pickHead()
	if err != nil || snapshot == nil {
		return Report{}, err
	}
	if err = refuseIdentity(o, r); err != nil {
		return Report{}, err
	}
	if r.parts, err = checkpoint.LoadParts(o.fs, r.db, r.head); err != nil {
		return Report{}, err
	}
	e, err := replayEngine(o, r, snapshot)
	if err != nil {
		return Report{}, err
	}
	var (
		rep Report
		l   = newLater()
	)
	if rep.Damage, rep.Lost, err = r.damage(ctx, e.replay, l.add); err != nil {
		return Report{}, err
	}
	rep.Buried = buried(e)
	if len(rep.Damage) == 0 && len(rep.Buried) == 0 {
		return rep, nil
	}
	placed, unrecovered, err := e.place(r, l)
	if err != nil {
		return Report{}, err
	}
	rep.Unrecovered = unrecovered
	if _, rep.Rebuilt, rep.Unplaced, rep.Tails, err = e.rebuild(l, placed); err != nil {
		return Report{}, err
	}
	return rep, nil
}

// replayEngine returns an engine in replay mode on snapshot, the head of r, whose state
// buries each sealed segment with no pointer.
func replayEngine(o *options, r *recovery, snapshot *state.Snapshot) (*engine, error) {
	o.repairing = true
	e, err := newEngine(o, true)
	if err != nil {
		return nil, err
	}
	if e.state, err = openState(o, e, r.head.Generation, snapshot); err != nil {
		return nil, err
	}
	e.attach(r, snapshot)
	e.replaying = true
	return e, nil
}

// buried returns the segments that the state of e buried at its load, by key and
// segment.
func buried(e *engine) []Segment {
	out := make([]Segment, 0, len(e.state.Buried()))
	for _, d := range e.state.Buried() {
		out = append(out, Segment{Key: d.Key, Segment: d.Segment})
	}
	slices.SortFunc(out, func(a, b Segment) int {
		return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Segment, b.Segment))
	})
	return out
}

// refuseIdentity refuses the database of r when the caller of o holds another identity
// or none. Unlike Open, it undoes nothing.
func refuseIdentity(o *options, r *recovery) error {
	switch {
	case !o.identified:
		return errors.Newf(
			"the root holds database %016x, and the caller gave no identity: pass the "+
				"identity from DB.Identity, or 0 when none is held",
			r.db,
		)
	case r.db != o.identity:
		return errors.Newf("the root holds database %016x, not %016x, which the "+
			"caller holds", r.db, o.identity)
	}
	return nil
}

// damage reads the logs after the head as replay does, and returns each bad frame that
// a valid frame follows in its log, and the records of the bad frames. At a bad frame,
// it goes on from the valid frame after it. It gives each entry before the first bad
// frame to before, and each entry of a valid frame after it to after.
func (r *recovery) damage(
	ctx context.Context,
	before, after func(sequence uint64, records []log.Record) error,
) ([]Damage, []Record, error) {
	b, h, prev, err := r.headLog()
	if err != nil {
		return nil, nil, err
	}
	var (
		dmg      []Damage
		lost     []Record
		sequence = h.Sequence
		end      = int64(log.FramesStart)
	)
	apply := func(sequence uint64, records []log.Record) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if dmg == nil {
			return before(sequence, records)
		}
		return after(sequence, records)
	}
	for {
		if end, sequence, prev, err = r.frames(
			b,
			end,
			sequence,
			prev,
			apply,
		); err != nil {
			return nil, nil, err
		}
		at, f, found, err := r.validAfter(b, end)
		if err != nil {
			return nil, nil, err
		}
		if found {
			d, records, err := r.damageOf(end, sequence, at, f)
			if err != nil {
				return nil, nil, err
			}
			dmg, lost = append(dmg, d), append(lost, records...)
			sequence, prev, end = f.Sequence, f.PrevCRC, at
			continue
		}
		next, crc, ok, err := r.follow(end, sequence, prev)
		if err != nil || !ok {
			return dmg, lost, err
		}
		r.buf, r.following = r.following, r.buf
		b, prev, end = next, crc, log.FramesStart
	}
}

// headLog reads the log of the head, and returns its bytes, its header, and the crc
// that
// its first frame links to.
func (r *recovery) headLog() ([]byte, log.Header, uint32, error) {
	r.gen = r.head.Generation
	b, ok, err := r.read(r.gen)
	if err != nil {
		return nil, log.Header{}, 0, err
	}
	if !ok {
		return nil, log.Header{}, 0, errors.Wrapf(place.ErrCorrupt, "log %d of the "+
			"head is missing", r.gen)
	}
	h, prev, ok, err := r.verifiedHeader(b, r.db, r.gen)
	if err != nil {
		return nil, log.Header{}, 0, err
	}
	if !ok {
		return nil, log.Header{}, 0, errors.Wrapf(place.ErrCorrupt, "no header of log "+
			"%d passes", r.gen)
	}
	return b, h, prev, r.checkHeadLog(h, b)
}

// damageOf returns the damage of the bad frame at end of the newest log of r, which
// entry sequence starts and the valid frame f at at follows, and its records.
func (r *recovery) damageOf(
	end int64,
	sequence uint64,
	at int64,
	f log.Frame,
) (Damage, []Record, error) {
	if f.Sequence < sequence {
		return Damage{}, nil, errors.Wrapf(place.ErrCorrupt, "log %d holds a frame of "+
			"entry %d at %d after entry %d", r.gen, f.Sequence, at, sequence)
	}
	d := Damage{
		Generation:    r.gen,
		Offset:        uint64(end),
		FirstSequence: sequence,
		LastSequence:  f.Sequence - 1,
	}
	records, err := r.lostOf(d, end, at)
	return d, records, err
}

// validAfter returns the first frame after off of the log in b that passes at its own
// offset, and its offset. It reads the bytes twice.
func (r *recovery) validAfter(b []byte, off int64) (int64, log.Frame, bool, error) {
	if off >= int64(len(b)) {
		return 0, log.Frame{}, false, nil
	}
	found := func(tail []byte) (int64, log.Frame, bool) {
		for at := (off + 7) &^ 7; at+log.FrameHeaderSize <= off+int64(len(tail)); at += 8 {
			c := tail[at-off:]
			if !bytes.Equal(c[:4], frameMagic) {
				continue
			}
			if f, err := log.DecodeFrame(c, r.db, r.gen, uint64(at)); err == nil {
				return at, f, true
			}
		}
		return 0, log.Frame{}, false
	}
	if at, f, ok := found(b[off:]); ok {
		return at, f, true, nil
	}
	second := make([]byte, len(b)-int(off))
	if err := r.reread(r.gen, second, off); err != nil {
		return 0, log.Frame{}, false, err
	}
	at, f, ok := found(second)
	return at, f, ok, nil
}

// lostOf returns the records of d, a bad frame from off to at of the newest log of r,
// as far as a second read of its bytes decodes. It lists each entry that does not
// decode by its sequence alone.
func (r *recovery) lostOf(d Damage, off, at int64) ([]Record, error) {
	b := make([]byte, at-off)
	if err := r.reread(r.gen, b, off); err != nil {
		return nil, err
	}
	var out []Record
	sequence := d.FirstSequence
	if _, count, entries, ok := log.DecodeDamaged(b); ok {
		for range count {
			records, n, ok := decodeEntry(&r.dec, entries)
			if !ok || sequence > d.LastSequence {
				break
			}
			for _, rec := range records {
				out = append(out, recordOf(sequence, rec))
			}
			entries, sequence = entries[n:], sequence+1
		}
	}
	for ; sequence <= d.LastSequence; sequence++ {
		out = append(out, Record{Sequence: sequence, Reason: damagedReason})
	}
	return out, nil
}

// decodeEntry decodes the entry at the start of b with dec, and returns its records and
// its length, or false when it does not decode.
func decodeEntry(dec *log.Decoder, b []byte) ([]log.Record, int, bool) {
	n, err := dec.Reset(b)
	if err != nil {
		return nil, 0, false
	}
	var records []log.Record
	for {
		rec, ok, err := dec.Next()
		if err != nil {
			return nil, 0, false
		}
		if !ok {
			return records, n, true
		}
		records = append(records, log.Clone(rec))
	}
}

// recordOf returns the line of the report for rec, a record of the entry of sequence of
// a bad frame.
func recordOf(sequence uint64, rec log.Record) Record {
	out := Record{Sequence: sequence, Reason: damagedReason}
	switch rec := rec.(type) {
	case *log.Reserve:
		out.Kind = "Reserve"
	case *log.Create:
		out.Kind, out.Key = "Create", rec.Key
	case *log.Commit:
		out.Kind = "Commit"
		out.TimeRange = telem.TimeRange{
			Start: telem.TimeStamp(rec.First),
			End:   telem.TimeStamp(rec.Last) + 1,
		}
	case *log.Seal:
		out.Kind = "Seal"
	case *log.Tombstone:
		out.Kind, out.Key = "Tombstone", rec.Key
		out.TimeRange = telem.TimeRange{
			Start: telem.TimeStamp(rec.Start),
			End:   telem.TimeStamp(rec.End),
		}
	case *log.Replace:
		out.Kind = "Replace"
	case *log.Drop:
		out.Kind, out.Key = "Drop", rec.Key
	case *log.Restart:
		out.Kind = "Restart"
	case *log.Check:
		out.Kind, out.Key = "Check", rec.Key
	case *log.Intent:
		out.Kind, out.Key = "Intent", rec.Key
	}
	return out
}
