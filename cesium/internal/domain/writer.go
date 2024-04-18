// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterConfig is the configuration for opening a writer.
type WriterConfig struct {
	// Start marks the starting bound of the domain. This starting bound must not
	// overlap with any existing domains within the DB.
	// [REQUIRED]
	Start telem.TimeStamp
	// End is an optional parameter that marks the ending bound of the domain. Defining this
	// parameter will allow the writer to write data to the domain without needing to
	// validate each call to Commit. If this parameter is not defined, Commit must
	// be called with a strictly increasing timestamp.
	// [OPTIONAL]
	End telem.TimeStamp
}

var WriterClosedError = core.EntityClosed("domain.writer")

// Domain returns the Domain occupied by the theoretical domain formed by the configuration.
// If End is not set, assumes the Domain has a zero span starting at Start.
func (w WriterConfig) Domain() telem.TimeRange {
	if w.End.IsZero() {
		return w.Start.SpanRange(0)
	}
	return telem.TimeRange{Start: w.Start, End: w.End}
}

// Write writes the given data to the DB new telemetry domain occupying the provided time
// range. If the time domain overlaps with any other domains in the DB, Write will return
// an error.
func Write(ctx context.Context, db *DB, tr telem.TimeRange, data []byte) error {
	w, err := db.NewWriter(ctx, WriterConfig{Start: tr.Start, End: tr.End})
	if err != nil {
		return err
	}
	if _, err = w.Write(data); err != nil {
		return err
	}
	if err = w.Commit(ctx /* ignored */, 0); err != nil {
		return err
	}
	return w.Close()
}

// Writer is used to write a telemetry domain to the DB. A Writer is opened using DB.NewWriter
// and a provided WriterConfig, which defines the starting bound of the domain. If no
// other domain overlaps with the starting bound, the caller can write telemetry data the
// Writer using an io.TypedWriter interface.
//
// Once the caller is done writing telemetry data, they must call Commit and provide the
// ending bound of the domain. If the ending bound of the domain overlaps with any other
// domains within the DB, Commit will return an error, and the domain will not be committed.
// If the caller explicitly knows the ending bound of the domain, they can set the WriterConfig.End
// parameter to pre-validate the ending bound of the domain. If the WriterConfig.End parameter
// is set, Commit will ignore the provided timestamp and use the WriterConfig.End parameter
// instead.
//
// A Writer is not safe for concurrent use, but it is safe to have multiple writer and
// iterators open concurrently over the same DB.
type Writer struct {
	alamos.Instrumentation
	WriterConfig
	prevCommit telem.TimeStamp
	idx        *index
	fileKey    uint16
	internal   xio.TrackedWriteCloser
	presetEnd  bool
	closed     bool
}

// NewWriter opens a new Writer using the given configuration.
func (db *DB) NewWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	key, internal, err := db.files.acquireWriter(ctx)
	if err != nil {
		return nil, err
	}
	if db.idx.overlap(cfg.Domain()) {
		return nil, ErrDomainOverlap
	}
	w := &Writer{
		WriterConfig:    cfg,
		Instrumentation: db.Instrumentation.Child("writer"),
		fileKey:         key,
		internal:        internal,
		idx:             db.idx,
		presetEnd:       !cfg.End.IsZero(),
	}

	// If we don't have a preset end, we defer to using the start of the next domain
	// as the end of the new domain.
	if !w.presetEnd {
		ptr, ok := w.idx.getGE(ctx, cfg.Start)
		if !ok {
			w.End = telem.TimeStampMax
		} else {
			w.End = ptr.Start
		}
	}
	return w, nil
}

// Len returns the number of bytes written to the domain.
func (w *Writer) Len() int64 { return w.internal.Len() }

// Writer writes binary telemetry to the domain. Write is not safe to call concurrently
// with any other Writer methods. The contents of p are safe to modify after Write
// returns.
func (w *Writer) Write(p []byte) (n int, err error) {
	if w.closed {
		return 0, WriterClosedError
	}
	return w.internal.Write(p)
}

// Commit commits the domain to the DB, making it available for reading by other processes.
// If the WriterConfig.End parameter was set, Commit will ignore the provided timestamp
// and use the WriterConfig.End parameter instead. If the WriterConfig.End parameter was
// not set, Commit will validate that the provided timestamp is strictly greater than the
// previous commit. If the provided timestamp is not strictly greater than the previous
// commit, Commit will return an error. If the domain formed by the WriterConfig.Start
// and the provided timestamp overlaps with any other domains within the DB, Commit will
// return an error.
func (w *Writer) Commit(ctx context.Context, end telem.TimeStamp) error {
	ctx, span := w.T.Prod(ctx, "commit")
	if w.closed {
		return span.EndWith(WriterClosedError)
	}
	if w.presetEnd {
		end = w.End
	}
	if err := w.validateCommitRange(end); err != nil {
		return span.EndWith(err)
	}
	length := w.internal.Len()
	if length == 0 {
		return nil
	}
	ptr := pointer{
		TimeRange: telem.TimeRange{Start: w.Start, End: end},
		offset:    uint32(w.internal.Offset()),
		length:    uint32(length),
		fileKey:   w.fileKey,
	}
	f := lo.Ternary(w.prevCommit.IsZero(), w.idx.insert, w.idx.update)
	w.prevCommit = end
	return span.EndWith(f(ctx, ptr))
}

// Close closes the writer, releasing any resources it may have been holding. Any
// uncommitted data will be discarded. Close is not idempotent, and is also not
// safe to call concurrently with any other writer methods.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return w.internal.Close()
}

func (w *Writer) validateCommitRange(end telem.TimeStamp) error {
	if !w.prevCommit.IsZero() && end.Before(w.prevCommit) {
		return errors.Wrap(validate.Error, "commit timestamp must be strictly greater than the previous commit")
	}
	if !w.Start.Before(end) {
		return errors.Wrap(validate.Error, "commit timestamp must be strictly greater than the starting timestamp")
	}
	return nil
}
