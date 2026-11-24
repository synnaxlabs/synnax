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
	"math"
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

// entryLimit is the size of the records since the last sync at which a commit syncs, so
// that the entry of a sync always fits in one log frame.
const entryLimit = log.MaxFrame / 4

// logWriter writes one clock and the channels it times to the segments of the pointer
// log layout, or only channels of a clock whose stamps another writer stored. Write
// appends with no sync, Commit makes the written samples visible, and Sync makes the
// commits durable with one log entry.
type logWriter struct {
	e  *engine
	st *state.Writer
	// clock is the key of the clock, and keys holds the clock first when the writer
	// writes it.
	clock   ChannelKey
	clockCh Channel
	keys    []*logKey
	byKey   map[ChannelKey]*logKey
	// unstored is true for a writer that stamps and commits no bytes.
	unstored bool
	// stamps places the samples of a writer that does not write its clock, and is nil
	// for a writer that does.
	stamps *placer
	// number is the alignment number of the stretch, or zero before its first write,
	// and next the index of the next sample in it.
	number, next uint32
	// last is the last stamp written.
	last telem.TimeStamp
	// runs are the uncommitted runs, in the order that Commit logs them.
	runs        []run
	lastCommit  telem.TimeStamp
	frames      int
	entry       log.Entry
	excluded    []ChannelKey
	sealed      []*segment.Handle
	keySegments []log.Slot
	// from and to hold the segment end of each key at the start and the end of the run
	// in commit.
	from, to []segmentEnd
	// whole and cuts hold the series of the frame in writeData and the byte offset that
	// each has written, or -1 for a series that it skips.
	whole []telem.Series
	cuts  []int
	// committed holds the keys of the run in progress, and commits and preKeys the
	// Commits of every run and the keys of their preloads, which pre holds.
	committed []ChannelKey
	commits   []*log.Commit
	preKeys   []ChannelKey
	pre       []state.Preload
}

// run is the samples that one Commit logs: their alignments touch, and their stamps
// rise strictly.
type run struct {
	alignment   telem.Alignment
	samples     uint32
	first, last telem.TimeStamp
	// from is the first stamp of the stored pointer that holds the stamp of the first
	// sample of a run of a writer that does not write its clock.
	from telem.TimeStamp
	// ends holds the segment end of each key, in the order of keys, at the end of a run
	// that a later run follows.
	ends []segmentEnd
}

// segmentEnd is the identity, stream length, and running checksum of the open segment
// of a key at the end of a run, or zero when the key has no open segment.
type segmentEnd struct {
	id     uint64
	length uint32
	crc    uint32
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

// openLogWriter opens a writer of cfg on the pointer log layout. The writer writes its
// clock and the channels that the clock times. The caller holds db.mu.
func (db *DB) openLogWriter(cfg WriterConfig) (*logWriter, error) {
	w := &logWriter{
		e:        db.engine,
		clock:    cfg.Clock,
		byKey:    make(map[ChannelKey]*logKey, len(cfg.Channels)),
		unstored: cfg.Unstored,
	}
	clock, ok := db.channelOf(cfg.Clock)
	if !ok {
		return nil, channel.NewNotFoundError(cfg.Clock)
	} else if !clock.IsIndex {
		return nil, channel.Wrap(cfg.Clock, errors.Wrap(
			validate.ErrValidation,
			"writer clock is not an index",
		))
	}
	w.clockCh = clock
	keys := make([]state.WriterKey, 0, len(cfg.Channels))
	for _, key := range cfg.Channels {
		ch, ok := db.channelOf(key)
		if !ok {
			return nil, channel.NewNotFoundError(key)
		}
		if ch.Clock() != cfg.Clock {
			return nil, channel.Wrap(key, errors.Wrapf(
				validate.ErrValidation,
				"channel is not timed by the writer clock <%d>",
				cfg.Clock,
			))
		}
		lk := &logKey{ch: ch, code: telem.CodeOf(ch.DataType)}
		w.byKey[key] = lk
		w.keys = append(w.keys, lk)
		keys = append(keys, state.WriterKey{Key: key, DataType: uint8(lk.code)})
	}
	if _, ok := w.byKey[w.clock]; !ok {
		w.stamps = &placer{e: db.engine, clock: w.clock, start: cfg.Start}
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
	stateCfg := state.WriterConfig{
		Keys:    keys,
		Clock:   w.clock,
		Start:   start,
		Bounded: cfg.Bounded,
	}
	stateKeys := make([]ChannelKey, len(keys))
	for i, k := range keys {
		stateKeys[i] = k.Key
	}
	slices.Sort(stateKeys)
	var st *state.Writer
	_, err := db.engine.withPreloads(nil, stateKeys, start, start, false, func(
		pre []state.Preload,
	) (err error) {
		st, err = db.engine.state.OpenWriter(stateCfg, pre...)
		return err
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
	if w.stamps != nil {
		return w.writeData(fr, uint32(n), skip)
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
	if refused, err := w.stretch(n, skip); err != nil || refused {
		return fr, refused, err
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
	w.extendRun(alignment, first, last, uint32(n))
	return fr.ExcludeKeys(w.excluded), len(w.excluded) > 0, nil
}

// extendRun adds n samples from alignment, with stamps from first to last, to the open
// run of the stretch of w.
func (w *logWriter) extendRun(
	alignment telem.Alignment,
	first, last telem.TimeStamp,
	n uint32,
) {
	if len(w.runs) == 0 {
		w.runs = append(w.runs, run{alignment: alignment, first: first})
	}
	r := &w.runs[len(w.runs)-1]
	r.last, r.samples = last, r.samples+n
	w.last = last
	w.next += n
}

// stretch makes room for n samples in the stretch of w. It starts a new stretch when w
// has none, or when n samples would pass the 2^32 - 1 samples of one, after a commit of
// the uncommitted run with skip. It returns true when skip refuses that commit.
func (w *logWriter) stretch(n int64, skip func(ChannelKey) bool) (bool, error) {
	if w.number != 0 && uint64(w.next)+uint64(n) > math.MaxUint32 {
		if _, refused, err := w.commit(skip); err != nil || refused {
			return refused, err
		}
		w.number = 0
	}
	if w.number != 0 {
		return false, nil
	}
	w.next = 0
	if w.unstored {
		a, err := w.e.unstored.Next()
		w.number = a.DomainIndex()
		return false, err
	}
	var err error
	w.number, err = w.st.NewAlignment()
	return false, err
}

// validate checks the series of fr for the keys of w: exactly one per key, of the key's
// type, and all of one length. It returns the clock series and the length, or zero when
// fr holds no series of w.
func (w *logWriter) validate(fr Frame) (telem.Series, int64, error) {
	var (
		clock telem.Series
		n     int64 = -1
		count int
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
			clock = s
		}
		count++
	}
	if n <= 0 {
		return clock, 0, nil
	}
	if count != len(w.keys) {
		return clock, 0, w.missingError()
	}
	return clock, n, nil
}

// missingError returns the error for a frame of the current write that leaves out a key
// of w.
func (w *logWriter) missingError() error {
	data := make([]Channel, 0, len(w.keys))
	for _, lk := range w.keys {
		if lk.ch.Key != w.clock {
			data = append(data, lk.ch)
		}
	}
	for _, lk := range w.keys {
		if lk.frame != w.frames {
			return missingChannelError(w.clockCh, lk.ch, data)
		}
	}
	return nil
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
	w.runs, w.number = w.runs[:0], 0
}

// commit makes the uncommitted runs visible and returns their end. It commits nothing
// and returns true when skip holds a key with samples to commit, and it commits no run
// when the state refuses one of them.
func (w *logWriter) commit(skip func(ChannelKey) bool) (telem.TimeStamp, bool, error) {
	if w.stamps != nil && w.stamps.err != nil {
		return 0, false, w.stamps.err
	}
	if len(w.runs) == 0 {
		return w.lastCommit, false, nil
	}
	if skip != nil {
		for _, lk := range w.keys {
			if !lk.skipped && skip(lk.ch.Key) {
				return 0, true, nil
			}
		}
	}
	if err := w.commitRuns(); err != nil {
		w.putBack()
		return 0, false, err
	}
	for _, lk := range w.keys {
		if lk.w != nil && lk.w.Len() != lk.committed {
			length, _ := lk.w.Commit()
			lk.committed = int64(length)
		}
	}
	end := w.runs[len(w.runs)-1].last + 1
	w.runs, w.lastCommit = w.runs[:0], end
	for _, lk := range w.keys {
		lk.skipped = false
	}
	err := w.sealFull()
	if err == nil && len(w.entry.Bytes()) > entryLimit {
		err = w.sync()
	}
	return end, false, err
}

// commitRuns logs one Commit for each run once the state accepts all of them. A writer
// that does not write its clock first reads the first and last stamps of each run.
func (w *logWriter) commitRuns() error {
	if w.stamps != nil {
		if err := w.readStamps(); err != nil {
			return err
		}
	}
	if w.unstored {
		return nil
	}
	w.commits, w.preKeys = w.commits[:0], w.preKeys[:0]
	w.from = w.ends(w.from, false)
	for i := range w.runs {
		r := &w.runs[i]
		to := r.ends
		if i == len(w.runs)-1 {
			w.to = w.ends(w.to, true)
			to = w.to
		}
		w.collectSlots(w.from, to)
		w.from = append(w.from[:0], to...)
		w.commits = append(w.commits, &log.Commit{
			Samples:   r.samples,
			Alignment: uint64(r.alignment),
			First:     int64(r.first),
			Last:      int64(r.last),
			Slots:     slices.Clone(w.keySegments),
		})
		w.preKeys = append(w.preKeys, w.committed...)
	}
	if w.stamps != nil {
		w.preKeys = append(w.preKeys, w.clock)
	}
	slices.Sort(w.preKeys)
	w.preKeys = slices.Compact(w.preKeys)
	first, last := w.runs[0].first, w.runs[len(w.runs)-1].last
	var err error
	w.pre, err = w.e.withPreloads(w.pre, w.preKeys, first, last, false, func(
		pre []state.Preload,
	) error {
		return w.st.Commit(w.commits, pre...)
	})
	if err != nil {
		return err
	}
	for _, c := range w.commits {
		w.entry.Add(c)
	}
	return nil
}

// readStamps sets the first and last stamps of each run of a writer that does not write
// its clock.
func (w *logWriter) readStamps() error {
	for i := range w.runs {
		// A later run starts on a later pointer, and the placer's pointer holds the
		// stamp of the last sample placed.
		to := w.stamps.cur.Last
		if i < len(w.runs)-1 {
			to = w.runs[i+1].from
		}
		r := &w.runs[i]
		var err error
		if r.first, r.last, err = w.e.stampEnds(w.clock, *r, to); err != nil {
			return err
		}
	}
	return nil
}

// ends sets dst to the segment end of each key, in the order of keys: at its written
// bytes when pending is true, and at its last commit otherwise.
func (w *logWriter) ends(dst []segmentEnd, pending bool) []segmentEnd {
	dst = dst[:0]
	for _, lk := range w.keys {
		var e segmentEnd
		if lk.w != nil {
			e.id, e.length = lk.id, uint32(lk.committed)
			if pending {
				e.length, e.crc = lk.w.Pending()
			}
		}
		dst = append(dst, e)
	}
	return dst
}

// collectSlots sets keySegments to a slot for each segment that grows from from to to,
// in rising order of segment, and committed to the keys of those segments in rising
// order.
func (w *logWriter) collectSlots(from, to []segmentEnd) {
	w.keySegments, w.committed = w.keySegments[:0], w.committed[:0]
	for i, lk := range w.keys {
		e := to[i]
		if e.id == 0 || (e.id == from[i].id && e.length == from[i].length) {
			continue
		}
		w.committed = append(w.committed, lk.ch.Key)
		w.keySegments = append(w.keySegments, log.Slot{
			Segment: e.id,
			Length:  e.length,
			TailCRC: e.crc,
		})
	}
	slices.SortFunc(w.keySegments, func(a, b log.Slot) int {
		return cmp.Compare(a.Segment, b.Segment)
	})
	slices.Sort(w.committed)
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
// durable record covers out of view, and logs only the Seals of each segment at its
// durable length.
func (w *logWriter) sync() error {
	handles := slices.Clone(w.sealed)
	for _, lk := range w.keys {
		if lk.dirty && lk.h != nil {
			handles = append(handles, lk.h)
		}
	}
	err := w.e.budget.Sync(handles)
	if errors.Is(err, segment.ErrDirSync) {
		w.e.fail(err)
	}
	if err != nil {
		seals, takeErr := w.st.TakeBack()
		w.entry.Reset()
		for i := range seals {
			w.entry.Add(&seals[i])
		}
		err = errors.Join(err, takeErr)
	} else {
		for _, lk := range w.keys {
			lk.dirty = false
		}
	}
	err = errors.Join(err, w.flush())
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
