// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

type WriterConfig struct {
	Start     telem.TimeStamp
	End       telem.TimeStamp
	Subject   control.Subject
	Authority control.Authority
	Persist   *bool
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{
		Persist: config.True(),
	}
	WriterClosedError = core.EntityClosed("unary.writer")
)

func (c WriterConfig) Validate() error {
	return nil
}

func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.End = override.Zero(c.End, other.End)
	c.Subject = override.If(c.Subject, other.Subject, other.Subject.Key != "")
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Persist = override.Nil(c.Persist, other.Persist)
	return c
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{Start: c.Start, End: c.End}
}

func (c WriterConfig) controlTimeRange() telem.TimeRange {
	// The automatic controlTimeRange is until the end of time, but we are not sure if
	// we should use this or the start of next domain.
	return c.Start.Range(lo.Ternary(c.End.IsZero(), telem.TimeStampMax, c.End))
}

type Writer struct {
	WriterConfig
	Channel          core.Channel
	decrementCounter func()
	control          *controller.Gate[controlledWriter]
	idx              index.Index
	// hwm is a hot-path optimization when writing to an index channel. We can avoid
	// unnecessary index lookups by keeping track of the highest timestamp written.
	// Only valid when Channel.IsIndex is true.
	hwm    telem.TimeStamp
	pos    int
	closed bool
}

func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (w *Writer, transfer controller.Transfer, err error) {
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, err
	}
	w = &Writer{WriterConfig: cfg, Channel: db.Channel, idx: db.index(), decrementCounter: func() { db.mu.Add(-1) }}
	gateCfg := controller.GateConfig{
		TimeRange: cfg.controlTimeRange(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
	var g *controller.Gate[controlledWriter]
	g, transfer, err = db.Controller.OpenGateAndMaybeRegister(gateCfg, func() (controlledWriter, error) {
		dw, err := db.Domain.NewWriter(ctx, cfg.domain())

		return controlledWriter{
			Writer:     dw,
			channelKey: db.Channel.Key,
		}, err
	})
	if err != nil {
		return nil, transfer, err
	}

	w.control = g
	db.mu.Add(1)
	return w, transfer, err
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
		return err
	}
	defer func() {
		_, err_ := w.Close()
		err = errors.CombineErrors(err, err_)
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
func (w *Writer) Write(series telem.Series) (a telem.Alignment, err error) {
	if w.closed {
		return 0, WriterClosedError
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, err
	}
	// ok signifies whether w is allowed to write.
	dw, ok := w.control.Authorize()
	if !ok {
		return 0, controller.Unauthorized(w.control.Subject.Name, w.Channel.Key)
	}
	a = telem.Alignment(w.len(dw.Writer))
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	if *w.Persist {
		_, err = dw.Write(series.Data)
	}
	return
}

func (w *Writer) SetPersist(persist bool) { w.Persist = config.Bool(persist) }

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
		return telem.TimeStampMax, WriterClosedError
	}
	if w.Channel.IsIndex {
		return w.commitWithEnd(ctx, w.hwm+1)
	}
	return w.commitWithEnd(ctx, telem.TimeStamp(0))
}

func (w *Writer) CommitWithEnd(ctx context.Context, end telem.TimeStamp) (err error) {
	if w.closed {
		return core.EntityClosed(("unary.writer"))
	}
	_, err = w.commitWithEnd(ctx, end)
	return err
}

func (w *Writer) commitWithEnd(ctx context.Context, end telem.TimeStamp) (telem.TimeStamp, error) {
	dw, ok := w.control.Authorize()
	if !ok {
		return 0, controller.Unauthorized(w.control.Subject.String(), w.Channel.Key)
	}
	if end.IsZero() {
		// we're using w.len - 1 here because we want the timestamp of the last
		// written frame.
		approx, err := w.idx.Stamp(ctx, w.Start, w.len(dw.Writer)-1, true)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, errors.New("could not get exact timestamp")
		}
		// Add 1 to the end timestamp because the end timestamp is exclusive.
		end = approx.Lower + 1
	}
	err := dw.Commit(ctx, end)
	return end, err
}

func (w *Writer) Close() (controller.Transfer, error) {
	if w.closed {
		return controller.Transfer{}, nil
	}

	w.closed = true
	dw, t := w.control.Release()
	w.decrementCounter()
	if t.IsRelease() {
		return t, dw.Close()
	}

	return t, nil
}
