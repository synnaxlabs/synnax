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
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type WriterConfig struct {
	Name           string
	Start          telem.TimeStamp
	End            telem.TimeStamp
	Authority      control.Authority
	ControlDigests confluence.Inlet[controller.Digest]
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{Start: c.Start, End: c.End}
}

type Writer struct {
	WriterConfig
	Channel core.Channel
	control *controller.Gate[*domain.Writer]
	idx     index.Index
	// hwm is a hot-path optimization when writing to an index channel. We can avoid
	// unnecessary index lookups by keeping track of the highest timestamp written.
	// Only valid when Channel.IsIndex is true.
	hwm telem.TimeStamp
	pos int
}

func Write(ctx context.Context, db *DB, start telem.TimeStamp, series telem.Series) (err error) {
	w, err := db.OpenWriter(ctx, WriterConfig{Start: start, Authority: control.Absolute})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.CombineErrors(err, w.Close())
	}()
	if _, err = w.Write(series); err != nil {
		return err
	}
	_, err = w.Commit(ctx)
	return err
}

func (w *Writer) numWritten(dw *domain.Writer) int64 {
	return w.Channel.DataType.Density().SampleCount(telem.Size(dw.Len()))
}

// Write validates and writes the given array.
func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if err := w.validate(series); err != nil {
		return 0, err
	}
	dw, ok := w.control.Authorize()
	if !ok {
		return 0, controller.Unauthorized(w.Channel.Key)
	}
	alignment := telem.Alignment(w.numWritten(dw))
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	_, err := dw.Write(series.Data)
	return alignment, err
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
		return 0, controller.Unauthorized(w.Channel.Key)
	}
	if end.IsZero() {
		// we're using w.numWritten - 1 here because we want the timestamp of the last
		// written frame.
		approx, err := w.idx.Stamp(ctx, w.Start, w.numWritten(dw)-1, true)
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

func (w *Writer) Close() error {
	dw, regionReleased := w.control.Release()
	if !regionReleased {
		return nil
	}
	return dw.Close()
}

func (w *Writer) validate(series telem.Series) error {
	if (series.DataType == telem.Int64T || series.DataType == telem.TimeStampT) && (w.Channel.DataType == telem.Int64T || w.Channel.DataType == telem.TimeStampT) {
		return nil
	}
	if series.DataType != w.Channel.DataType {
		return errors.Wrapf(
			validate.Error,
			"invalid array data type for channel %s, expected %s, got %s",
			w.Channel.Key,
			w.Channel.DataType,
			series.DataType,
		)
	}
	return nil
}
