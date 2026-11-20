// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"context"
	"sync/atomic"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/claim"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type WriterConfig struct {
	// Unstored makes the writer claim, stamp, and commit its series and write no byte
	// to disk. It does not check the order of an index series.
	// [OPTIONAL] - Defaults to false
	Unstored *bool
	// Bounded limits the writer's claim to the time it has committed, so that it can
	// write behind another writer. The writer fails to open when another writer holds
	// its start, and a commit fails when it reaches another claim.
	// [OPTIONAL] - Defaults to false
	Bounded *bool
	// EnableAutoCommit denotes whether each write is committed.
	//
	// [OPTIONAL] - Defaults to True
	EnableAutoCommit *bool
	// Start marks the start of the writer's claim and, unless Writer.SetDomainStart
	// sets another, of the domain it writes. A zero Start means the time the writer
	// opens. A non-zero Start fails the open when stored data holds it.
	// [OPTIONAL]
	Start telem.TimeStamp
	// End is an optional parameter that marks the ending bound of the domain. Defining
	// this parameter will allow the writer to write data to the domain without needing
	// to validate each call to Commit. If this parameter is not defined, Commit must be
	// called with a strictly increasing timestamp.
	// [OPTIONAL]
	End telem.TimeStamp
	// AutoIndexPersistInterval is the frequency at which the changes to index are
	// persisted to the disk.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan
}

var (
	_               config.Config[WriterConfig] = WriterConfig{}
	ErrWriterClosed                             = resource.NewClosedError(
		"unary.writer",
	)
)

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("unary.writer_config")
	v.NotNil("unstored", c.Unstored)
	v.NotNil("bounded", c.Bounded)
	v.NotNil("enable_auto_commit", c.EnableAutoCommit)
	v.Ternary(
		"end",
		!c.End.IsZero() && c.End.Before(c.Start),
		"end timestamp must be after or equal to start timestamp",
	)
	return v.Error()
}

// Override implements config.Config.
func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.End = override.Zero(c.End, other.End)
	c.Unstored = override.Nil(c.Unstored, other.Unstored)
	c.Bounded = override.Nil(c.Bounded, other.Bounded)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(
		c.AutoIndexPersistInterval,
		other.AutoIndexPersistInterval,
	)
	return c
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{
		Start:                    c.Start,
		End:                      c.End,
		EnableAutoCommit:         c.EnableAutoCommit,
		AutoIndexPersistInterval: c.AutoIndexPersistInterval,
	}
}

func (c WriterConfig) claimTimeRange(start telem.TimeStamp) telem.TimeRange {
	if *c.Bounded {
		return start.SpanRange(telem.Nanosecond)
	}
	return start.Range(lo.Ternary(c.End.IsZero(), telem.TimeStampMax, c.End))
}

// sharedWriter is the domain writer that the writers of one claim share, with the state
// that keeps their write positions consistent.
type sharedWriter struct {
	*domain.Writer
	// alignment tracks the position the next write will start at, as a packed domain
	// index (upper 32 bits) and sample index (lower 32 bits). Channels indexed by this
	// one read it to resolve their own alignment, so it is published after every write
	// rather than only at region creation. This field is accessed atomically because
	// Gate.Authorize and Gate.PeekResource return a shared pointer to this struct, and
	// the region's RWMutex is released before the caller accesses the field. This means
	// one goroutine may write alignment through Authorize while another reads it
	// through PeekResource concurrently.
	alignment atomic.Uint64
	// tracker holds per-domain offset state shared across every Writer of the claim. It
	// is created once when the resource is opened, so that per-sample byte offsets, the
	// cumulative session sample count, and the current domain start stay consistent
	// across writers. Reset on rollover via the domain.Writer's OnRollover hook.
	tracker *offsetTracker
	// highWaterMark is the last index timestamp that any owner persisted. Only valid
	// for index channels.
	highWaterMark telem.TimeStamp
	// sessionStart is the start of the resource's first domain since it last ended one.
	// Commits that resolve their end from the session sample count anchor on it.
	sessionStart telem.TimeStamp
	// alignments reserves the alignment domain of each domain after the first.
	alignments *telem.AlignmentCounter
}

func (w *sharedWriter) loadAlignment() telem.Alignment {
	return telem.Alignment(w.alignment.Load())
}

func (w *sharedWriter) storeAlignment(a telem.Alignment) {
	w.alignment.Store(uint64(a))
}

func (w *sharedWriter) setStart(start telem.TimeStamp) error {
	if err := w.SetStart(start); err != nil {
		return err
	}
	w.sessionStart = start
	w.tracker.currentStart = start
	return nil
}

func (w *sharedWriter) endDomain(ctx context.Context) error {
	if w.Len() == 0 {
		return nil
	}
	a, err := w.alignments.Next()
	if err != nil {
		return err
	}
	if err = w.Finish(ctx); err != nil {
		return err
	}
	w.tracker.reset()
	w.storeAlignment(a)
	return nil
}

type Writer struct {
	// claim is the writer's claim on the time range it writes.
	claim *claim.Claim[*sharedWriter]
	// idx stores the index of the unaryDB (rate or domain).
	idx *index.Domain
	// wrapError is a function that wraps any error originating from this writer to
	// provide context including the writer's channel key and name.
	wrapError func(error) error
	// Channel stores information about the channel this writer is writing to, including
	// but not limited to density and index.
	Channel channel.Channel
	cfg     WriterConfig
	// domainStart is where the domain starts if this writer persists its first
	// sample. Defaults to cfg.Start, or to the open time when cfg.Start is zero.
	domainStart telem.TimeStamp
	// closed stores whether the writer is closed. Operations like Write and Commit do
	// not succeed on closed writers.
	closed bool
}

func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (
	w *Writer,
	err error,
) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(WriterConfig{
		Unstored:                 new(false),
		Bounded:                  new(false),
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
	}, cfgs...)
	if err != nil {
		return nil, err
	}
	start := cfg.Start
	if start.IsZero() {
		start = telem.NewTimeStamp(db.cfg.Clock.Now())
	}
	w = &Writer{
		cfg:         cfg,
		Channel:     db.cfg.Channel,
		idx:         db.index(),
		wrapError:   db.wrapError,
		domainStart: start,
	}
	if w.claim, err = db.claims.Open(
		claim.Config[*sharedWriter]{
			Bounded:   *cfg.Bounded,
			TimeRange: cfg.claimTimeRange(start),
			Open: func() (*sharedWriter, error) {
				a, err := db.leadingAlignment.Next()
				if err != nil {
					return nil, err
				}
				cw := &sharedWriter{
					tracker:      db.resolver.newTracker(),
					sessionStart: start,
					alignments:   db.leadingAlignment,
				}
				cw.storeAlignment(a)
				domainCfg := cfg.domain()
				domainCfg.OnRollover = cw.tracker.rollover
				dw, err := db.domain.OpenWriter(ctx, domainCfg)
				cw.Writer = dw
				return cw, err
			},
		},
	); err != nil {
		return nil, w.wrapError(err)
	}
	return w, nil
}

// Write writes series to db in a domain that starts at start.
func Write(
	ctx context.Context,
	db *DB,
	start telem.TimeStamp,
	series telem.Series,
) (err error) {
	w, err := db.OpenWriter(ctx, WriterConfig{Start: start})
	if err != nil {
		return db.wrapError(err)
	}
	w.SetDomainStart(start)
	defer func() {
		err = db.wrapError(errors.Combine(err, w.Close()))
	}()
	if _, err = w.Write(series); err != nil {
		return err
	}
	_, err = w.Commit(ctx)
	return err
}

// Write validates and writes the given series, deriving its alignment from the
// channel's own sample position within the current domain. This is only meaningful for
// index channels, whose sample positions define the alignment space that every channel
// they index is expressed in. Data channels must use WriteAt.
func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	a, _, err := w.write(series, 0, true, nil)
	return a, err
}

// WriteUnless writes series as Write does, unless skip returns true for the channel. It
// calls skip first with no lock and skips at once on true, so a writer that skips takes
// no claim lock. Otherwise it calls skip again under the claim's lock, so the decision
// holds for the write. It returns true when it skipped the series.
func (w *Writer) WriteUnless(
	series telem.Series,
	skip func(channel.Key) bool,
) (telem.Alignment, bool, error) {
	return w.write(series, 0, true, skip)
}

// WriteAt validates and writes the given series, stamping it with a, an alignment
// resolved in the sample space of this channel's index. Returns the alignment of the
// first sample written.
func (w *Writer) WriteAt(
	series telem.Series,
	a telem.Alignment,
) (telem.Alignment, error) {
	a, _, err := w.write(series, a, false, nil)
	return a, err
}

// WriteAtUnless writes series as WriteAt does, unless skip returns true for the
// channel. It calls skip as WriteUnless does. It returns true when it skipped the
// series.
func (w *Writer) WriteAtUnless(
	series telem.Series,
	a telem.Alignment,
	skip func(channel.Key) bool,
) (telem.Alignment, bool, error) {
	return w.write(series, a, false, skip)
}

func (w *Writer) write(
	series telem.Series,
	a telem.Alignment,
	derive bool,
	skip func(channel.Key) bool,
) (telem.Alignment, bool, error) {
	if w.closed {
		return 0, false, w.wrapError(ErrWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, false, w.wrapError(err)
	}
	if skip != nil && skip(w.Channel.Key) {
		return 0, true, nil
	}
	skipped := false
	err := w.claim.Use(func(dw *sharedWriter) error {
		if skip != nil && skip(w.Channel.Key) {
			skipped = true
			return nil
		}
		mark := dw.highWaterMark
		stored := !*w.cfg.Unstored
		if w.Channel.IsIndex && stored {
			var err error
			if mark, err = checkOrder(mark, series); err != nil {
				return err
			}
		}
		if stored && dw.Len() == 0 {
			// Move first, so that no delete can reach the domain setStart checks.
			if err := w.claim.MoveStart(w.domainStart); err != nil {
				return err
			}
			if err := dw.setStart(w.domainStart); err != nil {
				return err
			}
		}
		if derive {
			// The tracker counts only stored samples, so an unstored writer derives
			// from the published cursor alone.
			a = dw.loadAlignment()
			if stored {
				a = telem.NewAlignment(
					a.DomainIndex(),
					uint32(dw.tracker.count(dw.Writer)),
				)
			}
		}
		// Publish where the next write lands before this one is on disk, so a channel
		// resolving its alignment against this one reserves the slot after this write
		// instead of colliding with it.
		dw.storeAlignment(a.AddSamples(uint32(series.Len())))
		if !stored {
			return nil
		}
		baseOffset := uint32(dw.tracker.domainBytes)
		dw.tracker.record(series.Data, baseOffset)
		if _, err := dw.Write(series.Data); err != nil {
			return err
		}
		dw.highWaterMark = mark
		return nil
	})
	if err != nil {
		return 0, false, w.wrapError(err)
	}
	return a, skipped, nil
}

// SetDomainStart sets where the writer's domain starts, in place of Start. It applies
// at the next persisted write, and only while the domain holds no data. The writer's
// claim moves to the start, unless the writer is bounded and the start comes after the
// claim.
func (w *Writer) SetDomainStart(start telem.TimeStamp) { w.domainStart = start }

// EndDomain ends the domain the writer holds, discarding any uncommitted samples. The
// next persisted write starts a new domain in a new alignment domain, at the start
// given to SetDomainStart. EndDomain does nothing when the domain holds no data.
func (w *Writer) EndDomain(ctx context.Context) error {
	if w.closed {
		return w.wrapError(ErrWriterClosed)
	}
	return w.wrapError(w.claim.Use(func(dw *sharedWriter) error {
		return dw.endDomain(ctx)
	}))
}

// checkOrder returns the last timestamp in series, or mark if series is empty. It
// returns a validation error if a timestamp is before mark or before the one that
// precedes it.
func checkOrder(mark telem.TimeStamp, series telem.Series) (telem.TimeStamp, error) {
	for i, ts := range series.Unmarshal[telem.TimeStamp]() {
		if ts < mark {
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"index out of order: timestamp %s at sample %d must not be before %s",
				ts,
				i,
				mark,
			)
		}
		mark = ts
	}
	return mark, nil
}

// Persist writes the index of the channel's domains to disk, so that a reopen of the DB
// sees every commit made before the call.
func (w *Writer) Persist() error {
	if w.closed {
		return w.wrapError(ErrWriterClosed)
	}
	return w.wrapError(w.claim.Resource().Persist())
}

// Commit commits the written series to the database.
func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.closed {
		return 0, w.wrapError(ErrWriterClosed)
	}
	ts, _, err := w.commitWithEnd(ctx, telem.TimeStamp(0), nil, nil)
	if err != nil {
		return 0, w.wrapError(err)
	}
	return ts, nil
}

// CommitWithEnd commits the written series with end as the end of the domain. When g is
// not nil, readers see the commit only once g publishes.
func (w *Writer) CommitWithEnd(
	ctx context.Context,
	end telem.TimeStamp,
	g *domain.Group,
) (err error) {
	_, err = w.CommitWithEndUnless(ctx, end, g, nil)
	return err
}

// CommitWithEndUnless commits as CommitWithEnd does, unless skip returns true for the
// channel. It calls skip first with no lock and skips at once on true. Otherwise it
// calls skip again under the claim's lock, so the decision holds for the commit. It
// returns true when it skipped the commit.
func (w *Writer) CommitWithEndUnless(
	ctx context.Context,
	end telem.TimeStamp,
	g *domain.Group,
	skip func(channel.Key) bool,
) (bool, error) {
	if w.closed {
		return false, w.wrapError(ErrWriterClosed)
	}
	if skip != nil && skip(w.Channel.Key) {
		return true, nil
	}
	_, skipped, err := w.commitWithEnd(ctx, end, g, skip)
	return skipped, w.wrapError(err)
}

func (w *Writer) commitWithEnd(
	ctx context.Context,
	end telem.TimeStamp,
	g *domain.Group,
	skip func(channel.Key) bool,
) (telem.TimeStamp, bool, error) {
	skipped := false
	err := w.claim.Use(func(dw *sharedWriter) error {
		if skip != nil && skip(w.Channel.Key) {
			skipped = true
			return nil
		}
		if end.IsZero() && w.Channel.IsIndex {
			end = dw.highWaterMark + 1
		} else if end.IsZero() {
			// Subtract 1 because we want the timestamp of the last written sample.
			// Anchor on the session start with the cumulative session sample count,
			// since after a rollover the per-domain start is "previous commit end +
			// 1ns" which is not a sample boundary in the index.
			approx, err := w.idx.Stamp(
				ctx,
				dw.sessionStart,
				dw.tracker.count(dw.Writer)-1,
				index.MustBeContinuous,
			)
			if err != nil {
				return err
			}
			if !approx.Exact() {
				return index.NewDiscontinuousStampError(dw.sessionStart)
			}
			// Add 1 to the end timestamp because the end timestamp is exclusive.
			end = approx.Lower + 1
		}
		if *w.cfg.Bounded {
			if err := w.claim.Reach(end); err != nil {
				return err
			}
		}
		if err := dw.Commit(ctx, end, g); err != nil {
			return err
		}
		dw.tracker.publish(end)
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return end, skipped, nil
}

// Close releases the writer's claim, and closes the shared domain writer when the claim
// was the last of its region.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	dw, last := w.claim.Release()
	if last {
		return w.wrapError(dw.Close())
	}
	return nil
}
