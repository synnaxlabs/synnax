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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	Channel    core.Channel
	internal   *domain.Writer
	start      telem.TimeStamp
	idx        index.Index
	hwm        telem.TimeStamp
	numWritten int64
}

func Write(ctx context.Context, db *DB, start telem.TimeStamp, series telem.Series) error {
	w, err := db.OpenWriter(ctx, domain.WriterConfig{Start: start})
	if err != nil {
		return err
	}
	if _, err = w.Write(series); err != nil {
		return err
	}
	_, err = w.Commit(ctx)
	return err
}

// Write validates and writes the given array.
func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if err := w.validate(series); err != nil {
		return 0, err
	}
	alignment := telem.Alignment(w.numWritten)
	w.numWritten += series.Len()
	if w.Channel.IsIndex {
		w.updateHwm(series)
	}
	_, err := w.internal.Write(series.Data)
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
	if end.IsZero() {
		// we're using w.numWritten - 1 here because we want the timestamp of the last
		// written frame.
		approx, err := w.idx.Stamp(ctx, w.start, w.numWritten-1, true)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, errors.New("could not get exact timestamp")
		}
		// Add 1 to the end timestamp because the end timestamp is exclusive.
		end = approx.Lower + 1
	}
	return end, w.internal.Commit(ctx, end)
}

func (w *Writer) Close() error { return w.internal.Close() }

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
