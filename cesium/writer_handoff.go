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
// writer. They lie on clock stamps that the writer wrote. Pending is not safe for
// concurrent use.
type Pending struct{ idx *idxWriter }

// Commit commits the samples of p once the clock holds a committed stamp at the last
// clock stamp of the writer that wrote them, then releases p. It returns the end of the
// commit, or false and commits nothing while the clock does not hold that stamp.
func (p *Pending) Commit(ctx context.Context) (telem.TimeStamp, bool, error) {
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

// Close releases the channels of p and drops its uncommitted samples.
func (p *Pending) Close() error { return errors.Join(p.idx.persist(), p.idx.Close()) }

// Handoff closes w and opens a writer of cfg on the clock of w in its place. Each key
// of cfg keeps the uncommitted samples that w or adopt holds for it, so the next commit
// of the new writer commits them, and the new writer keeps the last clock stamp of the
// writer that wrote them. Handoff returns the uncommitted samples of the keys of w that
// cfg lacks as a Pending, or nil when they have none. adopt is a Pending of an earlier
// Handoff for the keys of cfg, or nil. On an error, Handoff closes w and adopt, which
// drops their uncommitted samples. On the pointer log layout, Handoff closes w, which
// drops its uncommitted samples, opens a writer of cfg, and returns no Pending.
func (db *DB) Handoff(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
) (_ *Writer, _ *Pending, err error) {
	defer func() {
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
	if cfg, err = config.New(WriterConfig{}, cfg); err != nil {
		return nil, nil, err
	}
	if cfg.Clock != w.Clock || !slices.Contains(cfg.Channels, cfg.Clock) {
		return nil, nil, errors.Wrapf(
			validate.ErrValidation,
			"a handoff of the writer of clock %d writes that clock",
			w.Clock,
		)
	}
	if w.log != nil {
		if adopt != nil {
			return nil, nil, errors.Wrap(
				validate.ErrValidation,
				"a handoff on the pointer log layout adopts no pending samples",
			)
		}
		if err = w.Close(); err != nil {
			return nil, nil, err
		}
		next, err := db.OpenWriter(ctx, cfg)
		return next, nil, err
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

// handoffIdx moves the unary writers of the keys of cfg out of old and adopt into a new
// index writer of cfg, and returns the writers of the other keys of old as a Pending.
// The new writer takes the run state of adopt when it is not nil, or else of old. The
// caller holds db.mu.
func (db *DB) handoffIdx(
	ctx context.Context,
	old *idxWriter,
	cfg WriterConfig,
	adopt *Pending,
) (*idxWriter, *Pending, error) {
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
