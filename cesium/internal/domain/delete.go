// Copyright 2026 Synnax Labs, Inc.
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
	"os"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// OffsetResolver is a function that resolves the offset of a sample within a domain.
// See the DB.Delete function for more details on the values this function should
// return.
type OffsetResolver = func(
	ctx context.Context,
	domainStart telem.TimeStamp,
	ts telem.TimeStamp,
) (telem.Size, telem.TimeStamp, error)

// Delete deletes a set of samples in the DB, purging and splitting underlying domains
// as necessary. The operation is performed in the following manner.
//
// 1. The positions of the start and end domains are found within the database index.
// 2. calculateStartOffset(startDomainPosition) is called to find the start offset (in
// number of samples) from the beginning of the start domain that needs to be deleted.
// All data that comes before the returned offset will be kept, and all data that comes
// after will be deleted. This is inclusive, meaning that the sample at the returned
// offset will be deleted.
// 3. calculateEndOffset(endDomainPosition) is called to find the end offset  (in number
// of samples) from the end of the end domain that needs to be deleted. All data in the
// end domain from the end to the returned offset (backwards)will be kept, and all data
// that comes before will be deleted. This is exclusive, meaning that the sample at the
// returned end offset will not be deleted.
//
// As a short summary, the delete operation resembles the following code:
//
//	startDomainIndex := db.index.find(tr.Start)
//	endDomainIndex := db.index.find(tr.End)
//	endOffset := calculateEndOffset(endDomainIndex)
//	startOffset := calculateStartOffset(startDomainIndex)
//	[...data, startDomain + startOffset, ...deleted..., endDomain + endOffset, ...data...]
//
// The following requirements are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
func (db *DB) Delete(
	ctx context.Context,
	tr telem.TimeRange,
	calculateStartOffset OffsetResolver,
	calculateEndOffset OffsetResolver,
) (err error) {
	ctx, span := db.cfg.T.Bench(ctx, "Delete")
	defer span.End()

	if db.closed.Load() {
		return ErrDBClosed
	}
	db.resourceCount.Add(1)
	defer db.resourceCount.Add(-1)

	// Ensure that there cannot be deletion operations on the index between index lookup
	// as that would invalidate the offsets. However, we cannot lock the index as a
	// whole since index Distance() call requires using an iterator for the index's
	// domain, which is problematic if this channel is an index channel.
	db.idx.deleteLock.Lock()
	defer db.idx.deleteLock.Unlock()

	var (
		startDomain, endDomain int
		startOffset, endOffset telem.Size
		start, end             pointer
		newPointers            = make([]pointer, 0)
	)

	// Search for the start position: the first domain greater or containing tr.Start.
	db.idx.mu.RLock()
	startDomain, exact := db.idx.unprotectedSearch(tr.Start.SpanRange(0))
	if exact {
		start = db.idx.mu.pointers[startDomain]
		db.idx.mu.RUnlock()
		startOffset, tr.Start, err = calculateStartOffset(ctx, start.Start, tr.Start)
		if err != nil {
			return err
		}
	} else {
		// Non-exact: tr.Start is not contained within any domain.
		// Add 1 since we want the first domain greater than tr.Start.
		startDomain += 1

		if startDomain == len(db.idx.mu.pointers) {
			// delete nothing
			db.idx.mu.RUnlock()
			return err
		}

		start = db.idx.mu.pointers[startDomain]
		db.idx.mu.RUnlock()
		startOffset = 0
		tr.Start = start.Start
	}

	// Search for the end position: the first domain less or containing tr.End.
	db.idx.mu.RLock()
	endDomain, exact = db.idx.unprotectedSearch(tr.End.SpanRange(0))
	if exact {
		end = db.idx.mu.pointers[endDomain]
		db.idx.mu.RUnlock()
		if endOffset, tr.End, err = calculateEndOffset(ctx, end.Start, tr.End); err != nil {
			return err
		}
		endOffset = telem.Size(end.size) - endOffset
	} else {
		// Non-exact: tr.End is not contained within any domain.
		if endDomain == -1 {
			// delete nothing
			db.idx.mu.RUnlock()
			return err
		}

		end = db.idx.mu.pointers[endDomain]
		db.idx.mu.RUnlock()
		endOffset = 0
		tr.End = end.End
	}

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	// Repêchage: the location of start/end may have changed during the index lookup.
	if db.idx.mu.pointers[startDomain] != start {
		startDomain, exact = db.idx.unprotectedSearch(start.TimeRange)
		// Edge cases such as startDomain is after the end must have been already
		// handled before: a time range that existed in the domain before must not cease
		// to exist.
		if !exact {
			startDomain += 1
		}
	}
	if db.idx.mu.pointers[endDomain] != end {
		endDomain, _ = db.idx.unprotectedSearch(end.TimeRange)
	}

	ok, err := validateDelete(startDomain, endDomain, &startOffset, &endOffset, db.idx)
	if err != nil || !ok {
		return span.Error(err)
	}

	// Remove old pointers.
	db.idx.mu.pointers = append(db.idx.mu.pointers[:startDomain], db.idx.mu.pointers[endDomain+1:]...)

	if startOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{Start: start.Start, End: tr.Start},
			fileKey:   start.fileKey,
			offset:    start.offset,
			size:      uint32(startOffset), // size from start.Start to tr.Start
		})
	}

	if endOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{Start: tr.End, End: end.End},
			fileKey:   end.fileKey,
			offset:    end.offset + end.size - uint32(endOffset),
			size:      uint32(endOffset), // size from tr.End to end.End
		})
	}

	if len(newPointers) != 0 {
		db.idx.mu.pointers = append(
			db.idx.mu.pointers[:startDomain],
			append(newPointers, db.idx.mu.pointers[startDomain:]...)...,
		)
	}

	persist := db.idx.indexPersist.prepare(startDomain)
	// We choose to keep the mutex locked while persisting to index.
	return span.Error(persist())
}

// GarbageCollect rewrites all files that are over the size limit of a file and has
// enough tombstones to garbage collect, as defined by Threshold.
func (db *DB) GarbageCollect(ctx context.Context) error {
	_, span := db.cfg.T.Bench(ctx, "garbage_collect")
	defer span.End()

	if db.closed.Load() {
		return ErrDBClosed
	}
	db.resourceCount.Add(1)
	defer db.resourceCount.Add(-1)

	if _, err := db.fc.gcWriters(); err != nil {
		return span.Error(err)
	}

	// There also cannot be any readers open on the file, since any iterators that
	// acquire those readers will be symlinked to the old file, causing them to read bad
	// data since the new pointers no longer correspond to the old file.
	//
	// WE ARE BLOCKING ALL READ OPERATIONS ON THE FILE DURING THE ENTIRE DURATION OF GC:
	// this is a behaviour that we ideally change in the future to reduce downtime, but
	// for now, this is what we implemented. The challenge is with the two files during
	// GC: one copy file is made and an original file is made. However, existing file
	// handles will point to the original file instead of the new file, even after the
	// original file is renamed and "deleted" (unix does not actually delete the file
	// when there is a file handle open on it). This means that the pointers in the
	// index will reflect the updates made to the garbage collected file, but the old
	// file handles will no longer match the updated pointers, resulting in incorrect
	// read positions.

	// There are some potential solutions to this:
	//     1. Add a lock on readers before each read operation, and swap the underlying
	//        file handle for each reader under a lock.
	//     2. Use a one-file GC system where no duplicate file is created.
	//     3. Wait during GC until all file handles are closed, then swap the file under
	//        a lock on the file to disallow additional readers from being created.
	//        (This might be problematic since some readers may never get closed).
	if _, err := db.fc.gcReaders(); err != nil {
		return span.Error(err)
	}

	for fileKey := uint16(1); fileKey <= uint16(db.fc.counter.Value()); fileKey++ {
		if db.fc.hasWriter(fileKey) {
			continue
		}
		s, err := db.cfg.FS.Stat(fileKeyToName(fileKey))
		if err != nil {
			return span.Error(err)
		}
		if s.Size() < int64(db.cfg.FileSize) {
			continue
		}

		if err = db.garbageCollectFile(fileKey, s.Size()); err != nil {
			return span.Error(err)
		}
	}

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()
	persist := db.idx.indexPersist.prepare(0)
	// We choose to keep the mutex locked while persisting pointers: the time sacrifice
	// should not be substantial, and this ensures that the order of index persists are
	// in order of garbage collect so that the index does reflect the correct indexes.
	return persist()
}

func (db *DB) garbageCollectFile(key uint16, size int64) error {
	var (
		name                 = fileKeyToName(key)
		copyName             = name + "_gc"
		newOffset     uint32 = 0
		tombstoneSize        = size
		ptrs          []pointer
		// offsetDeltaMap maps each pointer (identified by the time range) to the difference
		// between its new offset and its old offset. Note that time ranges are
		// necessarily unique within a domain.
		offsetDeltaMap = make(map[telem.TimeRange]uint32)
	)

	db.fc.readers.RLock()
	defer db.fc.readers.RUnlock()
	rs, ok := db.fc.readers.files[key]
	// It's ok if there is no reader entry for the file, this means that no reader has
	// been created. And we can be sure that no reader will be created since we hold the
	// fc.readers mutex as well, preventing the readers map from being modified.
	if ok {
		rs.RLock()
		defer rs.RUnlock()
		// If there's any open file handles on the file, we cannot garbage collect.
		if len(rs.open) > 0 {
			return nil
		}
		// Otherwise, we continue with garbage collection while holding the mutex lock
		// to prevent more readers from being created.
	}

	// Find all pointers using the file: there cannot be more pointers using the file
	// during GC since the file must be already full — however, there can be less due to
	// deletion.
	db.idx.mu.RLock()
	for _, ptr := range db.idx.mu.pointers {
		if ptr.fileKey == key {
			ptrs = append(ptrs, ptr)
			tombstoneSize -= int64(ptr.size)
		}
	}
	db.idx.mu.RUnlock()

	// Decide whether we should GC
	if tombstoneSize < int64(db.cfg.GCThreshold*float32(db.cfg.FileSize)) {
		return nil
	}

	// Open a reader on the old file.
	r, err := db.cfg.FS.Open(name, os.O_RDONLY)
	if err != nil {
		return err
	}

	// Open a writer to the copy file.
	w, err := db.cfg.FS.Open(copyName, os.O_WRONLY|os.O_CREATE)
	if err != nil {
		return err
	}

	// Find all pointers stored in the old file, and write them to the new file.
	for _, ptr := range ptrs {
		buf := make([]byte, ptr.size)
		_, err = r.ReadAt(buf, int64(ptr.offset))
		if err != nil {
			return err
		}

		n, err := w.Write(buf)
		if err != nil {
			return err
		}

		if newOffset != ptr.offset {
			offsetDeltaMap[ptr.TimeRange] = ptr.offset - newOffset
		}
		newOffset += uint32(n)
	}

	if err = r.Close(); err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	// Update the file and index while holding the mutex lock. Note: the index might be
	// different at this point than before: old pointers on this file may be split into
	// multiple smaller pointers with different offsets. (Understand that since this
	// deletion occurred after garbage collection, it should not be garbage collected in
	// this run of GC.) However, two things cannot change:
	// 1. The resulting pointers from a pointer deletion, no matter into how many,
	// cannot end up in a larger time range than the original pointer.
	// 2. The delta in offset, i.e. oldOffset - newOffset are the same for all smaller,
	// resulting pointers from the original split.
	//
	// Using these two principles, we can find the new offset for any pointer with a
	// time range contained in the original pointer by subtracting it by the same delta.
	if err := func() error {
		db.idx.mu.Lock()
		defer db.idx.mu.Unlock()
		for i, ptr := range db.idx.mu.pointers {
			if ptr.fileKey == key {
				if deltaOffset, ok := resolvePointerOffset(ptr.TimeRange, offsetDeltaMap); ok {
					db.idx.mu.pointers[i].offset = ptr.offset - deltaOffset
				}
			}
		}

		if err = db.cfg.FS.Rename(name, name+"_temp"); err != nil {
			return err
		}
		return db.cfg.FS.Rename(copyName, name)
	}(); err != nil {
		return err
	}

	if err = db.fc.rejuvenate(key); err != nil {
		return err
	}

	return db.cfg.FS.Remove(name + "_temp")
}

func resolvePointerOffset(
	ptrRange telem.TimeRange,
	offsetDeltaMap map[telem.TimeRange]uint32,
) (uint32, bool) {
	for domain, delta := range offsetDeltaMap {
		if domain.ContainsRange(ptrRange) {
			return delta, true
		}
	}
	return 0, false
}

// validateDelete returns an error if the deletion request is invalid. In addition, it
// returns true if there is some data to be deleted (i.e. deleting nothing).
func validateDelete(
	startPosition int,
	endPosition int,
	startOffset *telem.Size,
	endOffset *telem.Size,
	idx *index,
) (bool, error) {
	if startPosition == len(idx.mu.pointers) {
		return false, nil
	}

	if endPosition == -1 {
		return false, nil
	}

	if *startOffset < 0 {
		*startOffset = 0
	}

	if *endOffset < 0 {
		*endOffset = 0
	}

	startPtrLen, endPtrLen := telem.Size(idx.mu.pointers[startPosition].size), telem.Size(idx.mu.pointers[endPosition].size)
	if *startOffset > startPtrLen {
		*startOffset = startPtrLen
	}

	if *endOffset > endPtrLen {
		*endOffset = endPtrLen
	}

	// If the startPosition is greater than end position and there are samples in between.
	if startPosition > endPosition && (startPosition != endPosition+1 ||
		*startOffset != 0 ||
		*endOffset != 0) {
		return false, errors.Newf(
			"deletion start domain %d is greater than deletion end domain %d",
			startPosition,
			endPosition,
		)
	}

	if startPosition == endPosition && *startOffset+*endOffset > startPtrLen {
		return false, errors.Newf(
			"deletion start offset %d is after end offset %d for size %d",
			*startOffset,
			*endOffset,
			idx.mu.pointers[startPosition].size,
		)
	}

	if (startPosition == endPosition-1 && *startOffset == endPtrLen && *endOffset == endPtrLen) ||
		startPosition == endPosition && *startOffset+*endOffset == startPtrLen {
		return false, nil
	}

	return true, nil
}
