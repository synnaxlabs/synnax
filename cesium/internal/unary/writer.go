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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type WriterConfig struct {
	// Persist denotes whether the writer writes its data to FS. If Persist is off, no
	// data is written.
	// [OPTIONAL] - Defaults to true
	Persist *bool
	// Bounded limits the writer's control region to the time it has committed, so that
	// it can write behind another writer. The writer fails to open when another writer
	// holds its start, and a commit fails when it reaches another region. A bounded
	// region reports no control transfers.
	// [OPTIONAL] - Defaults to false
	Bounded *bool
	// EnableAutoCommit denotes whether each write is committed.
	//
	// [OPTIONAL] - Defaults to True
	EnableAutoCommit *bool
	// ErrOnUnauthorizedOpen controls whether the writer will return an error on open
	// when attempting to write to a channel that is does not have authority over.
	// [OPTIONAL] - Defaults to false
	ErrOnUnauthorizedOpen *bool
	// Subject is the control subject held by the writer.
	// [REQUIRED]
	Subject xcontrol.Subject
	// Start marks the start of the writer's control region and, unless
	// Writer.SetDomainStart sets another, of the domain it writes. A zero Start means
	// the time the writer opens. A non-zero Start fails the open when stored data holds
	// it.
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
	// Authority is the control authority held by the writer: higher authority entities
	// have priority access to the region.
	// [OPTIONAL]
	Authority xcontrol.Authority
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	defaultWriterConfig                             = WriterConfig{
		Persist:                  new(true),
		Bounded:                  new(false),
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
		ErrOnUnauthorizedOpen:    new(false),
	}
	ErrWriterClosed = resource.NewClosedError("unary.writer")
)

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("unary.writer_config")
	v.NotEmptyString("subject.key", c.Subject.Key)
	v.NotNil("err_on_unauthorized_open", c.ErrOnUnauthorizedOpen)
	v.NotNil("persist", c.Persist)
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
	c.Subject = override.If(c.Subject, other.Subject, other.Subject.Key != "")
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Persist = override.Nil(c.Persist, other.Persist)
	c.Bounded = override.Nil(c.Bounded, other.Bounded)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(
		c.AutoIndexPersistInterval,
		other.AutoIndexPersistInterval,
	)
	c.ErrOnUnauthorizedOpen = override.Nil(
		c.ErrOnUnauthorizedOpen,
		other.ErrOnUnauthorizedOpen,
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

func (c WriterConfig) controlTimeRange(start telem.TimeStamp) telem.TimeRange {
	if *c.Bounded {
		return start.SpanRange(telem.Nanosecond)
	}
	return start.Range(lo.Ternary(c.End.IsZero(), telem.TimeStampMax, c.End))
}

// controlledWriter is used for exchanging control between multiple unary writers. When
// control is transferred, ownership of the domain writer is moved to the new unary
// writer. Additional state is included to ensure that write positions and channel
// information are consistent.
type controlledWriter struct {
	*domain.Writer
	channelKey channel.Key
	// alignment tracks the position the next write will start at, as a packed domain
	// index (upper 32 bits) and sample index (lower 32 bits). Channels indexed by this
	// one read it to resolve their own alignment, so it is published after every write
	// rather than only at region creation. This field is accessed atomically because
	// Gate.Authorize and Gate.PeekResource return a shared pointer to this struct, and
	// the region's RWMutex is released before the caller accesses the field. This means
	// one goroutine may write alignment through Authorize while another reads it
	// through PeekResource concurrently.
	alignment atomic.Uint64
	// tracker holds per-domain offset state shared across every Writer that takes
	// control of this resource. It is created once when the resource is opened and
	// survives control handoffs (where a higher-authority Writer joins an existing
	// region) so that per-sample byte offsets, the cumulative session sample count,
	// and the current domain start remain consistent across owners. Reset on rollover
	// via the domain.Writer's OnRollover hook.
	tracker *offsetTracker
	// sessionStart is the start of the resource's first domain since it last ended one.
	// Commits that resolve their end from the session sample count anchor on it.
	sessionStart telem.TimeStamp
	// alignments reserves the alignment domain of each domain after the first.
	alignments *alignment.Counter
}

var _ control.Resource = &controlledWriter{}

// ChannelKey implements controller.Resource.
func (w *controlledWriter) ChannelKey() channel.Key { return w.channelKey }

func (w *controlledWriter) loadAlignment() telem.Alignment {
	return telem.Alignment(w.alignment.Load())
}

func (w *controlledWriter) storeAlignment(a telem.Alignment) {
	w.alignment.Store(uint64(a))
}

func (w *controlledWriter) setStart(start telem.TimeStamp) error {
	if err := w.SetStart(start); err != nil {
		return err
	}
	w.sessionStart = start
	w.tracker.currentStart = start
	return nil
}

func (w *controlledWriter) endDomain(ctx context.Context) error {
	if w.Len() == 0 {
		return nil
	}
	a, err := w.alignments.Next()
	if err != nil {
		return err
	}
	if err = w.EndDomain(ctx); err != nil {
		return err
	}
	w.tracker.reset()
	w.storeAlignment(a)
	return nil
}

type Writer struct {
	// control stores the gate held by the writer in the controller of the unaryDB.
	control *control.Gate[*controlledWriter]
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
	// highWaterMark is a hot-path optimization when writing to an index channel. We can
	// avoid unnecessary index lookups by keeping track of the highest timestamp
	// written. Only valid when Channel.IsIndex is true.
	highWaterMark telem.TimeStamp
	// closed stores whether the writer is closed. Operations like Write and Commit do
	// not succeed on closed writers.
	closed bool
}

func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (
	w *Writer,
	transfer control.Transfer,
	err error,
) {
	if db.closed.Load() {
		return nil, transfer, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(defaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, err
	}
	start := cfg.Start
	if start.IsZero() {
		start = telem.Now()
	}
	w = &Writer{
		cfg:         cfg,
		Channel:     db.cfg.Channel,
		idx:         db.index(),
		wrapError:   db.wrapError,
		domainStart: start,
	}
	if w.control, transfer, err = db.controller.OpenGate(
		control.GateConfig[*controlledWriter]{
			Bounded:               cfg.Bounded,
			ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorizedOpen,
			TimeRange:             cfg.controlTimeRange(start),
			Authority:             cfg.Authority,
			Subject:               cfg.Subject,
			OpenResource: func() (*controlledWriter, error) {
				a, err := db.leadingAlignment.Next()
				if err != nil {
					return nil, err
				}
				cw := &controlledWriter{
					channelKey:   db.cfg.Channel.Key,
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
		return nil, transfer, w.wrapError(err)
	}
	return w, w.visible(transfer), w.wrapError(err)
}

// Write writes series to db in a domain that starts at start.
func Write(
	ctx context.Context,
	db *DB,
	start telem.TimeStamp,
	series telem.Series,
) (err error) {
	w, _, err := db.OpenWriter(ctx, WriterConfig{
		Start:     start,
		Authority: xcontrol.AuthorityAbsolute,
		Subject:   xcontrol.Subject{Key: uuid.New().String()},
	})
	if err != nil {
		return db.wrapError(err)
	}
	w.SetDomainStart(start)
	defer func() {
		_, errClose := w.Close()
		err = db.wrapError(errors.Combine(err, errClose))
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
	return w.write(series, 0, true)
}

// WriteAt validates and writes the given series, stamping it with a, an alignment
// resolved in the sample space of this channel's index. Returns the alignment of the
// first sample written.
func (w *Writer) WriteAt(
	series telem.Series,
	a telem.Alignment,
) (telem.Alignment, error) {
	return w.write(series, a, false)
}

func (w *Writer) write(
	series telem.Series,
	a telem.Alignment,
	derive bool,
) (telem.Alignment, error) {
	if w.closed {
		return 0, w.wrapError(ErrWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, w.wrapError(err)
	}
	if *w.cfg.Persist && dw.Len() == 0 {
		// Extend first, so that no delete can reach the domain setStart checks.
		if err = w.control.Extend(w.domainStart); err != nil {
			return 0, w.wrapError(err)
		}
		w.control.Advance(w.domainStart)
		if err = dw.setStart(w.domainStart); err != nil {
			return 0, w.wrapError(err)
		}
	}
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	if derive {
		// The tracker is only fed when persisting, so in stream-only mode the published
		// cursor is the sole record of where the next write lands.
		if *w.cfg.Persist {
			a = telem.NewAlignment(
				dw.loadAlignment().DomainIndex(),
				uint32(dw.tracker.count(dw.Writer)),
			)
		} else {
			a = dw.loadAlignment()
		}
	}
	// Publish where the next write lands before this one is on disk, so a channel
	// resolving its alignment against this one reserves the slot after this write
	// instead of colliding with it.
	dw.storeAlignment(a.AddSamples(uint32(series.Len())))
	if *w.cfg.Persist {
		baseOffset := uint32(dw.tracker.domainBytes)
		dw.tracker.record(series.Data, baseOffset)
		_, err = dw.Write(series.Data)
	}
	return a, w.wrapError(err)
}

// SetDomainStart sets where the writer's domain starts, in place of Start. It applies
// at the next persisted write, and only while the domain holds no data. The writer's
// control region moves to the start, unless the writer is bounded and the start comes
// after the region.
func (w *Writer) SetDomainStart(start telem.TimeStamp) { w.domainStart = start }

// Authorize returns control.ErrUnauthorized if the writer does not hold control.
func (w *Writer) Authorize() error {
	_, err := w.control.Authorize()
	return w.wrapError(err)
}

// EndDomain ends the domain the writer holds, discarding any uncommitted samples. The
// next persisted write starts a new domain in a new alignment domain, at the start
// given to SetDomainStart. EndDomain does nothing when the domain holds no data.
func (w *Writer) EndDomain(ctx context.Context) error {
	if w.closed {
		return w.wrapError(ErrWriterClosed)
	}
	dw, err := w.control.Authorize()
	if err != nil {
		return w.wrapError(err)
	}
	return w.wrapError(dw.endDomain(ctx))
}

func (w *Writer) SetAuthority(a xcontrol.Authority) control.Transfer {
	return w.visible(w.control.SetAuthority(a))
}

// visible hides a transfer of a bounded writer, which says nothing about who controls
// the channel.
func (w *Writer) visible(t control.Transfer) control.Transfer {
	if *w.cfg.Bounded {
		return control.Transfer{}
	}
	return t
}

func (w *Writer) updateHwm(series telem.Series) {
	if series.Len() != 0 {
		w.highWaterMark = series.ValueAt[telem.TimeStamp](-1)
	}
}

// Commit commits the written series to the database.
func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.closed {
		return 0, w.wrapError(ErrWriterClosed)
	}

	if w.Channel.IsIndex {
		ts, err := w.commitWithEnd(ctx, w.highWaterMark+1)
		if err != nil {
			return 0, w.wrapError(err)
		}
		return ts, nil
	}
	ts, err := w.commitWithEnd(ctx, telem.TimeStamp(0))
	if err != nil {
		return 0, w.wrapError(err)
	}
	return ts, nil
}

func (w *Writer) CommitWithEnd(ctx context.Context, end telem.TimeStamp) (err error) {
	if w.closed {
		return w.wrapError(ErrWriterClosed)
	}
	_, err = w.commitWithEnd(ctx, end)
	return w.wrapError(err)
}

func (w *Writer) commitWithEnd(
	ctx context.Context,
	end telem.TimeStamp,
) (telem.TimeStamp, error) {
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, err
	}

	if end.IsZero() {
		// Subtract 1 because we want the timestamp of the last written sample.
		// Anchor on the session start with the cumulative session sample count, since
		// after a rollover the per-domain start is "previous commit end + 1ns" which
		// is not a sample boundary in the index.
		approx, err := w.idx.Stamp(
			ctx,
			dw.sessionStart,
			dw.tracker.count(dw.Writer)-1,
			index.MustBeContinuous,
		)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, index.NewDiscontinuousStampError(dw.sessionStart)
		}
		// Add 1 to the end timestamp because the end timestamp is exclusive.
		end = approx.Lower + 1
	}

	if *w.cfg.Bounded {
		if err = w.control.Reach(end); err != nil {
			return 0, err
		}
	}
	if err := dw.Commit(ctx, end); err != nil {
		return 0, err
	}
	dw.tracker.publish(end)
	return end, nil
}

func (w *Writer) Close() (control.Transfer, error) {
	if w.closed {
		return control.Transfer{}, nil
	}
	w.closed = true
	dw, t := w.control.Release()
	if t.IsRelease() {
		return w.visible(t), w.wrapError(dw.Close())
	}
	return w.visible(t), nil
}
