// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type WriterConfig struct {
	// Start marks the starting bound of the writer.
	// [REQUIRED]
	Start telem.TimeStamp
	// End is an optional parameter that marks the ending bound of the domain. Defining this
	// parameter will allow the writer to write data to the domain without needing to
	// validate each call to Commit. If this parameter is not defined, Commit must
	// be called with a strictly increasing timestamp.
	// [OPTIONAL]
	End telem.TimeStamp
	// Subject is the control subject held by the writer.
	// [REQUIRED]
	Subject control.Subject
	// Authority is the control authority held by the writer: higher authority entities have
	// priority access to the region.
	// [OPTIONAL]
	Authority control.Authority
	// Persist denotes whether the writer writes its data to FS. If Persist is off, no data
	// is written.
	// [OPTIONAL] - Defaults to true
	Persist *bool
	// EnableAutoCommit denotes whether each write is committed.
	// [OPTIONAL] - Defaults to False
	EnableAutoCommit *bool
	// AutoIndexPersistInterval is the frequency at which the changes to index are persisted to the
	// disk.
	// [OPTIONAL] - Defaults to 1s.
	AutoIndexPersistInterval telem.TimeSpan
	// ErrOnUnauthorized controls whether the writer will return an error on open when
	// attempting to write to a channel that is does not have authority over.
	// [OPTIONAL] - Defaults to false
	ErrOnUnauthorized    *bool
	AlignmentDomainIndex uint32
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{
		Persist:                  config.True(),
		EnableAutoCommit:         config.False(),
		AutoIndexPersistInterval: 1 * telem.Second,
		ErrOnUnauthorized:        config.False(),
	}
	errWriterClosed = core.EntityClosed("unary.writer")
)

const AlwaysIndexPersistOnAutoCommit telem.TimeSpan = -1

// Validate implements config.Config.
func (c WriterConfig) Validate() error {
	v := validate.New("unary.WriterConfig")
	validate.NotEmptyString(v, "Subject.Key", c.Subject.Key)
	validate.NotNil(v, "ErrOnUnauthorized", c.ErrOnUnauthorized)
	v.Ternary("end", !c.End.IsZero() && c.End.Before(c.Start), "end timestamp must be after or equal to start timestamp")
	return v.Error()
}

// Override implements config.Config.
func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.End = override.Zero(c.End, other.End)
	c.Subject = override.If(c.Subject, other.Subject, other.Subject.Key != "")
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Persist = override.Nil(c.Persist, other.Persist)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(c.AutoIndexPersistInterval, other.AutoIndexPersistInterval)
	c.ErrOnUnauthorized = override.Nil(c.ErrOnUnauthorized, other.ErrOnUnauthorized)
	c.AlignmentDomainIndex = override.Numeric(c.AlignmentDomainIndex, other.AlignmentDomainIndex)
	return c
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{Start: c.Start, End: c.End, EnableAutoCommit: c.EnableAutoCommit, AutoIndexPersistInterval: c.AutoIndexPersistInterval}
}

func (c WriterConfig) controlTimeRange() telem.TimeRange {
	// The automatic controlTimeRange is until the end of time, but we are not sure if
	// we should use this or the start of next domain.
	return c.Start.Range(lo.Ternary(c.End.IsZero(), telem.TimeStampMax, c.End))
}

// controlledWriter is used for exchanging control between multiple unary writers. When
// control is transferred, ownership of the domain writer is moved to the new unary
// writer. Additional state is included to ensure that write positions and channel. information
// are consistent.
type controlledWriter struct {
	*domain.Writer
	channelKey core.ChannelKey
	alignment  telem.AlignmentPair
}

var _ controller.Entity = controlledWriter{}

// ChannelKey implements controller.Entity.
func (w controlledWriter) ChannelKey() core.ChannelKey { return w.channelKey }

type Writer struct {
	cfg WriterConfig
	// Channel stores information about the channel this writer is writing to, including
	// but not limited to density and index.
	Channel core.Channel
	// control stores the gate held by the writer in the controller of the unaryDB.
	control *controller.Gate[*controlledWriter]
	// idx stores the index of the unaryDB (rate or domain).
	idx index.Index
	// hwm is a hot-path optimization when writing to an index channel. We can avoid
	// unnecessary index lookups by keeping track of the highest timestamp written.
	// Only valid when Channel.IsIndex is true.
	hwm telem.TimeStamp
	// lastCommitFileSwitch describes whether the last commit involved a file switch.
	// If it did, then it is necessary to resolve the timestamp for that commit this time.
	lastCommitFileSwitch bool
	// wrapError is a function that wraps any error originating from this writer to
	// provide context including the writer's channel key and name.
	wrapError func(error) error
	// closed stores whether the writer is closed. Operations like Write and Commit do not
	// succeed on closed writers.
	closed bool
}

func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (
	w *Writer,
	transfer controller.Transfer,
	err error,
) {
	if db.closed.Load() {
		return nil, transfer, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, err
	}
	w = &Writer{
		cfg:       cfg,
		Channel:   db.cfg.Channel,
		idx:       db.index(),
		wrapError: db.wrapError,
	}
	gateCfg := controller.GateConfig{
		TimeRange: cfg.controlTimeRange(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
	var g *controller.Gate[*controlledWriter]
	g, transfer, err = db.controller.OpenGateAndMaybeRegister(gateCfg, func() (*controlledWriter, error) {
		dw, err := db.domain.OpenWriter(ctx, cfg.domain())
		cw := &controlledWriter{
			Writer:     dw,
			channelKey: db.cfg.Channel.Key,
			alignment:  telem.NewAlignmentPair(cfg.AlignmentDomainIndex, 0),
		}
		if cfg.AlignmentDomainIndex == 0 {
			cw.alignment = telem.NewAlignmentPair(db.leadingAlignment.Add(1), 0)
		}
		return cw, err
	})
	if err != nil {
		return nil, transfer, w.wrapError(err)
	}
	if *cfg.ErrOnUnauthorized {
		if _, err = g.Authorize(); err != nil {
			g.Release()
			return nil, transfer, err
		}
	}
	w.control = g
	return w, transfer, w.wrapError(err)
}

func Write(
	ctx context.Context,
	db *DB,
	start telem.TimeStamp,
	series telem.Series,
) (err error) {
	w, _, err := db.OpenWriter(ctx, WriterConfig{
		Start:     start,
		Authority: control.Absolute,
		Subject:   control.Subject{Key: uuid.New().String()},
	})
	if err != nil {
		return db.wrapError(err)
	}
	defer func() {
		_, err_ := w.Close()
		err = db.wrapError(errors.Combine(err, err_))
	}()
	if _, err = w.Write(series); err != nil {
		return err
	}
	_, err = w.Commit(ctx)
	return err
}

func (w *Writer) len(dw *domain.Writer) int64 {
	return w.Channel.DataType.Density().SampleCount(telem.Size(dw.Len()))
}

// Write validates and writes the given array.
func (w *Writer) Write(series telem.Series) (telem.AlignmentPair, error) {
	if w.closed {
		return 0, w.wrapError(errWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, w.wrapError(err)
	}
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	if *w.cfg.Persist {
		dw.alignment = telem.NewAlignmentPair(dw.alignment.DomainIndex(), uint32(w.len(dw.Writer)))
		_, err = dw.Write(series.Data)
	} else {
		dw.alignment = dw.alignment.AddSamples(uint32(series.Len()))
	}
	return dw.alignment, w.wrapError(err)
}

func (w *Writer) DomainIndex() uint32 {
	return w.control.PeekEntity().alignment.DomainIndex()
}

func (w *Writer) SetAuthority(a control.Authority) controller.Transfer {
	return w.control.SetAuthority(a)
}

func (w *Writer) updateHwm(series telem.Series) {
	if series.Len() == 0 {
		return
	}
	w.hwm = telem.ValueAt[telem.TimeStamp](series, series.Len()-1)
}

// Commit commits the written series to the database.
func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.closed {
		return telem.TimeStampMax, w.wrapError(errWriterClosed)
	}

	if w.Channel.IsIndex {
		ts, err := w.commitWithEnd(ctx, w.hwm+1)
		return ts, w.wrapError(err)
	}
	ts, err := w.commitWithEnd(ctx, telem.TimeStamp(0))
	return ts, w.wrapError(err)
}

func (w *Writer) CommitWithEnd(ctx context.Context, end telem.TimeStamp) (err error) {
	if w.closed {
		return w.wrapError(errWriterClosed)
	}
	_, err = w.commitWithEnd(ctx, end)
	return w.wrapError(err)
}

func (w *Writer) commitWithEnd(ctx context.Context, end telem.TimeStamp) (telem.TimeStamp, error) {
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, err
	}

	if end.IsZero() {
		// We're using w.len - 1 here because we want the timestamp of the last
		// written frame.
		approx, err := w.idx.Stamp(ctx, w.cfg.Start, w.len(dw.Writer)-1, true)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, errors.Wrapf(
				validate.Error,
				"writer start %s cannot be resolved in the index channel %v",
				w.cfg.Start,
				w.idx.Info(),
			)
		}
		// Add 1 to the end timestamp because the end timestamp is exclusive.
		end = approx.Lower + 1
	}

	return end, dw.Commit(ctx, end)
}

func (w *Writer) Close() (controller.Transfer, error) {
	if w.closed {
		return controller.Transfer{}, nil
	}
	w.closed = true
	dw, t := w.control.Release()
	if t.IsRelease() {
		return t, w.wrapError(dw.Close())
	}
	return t, nil
}
