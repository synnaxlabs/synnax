// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// WriterConfig is the configuration for opening a writer.
type WriterConfig struct {
	// Start marks the starting bound of the domain. This starting bound must not
	// overlap with any existing domains within the DB.
	// [REQUIRED]
	Start telem.TimeStamp
	// End is an optional parameter that marks the ending bound of the domain. Defining
	// this parameter will allow the writer to write data to the domain without needing
	// to validate each call to Commit. If this parameter is not defined, Commit must be
	// called with a strictly increasing timestamp.
	// [OPTIONAL]
	End telem.TimeStamp
	// EnableAutoCommit determines whether the writer will automatically commit after
	// each write. If EnableAutoCommit is true, then the writer will commit after each
	// write, and will flush that commit to index on FS after the specified
	// AutoIndexPersistInterval.
	// [OPTIONAL] - Defaults to false.
	EnableAutoCommit *bool
	// AutoIndexPersistInterval is the frequency at which the changes to index are
	// persisted to the disk. If AutoIndexPersistInterval <=0, then the writer persists
	// changes to disk after every commit. Setting an AutoIndexPersistInterval is
	// invalid if EnableAutoCommit is off.
	// [OPTIONAL] Defaults to 1s
	AutoIndexPersistInterval telem.TimeSpan
}

var (
	errWriterClosed     = core.NewErrResourceClosed("domain.writer")
	DefaultWriterConfig = WriterConfig{EnableAutoCommit: config.False(), AutoIndexPersistInterval: 1 * telem.Second}
)

const AlwaysIndexPersistOnAutoCommit telem.TimeSpan = -1

// Domain returns the Domain occupied by the theoretical domain formed by the
// configuration. If End is not set, assumes the Domain has a zero span starting at
// Start.
func (w WriterConfig) Domain() telem.TimeRange {
	if w.End.IsZero() {
		return w.Start.SpanRange(0)
	}
	return telem.TimeRange{Start: w.Start, End: w.End}
}

func (w WriterConfig) Validate() error {
	v := validate.New("domain.WriterConfig")
	v.Ternary("end", w.End.Before(w.Start), "end timestamp must be after or equal to start timestamp")
	return nil
}

func (w WriterConfig) Override(other WriterConfig) WriterConfig {
	w.Start = override.Zero(w.Start, other.Start)
	w.End = override.Zero(w.End, other.End)
	w.EnableAutoCommit = override.Nil(w.EnableAutoCommit, other.EnableAutoCommit)
	w.AutoIndexPersistInterval = override.Zero(w.AutoIndexPersistInterval, other.AutoIndexPersistInterval)
	return w
}

// Write writes the given data to the DB new telemetry domain occupying the provided time
// range. If the time domain overlaps with any other domains in the DB, Write will return
// an error.
func Write(ctx context.Context, db *DB, tr telem.TimeRange, data []byte) (err error) {
	w, err := db.OpenWriter(ctx, WriterConfig{Start: tr.Start, End: tr.End})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, w.Close())
	}()
	if _, err = w.Write(data); err != nil {
		return err
	}
	return w.Commit(ctx, tr.End)
}

// Writer is used to write a telemetry domain to the DB. A Writer is opened using
// DB.OpenWriter and a provided WriterConfig, which defines the starting bound of the
// domain. If no other domain overlaps with the starting bound, the caller can write
// telemetry data the Writer using an io.TypedWriter interface.
//
// Once the caller is done writing telemetry data, they must call Commit and provide the
// ending bound of the domain. If the ending bound of the domain overlaps with any other
// domains within the DB, Commit will return an error, and the domain will not be
// committed. If the caller explicitly knows the ending bound of the domain, they can
// set the WriterConfig.End parameter to pre-validate the ending bound of the domain. If
// the WriterConfig.End parameter is set, Commit will ignore the provided timestamp and
// use the WriterConfig.End parameter instead.
//
// A Writer is not safe for concurrent use, but it is safe to have multiple writer and
// iterators open concurrently over the same DB.
type Writer struct {
	alamos.Instrumentation
	WriterConfig
	// prevCommit is the timestamp for the previous Commit call made to the database.
	prevCommit telem.TimeStamp
	// idx is the underlying index for the database that stores locations of domains in FS.
	idx *index
	// fileKey represents the key of the file written to by the writer. One can convert
	// it to a filename via the fileKeyToName function.
	fileKey uint16
	// fc is the file controller for the writer's FS.
	fc *fileController
	// fileSize is the writer's file's size
	fileSize telem.Size
	// len is the number of bytes written by all internal writers of the domain writer.
	len int64
	// internal is a TrackedWriteCloser used to write telemetry to FS.
	internal xio.TrackedWriteCloser
	// presetEnd denotes whether the writer has a preset end as part of its
	// WriterConfig. If it does, then commits to the writer will use that end as the end
	// of the domain.
	presetEnd bool
	// lastIndexPersist stores the timestamp of the last time changes to index were
	// flushed to disk.
	lastIndexPersist telem.TimeStamp
	// closed denotes whether the writer is closed. A closed writer returns an error
	// when attempts to Write or Commit with it are made.
	closed bool
	// onClose is called when the writer is closed.
	onClose func()
}

// OpenWriter opens a new Writer using the given configuration. If err is nil, then the
// writer must be closed.
func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}

	cfg, err := config.New(DefaultWriterConfig, cfg)
	if err != nil {
		return nil, err
	}
	if db.idx.overlap(cfg.Domain()) {
		return nil, errors.Wrap(
			NewErrRangeWriteConflict(cfg.Domain(), db.idx.timeRange()),
			"cannot open writer because there is already data in the writer's time range",
		)
	}
	key, size, internal, err := db.fc.acquireWriter(ctx)
	if err != nil {
		return nil, err
	}
	db.resourceCount.Add(1)
	w := &Writer{
		WriterConfig:     cfg,
		Instrumentation:  db.cfg.Child("writer"),
		fileKey:          key,
		fc:               db.fc,
		fileSize:         telem.Size(size),
		internal:         internal,
		idx:              db.idx,
		presetEnd:        !cfg.End.IsZero(),
		lastIndexPersist: telem.Now(),
		onClose: func() {
			db.resourceCount.Add(-1)
		},
	}

	// If we don't have a preset end, we defer to using the start of the next domain as
	// the end of the new domain.
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
func (w *Writer) Len() int64 { return w.len }

// Writer writes binary telemetry to the domain. Write is not safe to call concurrently
// with any other Writer methods. The contents of p are safe to modify after Write
// returns.
func (w *Writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, errWriterClosed
	}
	n, err := w.internal.Write(p)
	w.fileSize += telem.Size(n)
	w.len += int64(n)
	return n, err
}

// Commit commits the domain to the DB, making it available for reading by other
// processes. If the WriterConfig.End parameter was set, Commit will ignore the provided
// timestamp and use the WriterConfig.End parameter instead. If the WriterConfig.End
// parameter was not set, Commit will validate that the provided timestamp is strictly
// greater than the previous commit. If the provided timestamp is not strictly greater
// than the previous commit, Commit will return an error. If the domain formed by the
// WriterConfig.Start and the provided timestamp overlaps with any other domains within
// the DB, Commit will return an ErrWriteConflict. If
// WriterCommit.AutoIndexPersistInterval is greater than 0, then the changes committed
// would only be persisted to disk after the set interval.
func (w *Writer) Commit(ctx context.Context, end telem.TimeStamp) error {
	var (
		now = telem.Now()
		// the only time we do not shouldPersist is when EnableAutoCommit and the interval is
		// not met yet.
		shouldPersist = !*w.EnableAutoCommit || w.lastIndexPersist.Span(now) >= w.AutoIndexPersistInterval
	)

	if *w.EnableAutoCommit && w.AutoIndexPersistInterval > 0 && shouldPersist {
		w.lastIndexPersist = now
	}

	return w.commit(ctx, end, shouldPersist)
}

func (w *Writer) commit(ctx context.Context, end telem.TimeStamp, shouldPersist bool) error {
	ctx, span := w.T.Prod(ctx, "commit")
	defer span.End()

	if w.closed {
		return span.Error(errWriterClosed)
	}
	if w.presetEnd && end.After(w.End) {
		return span.Error(errors.Newf(
			"commit timestamp %v cannot be greater than preset end timestamp %v: exceeded by a time span of %v",
			end,
			w.End,
			w.End.Span(end),
		))
	}

	length := w.internal.Len()
	if length == 0 {
		return nil
	}

	commitEnd, switchingFile := w.resolveCommitEnd(end)
	if err := w.validateCommitRange(commitEnd, switchingFile); err != nil {
		return span.Error(err)
	}

	ptr := pointer{
		TimeRange: telem.TimeRange{Start: w.Start, End: commitEnd},
		offset:    uint32(w.internal.Offset()),
		size:      uint32(length),
		fileKey:   w.fileKey,
	}
	f := lo.Ternary(w.prevCommit.IsZero(), w.idx.insert, w.idx.update)

	err := span.Error(f(ctx, ptr, shouldPersist))
	if err != nil {
		return span.Error(err)
	}

	if switchingFile {
		err = w.internal.Close()
		if err != nil {
			return span.Error(err)
		}

		newFileKey, newFileSize, newInternalWriter, err := w.fc.acquireWriter(ctx)
		if err != nil {
			return span.Error(err)
		}

		w.fileKey = newFileKey
		w.internal = newInternalWriter
		w.fileSize = telem.Size(newFileSize)
		w.Start = commitEnd
		w.prevCommit = 0
	} else {
		w.prevCommit = commitEnd
	}

	return nil
}

// resolveCommitEnd returns whether a file change is needed, the resolved commit end,
// and any errors.
func (w *Writer) resolveCommitEnd(end telem.TimeStamp) (telem.TimeStamp, bool) {
	// fc.ConfigValues.FileSize is the nominal file size to not exceed, in reality, this value
	// is set to 0.8 * the actual file size cap. Therefore, we only need to switch files
	// once we write to over 1.25 * that nominal value.
	if w.fileSize >= w.fc.realFileSizeCap() {
		return end, true
	}

	return lo.Ternary(w.presetEnd, w.End, end), false
}

// Close closes the writer, releasing any resources it may have been holding. Any
// uncommitted data will be discarded. Any committed, but unpersisted data will be
// persisted. Close is idempotent and is also not safe to call concurrently with any
// other writer methods.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	defer w.onClose()
	w.closed = true
	if err := w.internal.Close(); err != nil {
		return err
	}
	if *w.EnableAutoCommit && w.AutoIndexPersistInterval > 0 {
		w.idx.mu.RLock()
		persistPointers := w.idx.indexPersist.prepare(w.idx.persistHead)
		w.idx.mu.RUnlock()
		return persistPointers()
	}
	return nil
}

func (w *Writer) validateCommitRange(end telem.TimeStamp, switchingFile bool) error {
	if !w.prevCommit.IsZero() && !switchingFile && end.Before(w.prevCommit) {
		return errors.Wrapf(validate.Error, "commit timestamp %s must not be less than the previous commit timestamp %s: it is less by a time span of %v", end, w.prevCommit, end.Span(w.prevCommit))
	}
	if !w.Start.Before(end) {
		return errors.Wrapf(validate.Error, "commit timestamp %s must be strictly greater than the starting timestamp %s: it is less by a time span of %v", end, w.Start, end.Span(w.Start))
	}
	return nil
}
