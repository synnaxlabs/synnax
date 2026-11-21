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
	"cmp"
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// segmentLimit is the stream length at which a commit seals the segment of a key.
const segmentLimit = 1 << 30

// logWriter writes one clock and the channels it times to the segments of the pointer
// log layout. Write appends with no sync, Commit makes the written samples visible, and
// Sync makes the commits durable with one log entry.
type logWriter struct {
	e  *engine
	st *state.Writer
	// clock is the key of the clock, and keys holds the clock first.
	clock ChannelKey
	keys  []*logKey
	byKey map[ChannelKey]*logKey
	// unstored is true for a writer that stamps and commits no bytes.
	unstored bool
	// number is the alignment number of the stretch, or zero before its first write,
	// and next the index of the next sample in it.
	number, next uint32
	// last is the last stamp written.
	last telem.TimeStamp
	// The uncommitted run: the index of its first sample, its first and last stamps,
	// and its sample count.
	from        uint32
	first, end  telem.TimeStamp
	samples     uint32
	lastCommit  telem.TimeStamp
	frames      int
	entry       log.Entry
	excluded    []ChannelKey
	sealed      []*segment.Handle
	keySegments []log.Slot
}

// logKey is the open segment of one key of a logWriter.
type logKey struct {
	ch   Channel
	code telem.DataTypeCode
	h    *segment.Handle
	w    *segment.Writer
	id   uint64
	// committed is the committed stream length, and dirty is true when bytes since the
	// last sync are unsynced.
	committed int64
	dirty     bool
	// frame is the last frame that held a series of the key, and skipped is true when a
	// write of the uncommitted run skipped the key.
	frame   int
	skipped bool
}

// openLogWriter opens a writer of cfg on the pointer log layout. The writer writes one
// clock and the channels it times, the clock included. The caller holds db.mu.
func (db *DB) openLogWriter(cfg WriterConfig) (*logWriter, error) {
	w := &logWriter{
		e:        db.engine,
		byKey:    make(map[ChannelKey]*logKey, len(cfg.Channels)),
		unstored: cfg.Mode == WriterModeStreamOnly,
	}
	keys := make([]state.WriterKey, 0, len(cfg.Channels))
	for i, key := range cfg.Channels {
		ch, ok := db.channelOf(key)
		if !ok {
			return nil, channel.NewNotFoundError(key)
		}
		clock := ch.Index
		if ch.IsIndex {
			clock = ch.Key
		}
		if i == 0 {
			w.clock = clock
		} else if clock != w.clock {
			return nil, errors.Wrap(
				errors.ErrUnsupported,
				"a writer of the pointer log writes one clock",
			)
		}
		lk := &logKey{ch: ch, code: telem.CodeOf(ch.DataType)}
		w.byKey[key] = lk
		w.keys = append(w.keys, lk)
		keys = append(keys, state.WriterKey{Key: key, DataType: uint8(lk.code)})
	}
	if _, ok := w.byKey[w.clock]; !ok {
		return nil, errors.Wrap(
			errors.ErrUnsupported,
			"a writer of the pointer log writes its clock",
		)
	}
	slices.SortStableFunc(w.keys, func(a, b *logKey) int {
		return boolOrder(b.ch.Key == w.clock) - boolOrder(a.ch.Key == w.clock)
	})
	if err := w.openState(db, cfg, keys); err != nil {
		return nil, err
	}
	return w, nil
}

// openState opens the state writer of w, and seals each open segment of its keys that
// the state hands over, since a writer starts a new segment for each key.
func (w *logWriter) openState(db *DB, cfg WriterConfig, keys []state.WriterKey) error {
	start := cfg.Start
	if start.IsZero() {
		start = telem.NewTimeStamp(db.clock.Now())
	}
	st, err := db.engine.state.OpenWriter(state.WriterConfig{
		Keys:    keys,
		Clock:   w.clock,
		Start:   start,
		Bounded: cfg.Mode == WriterModePersistOnly,
	})
	if err != nil {
		return err
	}
	w.st = st
	for _, lk := range w.keys {
		_, held, err := st.Held(lk.ch.Key)
		if err != nil {
			return errors.Join(err, st.Close())
		}
		if !held {
			continue
		}
		seal, err := st.Seal(lk.ch.Key)
		if err != nil {
			return errors.Join(err, st.Close())
		}
		w.entry.Add(&seal)
	}
	return nil
}

func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

// write writes the series of fr for the keys of w, except the keys for which skip
// returns true. A skipped clock skips every key, and a key skipped in a write stays
// skipped until the next commit.
func (w *logWriter) write(fr Frame, skip func(ChannelKey) bool) (Frame, bool, error) {
	w.excluded = w.excluded[:0]
	w.frames++
	clock, n, err := w.validate(fr)
	if err != nil || n == 0 {
		return fr, false, err
	}
	if skip != nil && skip(w.clock) {
		for _, k := range fr.RawKeys() {
			if _, ok := w.byKey[k]; ok {
				w.excluded = append(w.excluded, k)
			}
		}
		return fr.ExcludeKeys(w.excluded), true, nil
	}
	first, last, err := w.checkOrder(clock)
	if err != nil {
		return fr, false, err
	}
	if w.number == 0 {
		if w.number, err = w.st.NewAlignment(); err != nil {
			return fr, false, err
		}
		w.next = 0
	}
	alignment := telem.NewAlignment(w.number, w.next)
	tr := telem.TimeRange{Start: first, End: last + 1}
	for i, k := range fr.RawKeys() {
		lk, ok := w.byKey[k]
		if !ok || fr.ShouldExcludeRaw(i) {
			continue
		}
		if lk.skipped || (k != w.clock && skip != nil && skip(k)) {
			lk.skipped = true
			w.excluded = append(w.excluded, k)
			continue
		}
		s := fr.RawSeriesAt(i)
		if !w.unstored {
			if err = w.append(lk, s.Data); err != nil {
				w.putBack()
				return fr, false, err
			}
		}
		s.Alignment, s.TimeRange = alignment, tr
		fr.SetRawSeriesAt(i, s)
	}
	if w.samples == 0 {
		w.from, w.first = w.next, first
	}
	w.end, w.last = last, last
	w.samples += uint32(n)
	w.next += uint32(n)
	return fr.ExcludeKeys(w.excluded), len(w.excluded) > 0, nil
}

// validate checks the series of fr for the keys of w: one per key, of the key's type,
// all of one length, and the clock's among them. It returns the clock series and the
// length, or zero when fr holds no series of w.
func (w *logWriter) validate(fr Frame) (telem.Series, int64, error) {
	var (
		clock    telem.Series
		hasClock bool
		n        int64 = -1
	)
	for i, k := range fr.RawKeys() {
		lk, ok := w.byKey[k]
		if !ok || fr.ShouldExcludeRaw(i) {
			continue
		}
		s := fr.RawSeriesAt(i)
		if s.DataType != lk.ch.DataType {
			return clock, 0, invalidDataTypeError(lk.ch, s.DataType)
		}
		if lk.frame == w.frames {
			return clock, 0, oneSeriesPerChannelError(lk.ch)
		}
		lk.frame = w.frames
		if n == -1 {
			n = s.Len()
		} else if s.Len() != n {
			return clock, 0, sameLengthForAllSeriesError(lk.ch, n, s)
		}
		if k == w.clock {
			clock, hasClock = s, true
		}
	}
	if n <= 0 {
		return clock, 0, nil
	}
	if !hasClock {
		return clock, 0, channel.Wrap(w.clock, errors.Wrap(
			validate.ErrValidation,
			"frame holds samples that the clock does not stamp",
		))
	}
	return clock, n, nil
}

// checkOrder returns the first and last stamps of the clock series s, which must rise
// strictly, from after the last stamp of the stretch.
func (w *logWriter) checkOrder(
	s telem.Series,
) (first, last telem.TimeStamp, err error) {
	prev := w.last
	for i := range int(s.Len()) {
		stamp := s.ValueAt[telem.TimeStamp](i)
		if (i > 0 || w.number != 0) && stamp <= prev {
			return 0, 0, channel.Wrap(w.clock, errors.Wrapf(
				validate.ErrValidation,
				"stamp %v must be after %v",
				stamp,
				prev,
			))
		}
		if i == 0 {
			first = stamp
		}
		prev = stamp
	}
	return first, prev, nil
}

// append appends p to the open segment of lk, which it creates first when lk has none.
func (w *logWriter) append(lk *logKey, p []byte) error {
	if lk.w == nil {
		<-w.e.ready
		id, err := w.st.NewSegment(lk.ch.Key)
		if err != nil {
			return err
		}
		w.e.named(id, lk.ch.Key)
		w.entry.Add(&log.Create{
			Segment:  id,
			Key:      lk.ch.Key,
			Clock:    w.clock,
			Version:  segmentVersion,
			DataType: uint8(lk.code),
		})
		cfg := segment.Config{
			DB:       w.e.db,
			ID:       id,
			DataType: uint8(lk.code),
			Clock:    w.clock,
			Variable: lk.ch.DataType.IsVariable(),
		}
		h, err := w.e.budget.Create(w.e.fs, segmentName(lk.ch.Key, id))
		if err != nil {
			return err
		}
		lk.h, lk.w, lk.id, lk.committed = h, segment.NewWriter(h, cfg), id, 0
	}
	if err := lk.w.Write(p); err != nil {
		return err
	}
	lk.dirty = true
	return nil
}

// putBack puts every segment back to its last commit, and ends the stretch.
func (w *logWriter) putBack() {
	for _, lk := range w.keys {
		if lk.w != nil {
			lk.w.PutBack()
		}
		lk.skipped = false
	}
	w.samples, w.number = 0, 0
}

// commit makes the uncommitted run visible and returns its end. It commits nothing and
// returns true when skip holds a key with samples to commit.
func (w *logWriter) commit(skip func(ChannelKey) bool) (telem.TimeStamp, bool, error) {
	if w.samples == 0 {
		return w.lastCommit, false, nil
	}
	if skip != nil {
		for _, lk := range w.keys {
			if !lk.skipped && skip(lk.ch.Key) {
				return 0, true, nil
			}
		}
	}
	end := w.end + 1
	if !w.unstored {
		c := &log.Commit{
			Samples:   w.samples,
			Alignment: uint64(telem.NewAlignment(w.number, w.from)),
			First:     int64(w.first),
			Last:      int64(w.end),
		}
		w.keySegments = w.keySegments[:0]
		for _, lk := range w.keys {
			if lk.w == nil || lk.w.Len() == lk.committed {
				continue
			}
			length, crc := lk.w.Commit()
			lk.committed = int64(length)
			w.keySegments = append(w.keySegments, log.Slot{
				Segment: lk.id,
				Length:  length,
				TailCRC: crc,
			})
		}
		slices.SortFunc(w.keySegments, func(a, b log.Slot) int {
			return cmp.Compare(a.Segment, b.Segment)
		})
		c.Slots = slices.Clone(w.keySegments)
		if err := w.st.Commit(c); err != nil {
			w.putBack()
			return 0, false, err
		}
		w.entry.Add(c)
	}
	w.samples, w.lastCommit = 0, end
	for _, lk := range w.keys {
		lk.skipped = false
	}
	return end, false, w.sealFull()
}

// sealFull seals each segment that reached segmentLimit, so the next write creates
// another.
func (w *logWriter) sealFull() error {
	for _, lk := range w.keys {
		if lk.w == nil || lk.w.Len() < segmentLimit {
			continue
		}
		if err := w.seal(lk); err != nil {
			return err
		}
	}
	return nil
}

func (w *logWriter) seal(lk *logKey) error {
	seal, err := w.st.Seal(lk.ch.Key)
	if err != nil {
		return err
	}
	w.entry.Add(&seal)
	w.sealed = append(w.sealed, lk.h)
	lk.h, lk.w = nil, nil
	return nil
}

// endStretch commits, then makes the next write start a new stretch.
func (w *logWriter) endStretch() error {
	if _, _, err := w.commit(nil); err != nil {
		return err
	}
	w.number = 0
	return nil
}

// sync syncs every segment with unsynced bytes, then logs the records since the last
// sync and waits until they are durable. A failed sync takes the samples that no
// durable record covers out of view and seals each segment.
func (w *logWriter) sync() error {
	handles := slices.Clone(w.sealed)
	for _, lk := range w.keys {
		if lk.dirty && lk.h != nil {
			handles = append(handles, lk.h)
		}
	}
	if err := w.e.budget.Sync(handles); err != nil {
		seals, takeErr := w.st.TakeBack()
		for i := range seals {
			w.entry.Add(&seals[i])
		}
		return errors.Join(err, takeErr, w.flush())
	}
	for _, lk := range w.keys {
		lk.dirty = false
	}
	if err := w.flush(); err != nil {
		return err
	}
	var err error
	for _, h := range w.sealed {
		err = errors.Join(err, h.Close())
	}
	w.sealed = w.sealed[:0]
	return err
}

// flush logs the records since the last sync and waits until they are durable.
func (w *logWriter) flush() error {
	if w.entry.Len() == 0 {
		return nil
	}
	ack := w.e.log.Append(&w.entry)
	w.entry.Reset()
	_, err := ack.Wait()
	return err
}

// close syncs, releases the claims of w, and closes its segment files.
func (w *logWriter) close() error {
	err := w.sync()
	err = errors.Join(err, w.st.Close())
	for _, lk := range w.keys {
		if lk.h != nil {
			err = errors.Join(err, lk.h.Close())
		}
	}
	return errors.Join(err, w.e.reap())
}
