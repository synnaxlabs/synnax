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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// IteratorConfig is the configuration for opening a new iterator.
type IteratorConfig struct {
	// Bounds represent the interval of time that the iterator will be able to access.
	// Any domains whose Bounds overlap with the iterator's Bounds will be accessible. A
	// zero span domain is valid, but is relatively useless.
	// [REQUIRED]
	Bounds telem.TimeRange
}

var errIteratorClosed = core.NewErrResourceClosed("domain.iterator")

// IterRange generates an IteratorConfig that iterates over the provided time domain.
func IterRange(tr telem.TimeRange) IteratorConfig { return IteratorConfig{Bounds: tr} }

// Iterator iterates over the telemetry domains of a DB in time order. Iterator does not
// read any of the underlying data of a domain, but instead provides a means to access
// it via calls to Iterator.OpenReader.
//
// Iterator is not safe for concurrent use, but it is safe to have multiple iterators
// over the same DB.
//
// It's important to note that an Iterator does NOT iterate over a snapshot of the DB,
// and is not isolated from any writes that may be committed during the iterators
// lifetime. This means that the position of an iterator may shift unexpectedly. There
// are plans to implement MVCC in the future, but until then you have been warned.
type Iterator struct {
	IteratorConfig
	alamos.Instrumentation
	// position stores the current position of the iterator in the idx. NOTE: At the
	// moment, this position may not hold a consistent reference to the same domain if
	// the idx is modified during iteration.
	position int
	// idx is the index that the iterator is iterating over.
	idx *index
	// currPtr stores the current domain that the iterator is pointing to. This value
	// is only valid if the iterator is valid.
	currPtr pointer
	// valid stores whether the iterator is currently valid.
	valid bool
	// readerFactory gets a new reader for the given domain pointer.
	readerFactory func(ctx context.Context, ptr pointer) (*Reader, error)
	// closed stores whether the iterator is still open
	closed bool
	// onClose is called when the iterator is closed.
	onClose func()
}

// OpenIterator opens a new invalidated Iterator using the given configuration. A
// seeking call is required before it can be used.
func (db *DB) OpenIterator(cfg IteratorConfig) *Iterator {
	db.resourceCount.Add(1)
	i := &Iterator{
		Instrumentation: db.cfg.Child("iterator"),
		idx:             db.idx,
		readerFactory:   db.newReader,
		onClose:         func() { db.resourceCount.Add(-1) },
	}
	i.SetBounds(cfg.Bounds)
	return i
}

// Read reads data for all domains that overlap with the two time ranges and accumulates
// it into a single buffer.
func Read(ctx context.Context, db *DB, tr telem.TimeRange) (b []byte, err error) {
	i := db.OpenIterator(IterRange(tr))
	defer func() {
		err = errors.Combine(err, i.Close())
	}()
	var data []byte
	for i.SeekFirst(ctx); i.Valid(); i.Next() {
		r, err := i.OpenReader(ctx)
		if err != nil {
			return nil, err
		}
		chunk := make([]byte, r.Size())
		if _, err = r.ReadAt(chunk, 0); err != nil {
			return nil, errors.Combine(err, r.Close())
		}
		data = append(data, chunk...)
		if err := r.Close(); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (i *Iterator) SetBounds(bounds telem.TimeRange) {
	i.Bounds = bounds
	i.valid = false
}

// SeekFirst seeks to the first domain in the iterator's bounds. If no such domain
// exists, SeekFirst returns false.
func (i *Iterator) SeekFirst(ctx context.Context) bool {
	if i.closed {
		return false
	}
	return i.SeekGE(ctx, i.Bounds.Start)
}

// SeekLast seeks to the last domain in the iterator's bounds. If no such domain exists,
// SeekLast returns false.
func (i *Iterator) SeekLast(ctx context.Context) bool {
	if i.closed {
		return false
	}
	return i.SeekLE(ctx, i.Bounds.End-1)
}

// SeekLE seeks to the domain whose TimeRange contain the provided timestamp. If no such domain
// exists, SeekLE seeks to the closest domain whose ending timestamp is less than the provided
// timestamp. If no such domain exists, SeekLE returns false.
func (i *Iterator) SeekLE(ctx context.Context, stamp telem.TimeStamp) bool {
	if i.closed {
		return false
	}
	i.valid = true
	i.position = i.idx.searchLE(ctx, stamp)
	return i.reload()
}

// SeekGE seeks to the domain whose TimeRange contain the provided timestamp. If no such domain
// exists, SeekGE seeks to the closest domain whose starting timestamp is greater than the
// provided timestamp. If no such domain exists, SeekGE returns false.
func (i *Iterator) SeekGE(ctx context.Context, stamp telem.TimeStamp) bool {
	if i.closed {
		return false
	}
	i.valid = true
	i.position = i.idx.searchGE(ctx, stamp)
	return i.reload()
}

// Next advances the iterator to the next domain. If the iterator has been exhausted, Next
// returns false.
func (i *Iterator) Next() bool {
	if !i.valid {
		return false
	}
	i.position++
	ok := i.reload()
	// If we've reached the end of the iterator, back up the position so that we
	// remain at the location of the last domain.
	if !ok {
		i.position--
	}
	return ok
}

// Prev advances the iterator to the previous domain. If the iterator has been exhausted,
// Prev returns false.
func (i *Iterator) Prev() bool {
	if !i.valid {
		return false
	}
	// Check if the position is zero to avoid uint32 underflow.
	if i.position == 0 {
		i.valid = false
		return i.valid
	}
	i.position--
	return i.reload()
}

// Valid returns true if the iterator is currently pointing to a valid domain and has
// not accumulated an error. Returns false otherwise.
func (i *Iterator) Valid() bool { return i.valid }

func (i *Iterator) Position() uint32 { return uint32(i.position) }

// TimeRange returns the time interval occupied by current domain.
func (i *Iterator) TimeRange() telem.TimeRange { return i.currPtr.TimeRange }

// OpenReader returns a new Reader that can be used to read telemetry from the current
// domain. The returned Reader is not safe for concurrent use, but it is safe to have
// multiple Readers open over the same domain. Note that the caller is responsible for
// closing the reader.
func (i *Iterator) OpenReader(ctx context.Context) (*Reader, error) {
	if i.closed {
		return nil, errIteratorClosed
	}
	return i.readerFactory(ctx, i.currPtr)
}

// Size returns the number of bytes occupied by the telemetry in the current domain.
func (i *Iterator) Size() telem.Size { return telem.Size(i.currPtr.size) }

// Close closes the iterator.
func (i *Iterator) Close() error {
	i.closed = true
	i.valid = false
	i.onClose()
	return nil
}

func (i *Iterator) reload() bool {
	if i.position == -1 {
		i.valid = false
		return i.valid
	}
	ptr, ok := i.idx.get(i.position)
	if !ok || !ptr.OverlapsWith(i.Bounds) {
		i.valid = false
		// it's important that we return here, so we don't clear the current current
		// value of the iterator.
		return i.valid
	}
	i.currPtr = ptr
	return i.valid
}
