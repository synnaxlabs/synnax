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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
	"os"
)

// Delete adds all pointers ranging from
// [db.get(startPosition).offset + startOffset, db.get(endPosition).offset + length - endOffset)
// into tombstone.
// Note that the deletion time range includes the sample at startOffset, and ends at
// the sample immediately before endOffset. Therefore, endOffset=0 denotes the sample past
// the pointer at endPosition.
//
// The following requirements are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
func (db *DB) Delete(
	ctx context.Context,
	calculateStartOffset func(
		ctx context.Context,
		domainStart telem.TimeStamp,
		ts telem.TimeStamp,
	) (int64, telem.TimeStamp, error),
	calculateEndOffset func(
		ctx context.Context,
		domainStart telem.TimeStamp,
		ts telem.TimeStamp,
	) (int64, telem.TimeStamp, error),
	tr telem.TimeRange,
	den telem.Density,
) (err error) {
	ctx, span := db.cfg.T.Bench(ctx, "Delete")
	defer span.End()

	if db.closed.Load() {
		return errDBClosed
	}
	db.entityCount.Add(1)
	defer db.entityCount.Add(-1)

	// Ensure that there cannot be deletion operations on the index between index lookup
	// as that would invalidate the offsets.
	// However, we cannot lock the index as a whole since index Distance() call requires
	// using an iterator for the index's domain, which is problematic if this channel
	// is an index channel.
	db.idx.deleteLock.Lock()
	defer db.idx.deleteLock.Unlock()

	var (
		startPosition, endPosition int
		startOffset, endOffset     int64
		start, end                 pointer
		newPointers                = make([]pointer, 0)
	)

	// Search for the start position: the first domain greater or containing tr.Start.
	db.idx.mu.RLock()
	startPosition, exact := db.idx.unprotectedSearch(tr.Start.SpanRange(0))
	if exact {
		start = db.idx.mu.pointers[startPosition]
		db.idx.mu.RUnlock()
		startOffset, tr.Start, err = calculateStartOffset(ctx, start.Start, tr.Start)
		if err != nil {
			return
		}
		startOffset = int64(den.Size(startOffset))
	} else {
		// Non-exact: tr.Start is not contained within any domain.
		// Add 1 since we want the first domain greater than tr.Start.
		startPosition += 1

		if startPosition == len(db.idx.mu.pointers) {
			// delete nothing
			db.idx.mu.RUnlock()
			return
		}

		start = db.idx.mu.pointers[startPosition]
		db.idx.mu.RUnlock()
		startOffset = 0
		tr.Start = start.Start
	}

	// Search for the end position: the first domain less or containing tr.End.
	db.idx.mu.RLock()
	endPosition, exact = db.idx.unprotectedSearch(tr.End.SpanRange(0))
	if exact {
		end = db.idx.mu.pointers[endPosition]
		db.idx.mu.RUnlock()
		endOffset, tr.End, err = calculateEndOffset(ctx, end.Start, tr.End)
		if err != nil {
			return
		}
		endOffset = int64(end.length) - int64(den.Size(endOffset))
	} else {
		// Non-exact: tr.End is not contained within any domain.
		if endPosition == -1 {
			// delete nothing
			db.idx.mu.RUnlock()
			return
		}

		end = db.idx.mu.pointers[endPosition]
		db.idx.mu.RUnlock()
		endOffset = 0
		tr.End = end.End
	}

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	// Repêchage: the location of start/end may have changed during the index lookup.
	if db.idx.mu.pointers[startPosition] != start {
		startPosition, exact = db.idx.unprotectedSearch(start.TimeRange)
		// Edge cases such as startPosition is after the end must have been already
		// handled before: a time range that existed in the domain before must not cease
		// to exist.
		if !exact {
			startPosition += 1
		}
	}
	if db.idx.mu.pointers[endPosition] != end {
		endPosition, _ = db.idx.unprotectedSearch(end.TimeRange)
	}

	err, ok := validateDelete(startPosition, endPosition, &startOffset, &endOffset, db.idx)
	if err != nil || !ok {
		return span.Error(err)
	}

	// Remove old pointers.
	db.idx.mu.pointers = append(db.idx.mu.pointers[:startPosition], db.idx.mu.pointers[endPosition+1:]...)

	if startOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{Start: start.Start, End: tr.Start},
			fileKey:   start.fileKey,
			offset:    start.offset,
			length:    uint32(startOffset), // length from start.Start to tr.Start
		})
	}

	if endOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{Start: tr.End, End: end.End},
			fileKey:   end.fileKey,
			offset:    end.offset + end.length - uint32(endOffset),
			length:    uint32(endOffset), // length from tr.End to end.End
		})
	}

	if len(newPointers) != 0 {
		db.idx.mu.pointers = append(
			db.idx.mu.pointers[:startPosition],
			append(newPointers, db.idx.mu.pointers[startPosition:]...)...,
		)
	}

	persist := db.idx.indexPersist.prepare(startPosition)
	// We choose to keep the mutex locked while persisting to index.
	return span.Error(persist())
}

// GarbageCollect rewrites all files that are over the size limit of a file and has
// enough tombstones to garbage collect, as defined by GCThreshold.
func (db *DB) GarbageCollect(ctx context.Context) error {
	ctx, span := db.cfg.T.Bench(ctx, "garbage_collect")
	defer span.End()

	if db.closed.Load() {
		return errDBClosed
	}
	db.entityCount.Add(1)
	defer db.entityCount.Add(-1)

	_, err := db.fc.gcWriters()
	if err != nil {
		return span.Error(err)
	}

	// There also cannot be any readers open on the file, since any iterators that
	// acquire those readers will be symlinked to the old file, causing them to read
	// bad data since the new pointers no longer correspond to the old file.
	//
	// WE ARE BLOCKING ALL READ OPERATIONS ON THE FILE DURING THE ENTIRE DURATION OF GC:
	// this is a behaviour that we ideally change in the future to reduce downtime, but
	// for now, this is what we implemented.
	// The challenge is with the two files during GC: one copy file is made and an
	// original file is made. However, existing file handles will point to the original
	// file instead of the new file, even after the original file is renamed and "deleted"
	// (unix does not actually delete the file when there is a file handle open on it).
	// This means that the pointers in the index will
	// reflect the updates made to the garbage collected file, but the old file handles
	// will no longer match the updated pointers, resulting in incorrect read positions.

	// There are some potential solutions to this:
	//     1. Add a lock on readers before each read operation, and swap the underlying
	// file handle for each reader under a lock.
	//     2. Use a one-file GC system where no duplicate file is created.
	//     3. Wait during GC until all file handles are closed, then swap the file under
	// a lock on the file to disallow additional readers from being created. (This might
	// be problematic since some readers may never get closed).
	_, err = db.fc.gcReaders()
	if err != nil {
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
		// offsetMap maps each pointer (identified by the time range) to the difference
		// between its new offset and its old offset. Note that time ranges are
		// necessarily unique within a domain.
		offsetDeltaMap = make(map[telem.TimeRange]uint32)
	)

	db.fc.readers.RLock()
	defer db.fc.readers.RUnlock()
	rs, ok := db.fc.readers.files[key]
	// It's ok if there is no reader entry for the file, this means that no reader has
	// been created. And we can be sure that no reader will be created since we hold
	// the fc.readers mutex as well, preventing the readers map from being modified.
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
	// during GC since the file must be already full – however, there can be less due
	// to deletion.
	db.idx.mu.RLock()
	for _, ptr := range db.idx.mu.pointers {
		if ptr.fileKey == key {
			ptrs = append(ptrs, ptr)
			tombstoneSize -= int64(ptr.length)
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
		buf := make([]byte, ptr.length)
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

	// Update the file and index while holding the mutex lock.
	// Note: the index might be different at this point than before: old pointers on
	// this file may be split into multiple smaller pointers with different offsets.
	// (Understand that since this deletion occurred after garbage collection, it should
	// not be garbage collected in this run of GC.)
	// However, two things cannot change:
	// 1. The resulting pointers from a pointer deletion, no matter into how many,
	// cannot end up in a larger time range than the original pointer.
	// 2. The delta in offset, i.e. oldOffset - newOffset are the same for all smaller,
	// resulting pointers from the original split.
	//
	// Using these two principles, we can find the new offset for any pointer with a
	// time range contained in the original pointer by subtracting it by the same delta.
	db.idx.mu.Lock()
	for i, ptr := range db.idx.mu.pointers {
		if ptr.fileKey == key {
			if deltaOffset, ok := resolvePointerOffset(ptr.TimeRange, offsetDeltaMap); ok {
				db.idx.mu.pointers[i].offset = ptr.offset - deltaOffset
			}
		}
	}

	if err = db.cfg.FS.Rename(name, name+"_temp"); err != nil {
		db.idx.mu.Unlock()
		return err
	}
	if err = db.cfg.FS.Rename(copyName, name); err != nil {
		db.idx.mu.Unlock()
		return err
	}
	db.idx.mu.Unlock()

	if err = db.fc.rejuvenate(key); err != nil {
		return err
	}

	return db.cfg.FS.Remove(name + "_temp")
}

func resolvePointerOffset(ptrRange telem.TimeRange, offsetDeltaMap map[telem.TimeRange]uint32) (uint32, bool) {
	for domain, delta := range offsetDeltaMap {
		if domain.ContainsRange(ptrRange) {
			return delta, true
		}
	}

	return 0, false
}

// validateDelete returns an error if the deletion request is valid. In addition, it
// returns true if there is some data to be deleted (i.e. deleting nothing).
func validateDelete(startPosition int, endPosition int, startOffset *int64, endOffset *int64, idx *index) (error, bool) {
	if startPosition == len(idx.mu.pointers) {
		return nil, false
	}

	if endPosition == -1 {
		return nil, false
	}

	if *startOffset < 0 {
		*startOffset = 0
	}

	if *endOffset < 0 {
		*endOffset = 0
	}

	if *startOffset > int64(idx.mu.pointers[startPosition].length) {
		*startOffset = int64(idx.mu.pointers[startPosition].length)
	}

	if *endOffset > int64(idx.mu.pointers[endPosition].length) {
		*endOffset = int64(idx.mu.pointers[endPosition].length)
	}

	// If the startPosition is greater than end position and there are samples in between.
	if startPosition > endPosition && !(startPosition == endPosition+1 &&
		*startOffset == 0 &&
		*endOffset == 0) {
		return errors.Newf("deletion start domain %d is greater than deletion end domain %d", startPosition, endPosition), false
	}

	if startPosition == endPosition && *startOffset+*endOffset > int64(idx.mu.pointers[startPosition].length) {
		return errors.Newf("deletion start offset %d is after end offset %d for length %d", *startOffset, *endOffset, idx.mu.pointers[startPosition].length), false
	}

	if (startPosition == endPosition-1 && *startOffset == int64(idx.mu.pointers[endPosition].length) && *endOffset == int64(idx.mu.pointers[endPosition].length)) ||
		startPosition == endPosition && *startOffset+*endOffset == int64(idx.mu.pointers[startPosition].length) {
		return nil, false
	}

	return nil, true
}
