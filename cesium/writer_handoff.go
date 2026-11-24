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
// that run names. It fails when w has uncommitted samples or adopt is not nil, and it
// returns no Pending. On an error, Handoff closes w and adopt, and its error names the
// clock and the time range of the samples it dropped. A handoff refuses a start of cfg
// in stored data, as an open does. Handoff fails on the old layout, where each writer
// of a clock stays open.
func (db *DB) Handoff(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (_ *Writer, _ *Pending, err error) {
	if w.log == nil {
		return nil, nil, errors.Join(
			errors.Wrap(errors.ErrUnsupported, "the old layout hands off no writer"),
			w.Close(),
		)
	}
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
	if cfg, err = config.New(WriterConfig{}, cfg); err != nil {
		return nil, nil, err
	}
	if err = checkHandoff(w, cfg, run); err != nil {
		return nil, nil, err
	}
	next, err := db.handoffLog(ctx, w, cfg, adopt, run)
	return next, nil, err
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

// handoffLog closes w and opens a writer of cfg in run. It fails when w has uncommitted
// samples, which the pointer log layout does not move.
func (db *DB) handoffLog(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (*Writer, error) {
	if adopt != nil {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"a handoff on the pointer log layout adopts no pending samples",
		)
	}
	if len(w.log.runs) > 0 {
		return nil, errors.Wrap(
			errors.ErrUnsupported,
			"the pointer log layout does not move uncommitted samples",
		)
	}
	old := *w.log
	if err := w.Close(); err != nil {
		return nil, err
	}
	next, err := db.OpenWriter(ctx, cfg)
	if err != nil || run != SameRun {
		return next, err
	}
	next.log.number, next.log.next = old.number, old.next
	next.log.last, next.log.lastCommit = old.last, old.lastCommit
	return next, nil
}

// uncommitted returns the time range of the samples that w has not committed, and false
// when it has none.
func (w *Writer) uncommitted() (telem.TimeRange, bool) {
	runs := w.log.runs
	if len(runs) == 0 {
		return telem.TimeRange{}, false
	}
	return telem.TimeRange{Start: runs[0].first, End: runs[len(runs)-1].last + 1}, true
}
