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
type Pending struct{ log *logWriter }

// Commit commits the samples of p once the clock holds the exact stamps of each of
// their runs, then releases p once those stamps are durable. It returns the end of the
// commit, or false while the clock does not hold those stamps or they are not durable,
// and it fails when the clock holds other stamps there. It never waits.
func (p *Pending) Commit(ctx context.Context) (telem.TimeStamp, bool, error) {
	return p.commitLog(ctx)
}

// Close releases the channels of p and drops its uncommitted samples.
func (p *Pending) Close() error { return p.log.close() }

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

// Handoff moves w to the keys of cfg on the pointer log layout, and returns the new
// writer. Each key of cfg keeps the uncommitted samples that w or adopt holds for it,
// so the next commit of the new writer commits them, and the uncommitted run of w goes
// on in either run; run applies only when w holds no uncommitted sample. Handoff
// returns the uncommitted samples of the keys of w that cfg lacks as a Pending, or nil
// when they have none. adopt is a Pending of an earlier Handoff for the keys of cfg, or
// nil. On an error, Handoff closes w and adopt, which drops their uncommitted samples,
// and its error names the clock and their time range. A handoff refuses a start of cfg
// in stored data, as an open does. Handoff fails on the old layout, where each writer
// of a clock stays open. When w does not write its clock and cfg holds no channel,
// Handoff closes w, opens no writer, and refuses a non-nil adopt. It returns the
// uncommitted samples of w as a Pending, or nil when w has none or stores no samples.
func (db *DB) Handoff(
	ctx context.Context,
	w *Writer,
	cfg WriterConfig,
	adopt *Pending,
	run HandoffRun,
) (_ *Writer, _ *Pending, err error) {
	if w.log == nil {
		err = errors.Join(
			errors.Wrap(errors.ErrUnsupported, "the old layout hands off no writer"),
			w.Close(),
		)
		if adopt != nil {
			err = errors.Join(err, adopt.Close())
		}
		return nil, nil, err
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
	return db.handoffLog(ctx, w, cfg, adopt, run)
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
// Pending, or nil when it has none or stores no samples. It refuses a non-nil adopt.
// The Pending claims its keys over the stamps of its samples.
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
	runs := w.log.runs
	if err := w.log.readStamps(); err != nil {
		return nil, err
	}
	if err := w.log.st.Bound(runs[0].first, runs[len(runs)-1].last); err != nil {
		return nil, err
	}
	w.closed = true
	return &Pending{log: w.log}, nil
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
