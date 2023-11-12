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
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

type WriterConfig struct {
	Start     telem.TimeStamp
	End       telem.TimeStamp
	Subject   control.Subject
	Authority control.Authority
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{Start: c.Start, End: c.End}
}

func (c WriterConfig) controlTimeRange() telem.TimeRange {
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
	hwm telem.TimeStamp
	pos int
}

func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (w *Writer, transfer controller.Transfer, err error) {
	w = &Writer{WriterConfig: cfg, Channel: db.Channel, idx: db.index(), decrementCounter: func() { db.openIteratorWriters.Add(-1) }}
	gateCfg := controller.GateConfig{
		TimeRange: cfg.controlTimeRange(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
	var (
		g  *controller.Gate[controlledWriter]
		ok bool
	)
	g, transfer, ok, err = db.Controller.OpenGate(gateCfg)
	if err != nil {
		return nil, transfer, err
	}
	if !ok {
		dw, err := db.Domain.NewWriter(ctx, cfg.domain())
		if err != nil {
			return nil, transfer, err
		}
		gateCfg.TimeRange = cfg.controlTimeRange()
		g, transfer, err = db.Controller.RegisterAndOpenGate(gateCfg, controlledWriter{
			Writer:     dw,
			channelKey: db.Channel.Key,
		})
	}
	w.control = g
	db.openIteratorWriters.Add(1)
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
func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, err
	}
	dw, ok := w.control.Authorize()
	if !ok {
		return 0, controller.Unauthorized(w.control.Subject.Name, w.Channel.Key)
	}
	alignment := telem.Alignment(w.len(dw.Writer))
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	_, err := dw.Write(series.Data)
	return alignment, err
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
	if w.Channel.IsIndex {
		return w.commitWithEnd(ctx, w.hwm+1)
	}
	return w.commitWithEnd(ctx, telem.TimeStamp(0))
}

func (w *Writer) CommitWithEnd(ctx context.Context, end telem.TimeStamp) (err error) {
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
	w.decrementCounter()
	dw, t := w.control.Release()
	if t.IsRelease() {
		return t, dw.Close()
	}
	return t, nil
}
