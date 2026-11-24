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

// Handoff closes w and opens a writer of cfg on the clock of w in its place. The new
// writer goes on in the run of w: its samples take the next alignments of the run, and
// its first stamp must come after the last stamp of w. Each key of cfg keeps the
// uncommitted samples that w or adopt holds for it, so the next commit of the new
// writer commits them. Handoff returns the uncommitted samples of the keys of w that
// cfg lacks as a Pending, or nil when they have none. adopt is a Pending of an earlier
// Handoff for the keys of cfg, or nil. On an error, Handoff closes w and adopt, which
// drops their uncommitted samples, and its error names the clock and their time range.
// On the pointer log layout, the new writer is the writer of w with the keys of cfg. On
// the old layout, Handoff refuses a start of cfg before the end of the last commit of
// w, and a handoff that adopts a Pending when w holds no uncommitted sample starts a
// new run. When w does not write its clock and cfg holds no channel, Handoff closes w,
// opens no writer, and refuses a non-nil adopt. It returns the uncommitted samples of w
// as a Pending, or nil when w has none or stores no samples.
func (db *DB) Handoff(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
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
	if cfg, err = w.handoffConfig(cfg); err != nil {
		return nil, nil, err
	}
	if w.log != nil {
		return db.handoffLog(ctx, w, cfg, adopt)
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
	next, pending, err := db.handoffIdx(ctx, w.idx, cfg, adopt)
	if err != nil {
		return nil, nil, err
	}
	w.closed = true
	return &Writer{WriterConfig: cfg, idx: next}, pending, nil
}

// handoffConfig returns cfg with its defaults. It refuses a cfg that does not write the
// clock of w, or that changes whether the writer stores samples.
func (w *Writer) handoffConfig(cfg WriterConfig) (WriterConfig, error) {
	cfg, err := config.New(WriterConfig{}, cfg)
	if err != nil {
		return cfg, err
	}
	if cfg.Clock != w.Clock || !slices.Contains(cfg.Channels, cfg.Clock) {
		return cfg, errors.Wrapf(
			validate.ErrValidation,
			"a handoff of the writer of clock %d writes that clock",
			w.Clock,
		)
	}
	if cfg.Unstored != w.Unstored {
		return cfg, errors.Wrapf(
			validate.ErrValidation,
			"a handoff of the writer of clock %d keeps whether it stores samples",
			w.Clock,
		)
	}
	return cfg, nil
}

// pend closes w, a writer of data alone, and returns its uncommitted samples as a
// Pending, or nil when it has none or stores no samples. It refuses a non-nil adopt.
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
		if err := w.log.readStamps(); err != nil {
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
	return tr, w.idx.hasUncommittedData || w.idx.dataPending
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
) (*idxWriter, *Pending, error) {
	if !old.hasUncommittedData && !old.dataPending {
		if adopt == nil && cfg.Start < old.lastCommitEnd {
			return nil, nil, errors.Wrapf(
				validate.ErrValidation,
				"cannot open writer because there is already data in the writer's "+
					"time range: start %v is before the end %v of the last commit",
				cfg.Start,
				old.lastCommitEnd,
			)
		}
		// The pending samples lie on stamps that old committed, so their run is not the
		// clock run of old.
		if adopt != nil {
			if err := errors.Join(old.persist(), old.Close()); err != nil {
				return nil, nil, err
			}
			clear(old.internal)
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
	if len(old.internal) == 0 {
		return next, nil, nil
	}
	p := old.runState(old.internal)
	p.dataPending, p.hasUncommittedData = old.hasUncommittedData ||
		old.dataPending, false
	if !p.dataPending {
		return next, nil, p.Close()
	}
	return next, &Pending{idx: p}, nil
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
