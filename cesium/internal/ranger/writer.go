// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/cockroachdb/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterConfig is the configuration for opening a writer.
type WriterConfig struct {
	// Start marks the starting bound of the range. This starting bound must not
	// overlap with any existing ranges within the DB.
	// [REQUIRED]
	Start telem.TimeStamp
	// End is an optional parameter that marks the ending bound of the range. Defining this
	// parameter will allow the writer to write data to the range without needing to
	// validate each call to Commit. If this parameter is not defined, Commit must
	// be called with a strictly increasing timestamp.
	// [OPTIONAL]
	End telem.TimeStamp
}

// Range returns the Range occupied by the theoretical range formed by the configuration.
// If End is not set, assumes the Range has a zero span starting at Start.
func (w WriterConfig) Range() telem.TimeRange {
	if w.End.IsZero() {
		return w.Start.SpanRange(0)
	}
	return telem.TimeRange{Start: w.Start, End: w.End}
}

// Write writes the given data to the DB new telemetry range occupying the provided time
// range. If the time range overlaps with any other ranges in the DB, Write will return
// an error.
func Write(db *DB, tr telem.TimeRange, data []byte) error {
	w, err := db.NewWriter(WriterConfig{Start: tr.Start, End: tr.End})
	if err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if err = w.Commit( /* ignored */ 0); err != nil {
		return err
	}
	return w.Close()
}

// Writer is used to write a telemetry range to the DB. A Writer is opened using DB.NewWriter
// and a provided WriterConfig, which defines the starting bound of the range. If no
// other range overlaps with the starting bound, the caller can write telemetry data the
// Writer using an io.Writer interface.
//
// Once the caller is done writing telemetry data, they must call Commit and provide the
// ending bound of the range. If the ending bound of the range overlaps with any other
// ranges within the DB, Commit will return an error, and the range will not be committed.
// If the caller explicitly knows the ending bound of the range, they can set the WriterConfig.End
// parameter to pre-validate the ending bound of the range. If the WriterConfig.End parameter
// is set, Commit will ignore the provided timestamp and use the WriterConfig.End parameter
// instead.
//
// A Writer is not safe for concurrent use, but it is safe to have multiple writer and
// iterators open concurrently over the same DB.
type Writer struct {
	cfg        WriterConfig
	prevCommit telem.TimeStamp
	idx        *index
	fileKey    uint16
	internal   xio.OffsetWriteCloser
}

// Len returns the number of bytes written to the range.
func (w *Writer) Len() int64 { return w.internal.Len() }

// Writer writes binary telemetry to the range. Write is not safe to call concurrently
// with any other Writer methods. The contents of p are safe to modify after Write
// returns.
func (w *Writer) Write(p []byte) (n int, err error) { return w.internal.Write(p) }

// Commit commits the range to the DB, making it available for reading by other processes.
// If the WriterConfig.End parameter was set, Commit will ignore the provided timestamp
// and use the WriterConfig.End parameter instead. If the WriterConfig.End parameter was
// not set, Commit will validate that the provided timestamp is strictly greater than the
// previous commit. If the provided timestamp is not strictly greater than the previous
// commit, Commit will return an error. If the range formed by the WriterConfig.Start
// and the provided timestamp overlaps with any other ranges within the DB, Commit will
// return an error.
func (w *Writer) Commit(end telem.TimeStamp) error {
	if !w.cfg.End.IsZero() {
		end = w.cfg.End
	}
	if err := w.validateCommitRange(end); err != nil {
		return err
	}
	ptr := pointer{
		TimeRange: telem.TimeRange{Start: w.cfg.Start, End: end},
		offset:    uint32(w.internal.Offset()),
		length:    uint32(w.internal.Len()),
		fileKey:   w.fileKey,
	}
	if w.prevCommit.IsZero() {
		w.prevCommit = end
		return w.idx.insert(ptr)
	}
	return w.idx.update(ptr)
}

// Close closes the writer, releasing any resources it may have been holding. Any
// uncommitted data will be discarded. Close is not idempotent, and is also not
// safe to call concurrently with any other writer methods.
func (w *Writer) Close() error { return w.internal.Close() }

func (w *Writer) validateCommitRange(end telem.TimeStamp) error {
	if !w.prevCommit.IsZero() && end.Before(w.prevCommit) {
		return errors.Wrap(validate.Error, "commit timestamp must be strictly greater than the previous commit")
	}
	if !w.cfg.Start.Before(end) {
		return errors.Wrap(validate.Error, "commit timestamp must be strictly greater than the starting timestamp")
	}
	return nil
}
