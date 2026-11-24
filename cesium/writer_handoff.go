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
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Pending holds the uncommitted samples of the channels that Handoff took out of a
// writer. They lie on stamps of the writer's clock. Pending is not safe for concurrent
// use.
type Pending struct {
	idx *idxWriter
	log *logWriter
}

// Commit commits the samples of p once the clock holds committed stamps for them, then
// releases p. It returns the end of the commit, or false and commits nothing while the
// clock does not hold them. On the pointer log layout, it fails when the clock holds
// other stamps there, and it returns false, with the samples committed, until their
// stamps are durable.
func (p *Pending) Commit(ctx context.Context) (telem.TimeStamp, bool, error) {
	if p.log != nil {
		return p.commitLog(ctx)
	}
	if p.idx.hasUncommittedData {
		return p.commitData(ctx)
	}
	held, err := p.idx.indexHoldsHighWater(ctx)
	if err != nil || !held {
		return 0, false, err
	}
	end, _, err := p.idx.commitPendingData(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	return end, true, p.Close()
}

// commitData commits the samples of p, which a writer of data alone left, once the
// index holds a stamp for each of them, then releases p.
func (p *Pending) commitData(ctx context.Context) (telem.TimeStamp, bool, error) {
	_, count, err := p.idx.idx.Coverage(ctx, p.idx.start.Range(telem.TimeStampMax))
	if err != nil || count < p.idx.sampleCount {
		return 0, false, err
	}
	end, _, err := p.idx.Commit(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	return end, true, p.Close()
}

// commitAlone commits the samples of p, which a writer of data alone left, and refuses
// them while clock holds no stamps for them.
func (p *Pending) commitAlone(ctx context.Context, clock ChannelKey) error {
	_, held, err := p.Commit(ctx)
	if err != nil || held {
		return err
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"clock %d holds no stamps for the pending samples",
		clock,
	)
}

// Close releases the channels of p and drops its uncommitted samples.
func (p *Pending) Close() error {
	if p.log != nil {
		return p.log.close()
	}
	return errors.Join(p.idx.persist(), p.idx.Close())
}

// HandoffRun is the run of the writer that Handoff opens.
type HandoffRun uint8

const (
	// NewRun starts a new run, for a handoff that moves the clock to another session.
	NewRun HandoffRun = iota
	// SameRun goes on in the run of the writer that Handoff replaces, for a handoff
	// that keeps the clock with the same session: the samples take the next alignments
	// of the run, and the first stamp must come after the last stamp of the run.
	SameRun
)

// Handoff closes w and opens a writer of cfg on the clock of w in its place, in the run
// that run names. Each key of cfg keeps the uncommitted samples that w or adopt holds
// for it, so the next commit of the new writer commits them, and the new writer keeps
// the last clock stamp of the writer that wrote them. Handoff returns the uncommitted
// samples of the keys of w that cfg lacks as a Pending, or nil when they have none.
// adopt is a Pending of an earlier Handoff for the keys of cfg, or nil. On an error,
// Handoff closes w and adopt, which drops their uncommitted samples, and its error
// names the clock and their time range. A handoff refuses a start of cfg in stored
// data, as an open does. On the old layout, a handoff that adopts a Pending when w
// holds no uncommitted sample starts a new run. On the pointer log layout, the new
// writer is the writer of w with the keys of cfg. On both layouts, run applies only
// when w holds no uncommitted sample: the uncommitted run of w goes on in either run.
// When w does not write its clock and cfg holds no channel, Handoff closes w, opens no
// writer, and refuses a non-nil adopt. It returns the uncommitted samples of w as a
// Pending, or nil when w has none or stores no samples.
func (db *DB) Handoff(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (_ *Writer, _ *Pending, err error) {
	dropped, uncommitted := w.uncommitted()
	defer func() {
		if err != nil && uncommitted {
			err = errors.Wrapf(
				err,
				"dropped the uncommitted samples of clock %d in %s",
				w.Clock,
				dropped,
			)
		}
		if err != nil {
			err = errors.Join(err, w.Close())
			if adopt != nil {
				err = errors.Join(err, adopt.Close())
			}
		}
	}()
	if db.closed.Load() {
		return nil, nil, ErrDBClosed
	}
	if err = w.check(); err != nil {
		return nil, nil, err
	}
	if len(cfg.Channels) == 0 && !slices.Contains(w.Channels, w.Clock) {
		var pending *Pending
		pending, err = w.pend(adopt)
		return nil, pending, err
	}
	if cfg, err = config.New(WriterConfig{}, cfg); err != nil {
		return nil, nil, err
	}
	if err = checkHandoff(w, cfg, run); err != nil {
		return nil, nil, err
	}
	if w.log != nil {
		return db.handoffLog(ctx, w, cfg, adopt, run)
	}
	if adopt != nil && adopt.idx.hasUncommittedData {
		// The old layout matches data to its index by position, so the samples of a
		// writer of data alone commit on their own.
		if err = adopt.commitAlone(ctx, w.Clock); err != nil {
			return nil, nil, err
		}
		adopt = nil
	}
	if err = w.persist(); err != nil {
		return nil, nil, err
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	next, pending, err := db.handoffIdx(ctx, w.idx, cfg, adopt, run)
	if err != nil {
		return nil, nil, err
	}
	w.closed = true
	return &Writer{WriterConfig: cfg, idx: next}, pending, nil
}

// checkHandoff refuses a handoff of w to cfg in run that changes the clock or whether
// the writer stores samples, or that names an unknown run.
func checkHandoff(w *Writer, cfg WriterConfig, run HandoffRun) error {
	if cfg.Clock != w.Clock || !slices.Contains(cfg.Channels, cfg.Clock) {
		return errors.Wrapf(
			validate.ErrValidation,
			"a handoff of the writer of clock %d writes that clock",
			w.Clock,
		)
	}
	if run != NewRun && run != SameRun {
		return errors.Wrapf(validate.ErrValidation, "unknown handoff run %d", run)
	}
	if cfg.Unstored != w.Unstored {
		return errors.Wrapf(
			validate.ErrValidation,
			"a handoff of the writer of clock %d keeps whether it stores samples",
			w.Clock,
		)
	}
	return nil
}

// pend closes w, a writer of data alone, and returns its uncommitted samples as a
// Pending, or nil when it has none or stores no samples. It refuses a non-nil adopt. On
// the pointer log layout, the Pending claims its keys over the stamps of its samples.
func (w *Writer) pend(adopt *Pending) (*Pending, error) {
	if adopt != nil {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"a handoff with no channels adopts no pending samples",
		)
	}
	if _, uncommitted := w.uncommitted(); !uncommitted || w.Unstored {
		return nil, w.Close()
	}
	if w.log != nil {
		runs := w.log.runs
		if err := w.log.readStamps(); err != nil {
			return nil, err
		}
		err := w.log.st.Bound(runs[0].first, runs[len(runs)-1].last)
		if err != nil {
			return nil, err
		}
		w.closed = true
		return &Pending{log: w.log}, nil
	}
	if err := w.persist(); err != nil {
		return nil, err
	}
	w.closed = true
	return &Pending{idx: w.idx}, nil
}

// uncommitted returns the time range of the samples that w has not committed, and false
// when it has none.
func (w *Writer) uncommitted() (telem.TimeRange, bool) {
	if w.log != nil {
		runs := w.log.runs
		if len(runs) == 0 {
			return telem.TimeRange{}, false
		}
		return telem.TimeRange{
			Start: runs[0].first,
			End:   runs[len(runs)-1].last + 1,
		}, true
	}
	tr := telem.TimeRange{Start: w.idx.lastCommitEnd, End: w.idx.idx.highWaterMark + 1}
	return tr, w.idx.holdsUncommitted()
}

// handoffIdx moves the unary writers of the keys of cfg out of old and adopt into a new
// index writer of cfg, and returns the writers of the other keys of old that hold
// uncommitted samples as a Pending. The new writer takes the run state of adopt when it
// is not nil, or else of old. When old holds no uncommitted sample, the keys of an
// adopting writer open again at the start of cfg. The caller holds db.mu.
func (db *DB) handoffIdx(
	ctx context.Context,
	old *idxWriter,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (*idxWriter, *Pending, error) {
	if !old.holdsUncommitted() {
		next, done, err := db.handoffIdle(ctx, old, cfg, adopt, run)
		if done || err != nil {
			return next, nil, err
		}
	}
	base := old
	if adopt != nil {
		base = adopt.idx
	}
	next := base.runState(make(map[ChannelKey]*unaryWriterState, len(cfg.Channels)))
	next.writingToIdx = true
	// The clock run that old holds moves with the clock, so its samples join the run.
	if adopt != nil && old.hasUncommittedData {
		next.hasUncommittedData = true
		next.idx.highWaterMark = max(next.idx.highWaterMark, old.idx.highWaterMark)
		next.sampleCount = max(next.sampleCount, old.sampleCount)
	}
	for _, key := range cfg.Channels {
		if next.take(old, key) || (adopt != nil && next.take(adopt.idx, key)) {
			continue
		}
		if err := db.openChannelWriter(ctx, cfg, next, key); err != nil {
			return nil, nil, errors.Join(err, next.Close())
		}
	}
	if adopt != nil && len(adopt.idx.internal) > 0 {
		return nil, nil, errors.Join(
			errors.Wrap(
				validate.ErrValidation,
				"the writer lacks a key of the pending",
			),
			next.Close(),
		)
	}
	// A run that skipped data channels held back is the new writer's to commit once the
	// handoff drops every one of them.
	for _, key := range old.held {
		if slices.Contains(cfg.Channels, key) {
			next.held = append(next.held, key)
		}
	}
	if len(old.held) > 0 && len(next.held) == 0 {
		next.hasUncommittedData = true
	}
	p, err := pendRest(old)
	return next, p, err
}

// pendRest returns the writers that remain in old after a handoff took its keys as a
// Pending, with those that hold no uncommitted sample closed, or nil when none remain.
func pendRest(old *idxWriter) (*Pending, error) {
	var err error
	for key, u := range old.internal {
		if !u.uncommitted {
			err = errors.Join(err, u.Close())
			delete(old.internal, key)
		}
	}
	if err != nil || len(old.internal) == 0 {
		return nil, err
	}
	p := old.runState(old.internal)
	p.dataPending, p.hasUncommittedData = true, false
	return &Pending{idx: p}, nil
}

// handoffIdle hands off old, which holds no uncommitted sample. In a new run, or when
// it adopts samples, it closes old, and with nothing to adopt it returns done with a
// new writer of cfg. In the same run, it keeps old open after the check of
// refuseStored.
func (db *DB) handoffIdle(
	ctx context.Context,
	old *idxWriter,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (*idxWriter, bool, error) {
	// The pending samples of an adopting handoff lie on stamps that old committed, so
	// their run is not the clock run of old.
	if run == SameRun && adopt == nil {
		return nil, false, db.refuseStored(ctx, old, cfg)
	}
	if err := errors.Join(old.persist(), old.Close()); err != nil {
		return nil, false, err
	}
	clear(old.internal)
	if adopt != nil {
		return nil, false, nil
	}
	next, err := db.openClockWriter(ctx, cfg)
	return next, true, err
}

// refuseStored refuses a handoff from old to cfg in the same run when the start of cfg
// lies in stored data of a key that old holds, as an open of cfg would.
func (db *DB) refuseStored(
	ctx context.Context,
	old *idxWriter,
	cfg WriterConfig,
) error {
	tr := cfg.Start.SpanRange(telem.Nanosecond)
	for _, key := range cfg.Channels {
		if _, held := old.internal[key]; !held {
			continue
		}
		u := db.mu.dbs.unary[key]
		stored, err := u.HasStoredDataFor(ctx, tr)
		if err != nil {
			return err
		}
		if stored {
			return channel.Wrap(key, errors.Wrapf(
				validate.ErrValidation,
				"cannot open writer because there is already data in the writer's "+
					"time range at %v",
				cfg.Start,
			))
		}
	}
	return nil
}

// runState returns an index writer of internal with the clock and run state of w.
func (w *idxWriter) runState(internal map[ChannelKey]*unaryWriterState) *idxWriter {
	return &idxWriter{
		internal:           internal,
		idx:                w.idx,
		sampleCount:        w.sampleCount,
		start:              w.start,
		anchor:             w.anchor,
		anchorSet:          w.anchorSet,
		hasUncommittedData: w.hasUncommittedData,
		dataPending:        w.dataPending,
		lastCommitEnd:      w.lastCommitEnd,
	}
}

// take moves the unary writer of key from src into w, and returns false when src has
// none.
func (w *idxWriter) take(src *idxWriter, key ChannelKey) bool {
	u, ok := src.internal[key]
	if !ok {
		return false
	}
	delete(src.internal, key)
	u.lastWriteCall = 0
	w.internal[key] = u
	return true
}
