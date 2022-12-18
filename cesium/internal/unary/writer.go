package unary

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	Channel    core.Channel
	internal   *ranger.Writer
	start      telem.TimeStamp
	idx        index.Index
	hwm        telem.TimeStamp
	numWritten int64
}

func Write(db *DB, start telem.TimeStamp, arr telem.Array) error {
	w, err := db.NewWriter(ranger.WriterConfig{Start: start})
	if err != nil {
		return err
	}
	if err = w.Write(arr); err != nil {
		return err
	}
	_, err = w.Commit()
	return err
}

// Write validates and writes the given array.
func (w *Writer) Write(arr telem.Array) error {
	if err := w.validate(arr); err != nil {
		return err
	}
	w.numWritten += arr.Len()
	if w.Channel.IsIndex {
		w.updateHwm(arr)
	}
	_, err := w.internal.Write(arr.Data)
	return err
}

func (w *Writer) updateHwm(arr telem.Array) {
	if arr.Len() == 0 {
		return
	}
	w.hwm = telem.ValueAt[telem.TimeStamp](arr, arr.Len()-1)
}

// Commit commits the written Array to the database.
func (w *Writer) Commit() (telem.TimeStamp, error) {
	if w.Channel.IsIndex {
		return w.commitWithEnd(w.hwm + 1)
	}
	return w.commitWithEnd(telem.TimeStamp(0))
}

func (w *Writer) CommitWithEnd(end telem.TimeStamp) (err error) {
	_, err = w.commitWithEnd(end)
	return err
}

func (w *Writer) commitWithEnd(end telem.TimeStamp) (telem.TimeStamp, error) {
	if end.IsZero() {
		// we're using w.numWritten - 1 here because we want the timestamp of the last
		// written frame.
		approx, err := w.idx.Stamp(w.start, w.numWritten-1, true)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, errors.New("could not get exact timestamp")
		}
		// Add 1 to the end timestamp because the end timestamp is exclusive.
		end = approx.Lower + 1
	}
	return end, w.internal.Commit(end)
}

func (w *Writer) Close() error { return w.internal.Close() }

func (w *Writer) validate(arr telem.Array) error {
	if arr.DataType != w.Channel.DataType {
		return errors.Wrapf(
			validate.Error,
			"invalid array data type for channel %s, expected %s, got %s",
			w.Channel.Key,
			w.Channel.DataType,
			arr.DataType,
		)
	}
	return nil
}
