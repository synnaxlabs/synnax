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
	"context"
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
// log layout. Write appends with no sync, Commit makes the written samples visible, and
// Sync makes the commits durable with one log entry.
type logWriter struct {
	e  *engine
	st *state.Writer
	// history reports the time range of a commit before the newest data of a key.
	history func(context.Context, HistoryChange)
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
	from       uint32
	first, end telem.TimeStamp
	samples    uint32
	lastCommit telem.TimeStamp
	// since is the end of the last commit of the stretch, or zero before its first.
	since       telem.TimeStamp
	frames      int
	entry       log.Entry
	excluded    []ChannelKey
	sealed      []*segment.Handle
	keySegments []log.Slot
	// committed holds the keys of the commit in progress, and pre their preloads.
	committed []ChannelKey
	pre       []state.Preload
	// backfilled holds the keys of the commit in progress that readers see data of
	// after it.
	backfilled []ChannelKey
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
// clock and the channels that the clock times. It holds db.mu only while it reads the
// channels of cfg, since the open of the state writer can wait.
func (db *DB) openLogWriter(ctx context.Context, cfg WriterConfig) (*logWriter, error) {
	w := &logWriter{
		e:        db.engine,
		clock:    cfg.Clock,
		history:  db.historyChanges.Notify,
		byKey:    make(map[ChannelKey]*logKey, len(cfg.Channels)),
		unstored: cfg.Unstored,
	}
	stateCfg, err := db.readLogChannels(w, cfg)
	if err != nil {
		return nil, err
	}
	if w.unstored {
		return w, nil
	}
	if err = w.openState(ctx, db, cfg, stateCfg); err != nil {
		return nil, err
	}
	return w, nil
}

// readLogChannels reads the channel of each key of cfg into w under db.mu, and returns
// the config of its state writer with no start. It returns ErrDBClosed when db is
// closed.
func (db *DB) readLogChannels(
	w *logWriter,
	cfg WriterConfig,
) (state.WriterConfig, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.closed.Load() {
		return state.WriterConfig{}, ErrDBClosed
	}
	if clock, ok := db.channelOf(cfg.Clock); !ok {
		return state.WriterConfig{}, channel.NewNotFoundError(cfg.Clock)
	} else if !clock.IsIndex {
		return state.WriterConfig{}, channel.Wrap(cfg.Clock, errors.Wrap(
			validate.ErrValidation,
			"writer clock is not an index",
		))
	}
	keys := make([]state.WriterKey, 0, len(cfg.Channels))
	for _, key := range cfg.Channels {
		ch, ok := db.channelOf(key)
		if !ok {
			return state.WriterConfig{}, channel.NewNotFoundError(key)
		}
		if ch.Clock() != cfg.Clock {
			return state.WriterConfig{}, channel.Wrap(key, errors.Wrapf(
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
		return state.WriterConfig{}, errors.Wrap(
			errors.ErrUnsupported,
			"a writer of the pointer log writes its clock",
		)
	}
	slices.SortStableFunc(w.keys, func(a, b *logKey) int {
		return boolOrder(b.ch.Key == w.clock) - boolOrder(a.ch.Key == w.clock)
	})
	return state.WriterConfig{
		Keys:  keys,
		Clock: w.clock,
		Drops: db.engine.state.Drops(),
	}, nil
}

// openState opens the state writer of w on stateCfg, with the start and bound of cfg,
// and seals each open segment of its keys that the state hands over, since a writer
// starts a new segment for each key.
func (w *logWriter) openState(
	ctx context.Context,
	db *DB,
	cfg WriterConfig,
	stateCfg state.WriterConfig,
) error {
	stateCfg.Start = cfg.Start
	if stateCfg.Start.IsZero() {
		stateCfg.Start = telem.NewTimeStamp(db.clock.Now())
	}
	stateCfg.Bounded = cfg.Bounded
	start := stateCfg.Start
	stateKeys := make([]ChannelKey, len(stateCfg.Keys))
	for i, k := range stateCfg.Keys {
		stateKeys[i] = k.Key
	}
	slices.Sort(stateKeys)
	var st *state.Writer
	_, err := db.engine.withPreloads(nil, stateKeys, start, start, false, func(
		pre []state.Preload,
	) (err error) {
		st, err = db.engine.state.OpenWriter(ctx, stateCfg, pre...)
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
func (w *logWriter) write(
	ctx context.Context,
	fr Frame,
	skip func(ChannelKey) bool,
) (Frame, bool, error) {
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
	if refused, err := w.stretch(ctx, n, skip); err != nil || refused {
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
			if err = w.append(ctx, lk, s.Data); err != nil {
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

// stretch makes room for n samples in the stretch of w. It starts a new stretch when w
// has none, or when n samples would pass the 2^32 - 1 samples of one, after a commit of
// the uncommitted run with skip. It returns true when skip refuses that commit.
func (w *logWriter) stretch(
	ctx context.Context,
	n int64,
	skip func(ChannelKey) bool,
) (bool, error) {
	if w.number != 0 && uint64(w.next)+uint64(n) > math.MaxUint32 {
		if _, refused, err := w.commit(ctx, skip); err != nil || refused {
			return refused, err
		}
		w.number = 0
	}
	if w.number != 0 {
		return false, nil
	}
	w.next, w.since = 0, 0
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
		// The clock takes int64 stamps too, as the index writer of the old layout does.
		if err := lk.ch.ValidateSeries(s); err != nil {
			return clock, 0, channel.Wrap(k, err)
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
	data := make([]Channel, 0, len(w.keys)-1)
	for _, lk := range w.keys[1:] {
		data = append(data, lk.ch)
	}
	for _, lk := range w.keys {
		if lk.frame != w.frames {
			return missingChannelError(w.keys[0].ch, lk.ch, data)
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
				"index out of order: timestamp %v at sample %d must be after %v",
				stamp,
				i,
				prev,
			))
		}
		// A time range ends after its last stamp, so none holds the largest one.
		if stamp == telem.TimeStampMax {
			return 0, 0, channel.Wrap(w.clock, errors.Wrapf(
				validate.ErrValidation,
				"stamp %v must be before %v",
				stamp,
				telem.TimeStampMax,
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
func (w *logWriter) append(ctx context.Context, lk *logKey, p []byte) error {
	if lk.w == nil {
		select {
		case <-w.e.ready:
		case <-ctx.Done():
			return ctx.Err()
		}
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
		h, err := w.e.budget.CreateHeld(w.e.fs, segmentName(lk.ch.Key, id))
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
// returns true when skip holds a key with samples to commit. It reports to history each
// key whose committed samples lie before data that readers see.
func (w *logWriter) commit(
	ctx context.Context,
	skip func(ChannelKey) bool,
) (telem.TimeStamp, bool, error) {
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
		if err := w.collectSlots(); err != nil {
			w.putBack()
			return 0, false, err
		}
		c.Slots = slices.Clone(w.keySegments)
		var err error
		w.pre, err = w.e.withPreloads(w.pre, w.committed, w.first, w.end, false, func(
			pre []state.Preload,
		) error {
			return w.st.Commit(c, pre...)
		})
		if err != nil {
			w.putBack()
			return 0, false, err
		}
		for _, lk := range w.keys {
			if lk.w != nil && lk.w.Len() != lk.committed {
				length, _ := lk.w.Commit()
				lk.committed = int64(length)
			}
		}
		// A commit that goes on in a stretch changes the time from the end of the
		// commit before it, as the domain of the old layout does.
		start := w.first
		if w.since != 0 {
			start = w.since
		}
		for _, key := range w.backfilled {
			w.history(ctx, HistoryChange{Key: key, TimeRange: start.Range(end)})
		}
		w.entry.Add(c)
	}
	w.samples, w.lastCommit, w.since = 0, end, end
	for _, lk := range w.keys {
		lk.skipped = false
	}
	err := w.sealFull()
	if err == nil && len(w.entry.Bytes()) > entryLimit {
		err = w.sync()
	}
	return end, false, err
}

// collectSlots sets keySegments to a slot for each segment with uncommitted samples, in
// rising order of segment, committed to the keys of those segments in rising order, and
// backfilled to those keys that readers see data of after the uncommitted run.
func (w *logWriter) collectSlots() error {
	w.keySegments, w.committed = w.keySegments[:0], w.committed[:0]
	w.backfilled = w.backfilled[:0]
	for _, lk := range w.keys {
		if lk.w == nil || lk.w.Len() == lk.committed {
			continue
		}
		w.committed = append(w.committed, lk.ch.Key)
		after, err := w.e.storedAfter(lk.ch.Key, w.end)
		if err != nil {
			return err
		}
		if after {
			w.backfilled = append(w.backfilled, lk.ch.Key)
		}
		length, crc := lk.w.Pending()
		w.keySegments = append(w.keySegments, log.Slot{
			Segment: lk.id,
			Length:  length,
			TailCRC: crc,
		})
	}
	slices.SortFunc(w.keySegments, func(a, b log.Slot) int {
		return cmp.Compare(a.Segment, b.Segment)
	})
	slices.Sort(w.committed)
	return nil
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
func (w *logWriter) endStretch(ctx context.Context) error {
	if _, _, err := w.commit(ctx, nil); err != nil {
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

// close syncs, releases the claims of w, and closes its segment files. An unstored
// writer holds none of them.
func (w *logWriter) close() error {
	if w.unstored {
		return nil
	}
	err := w.sync()
	err = errors.Join(err, w.st.Close())
	for _, lk := range w.keys {
		if lk.h != nil {
			err = errors.Join(err, lk.h.Close())
		}
	}
	return errors.Join(err, w.e.reap())
}
