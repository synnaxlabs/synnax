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

// offsetResolver is a function that resolves the offset of a sample within a domain.
// See the DB.Delete function for more details on the values this function should
// return.
type offsetResolver = func(
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
//	[...data, startDomain + startOffset, ...deleted..., endDomain + endOffset, ...data]
//
// The following requirements are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
func (db *DB) Delete(
	ctx context.Context,
	tr telem.TimeRange,
	calculateStartOffset offsetResolver,
	calculateEndOffset offsetResolver,
) (err error) {
	ctx, span := db.cfg.T.Bench(ctx, "Delete")
	defer span.End()

	// Pin before the check, so a Drop that runs in between waits for this call.
	db.resourceCount.Add(1)
	defer db.unpin()
	if db.closed.Load() {
		return ErrDBClosed
	}

	// Ensure that there cannot be deletion operations on the index between index lookup
	// as that would invalidate the offsets. However, we cannot lock the index as a
	// whole since index Distance() call requires using an iterator for the index's
	// domain, which is problematic if this channel is an index channel.
	db.idx.deleteLock.Lock()
	defer db.idx.deleteLock.Unlock()

	var (
		start, end deleteBound
		found      bool
	)
	if start, tr.Start, found, err = db.findDeleteStart(
		ctx,
		tr.Start,
		calculateStartOffset,
	); err != nil || !found {
		return err
	}
	if end, tr.End, found, err = db.findDeleteEnd(
		ctx,
		tr.End,
		calculateEndOffset,
	); err != nil || !found {
		return err
	}

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	db.idx.unprotectedRelocate(&start, &end)
	ok, err := validateDelete(&start, &end, db.idx)
	if err != nil || !ok {
		return span.Error(err)
	}

	// A garbage collection may have moved these domains within their file since the
	// lookup above. The partial pointers below inherit their offsets, so take the ones
	// the index holds now.
	start.ptr.offset = db.idx.mu.pointers[start.position].offset
	end.ptr.offset = db.idx.mu.pointers[end.position].offset

	db.idx.unprotectedSplice(start, end, tr)

	write := db.idx.indexPersist.prepare(start.position)
	db.idx.notifyHistoryChange(ctx, tr)
	// We choose to keep the mutex locked while persisting to index.
	err = write.run()
	if errors.Is(err, errUncovered) {
		err = db.idx.indexPersist.prepare(0).run()
	}
	return span.Error(err)
}

// deleteBound is one end of a deletion: the position of its domain in the index, the
// domain's pointer at lookup time, and the offset of the deletion within the domain.
type deleteBound struct {
	position int
	ptr      pointer
	offset   telem.Size
}

// findDeleteStart finds the first domain greater than or containing ts and the offset
// of ts within it. It returns ts snapped to the domain, and false if no domain follows
// ts.
func (db *DB) findDeleteStart(
	ctx context.Context,
	ts telem.TimeStamp,
	calculateOffset offsetResolver,
) (deleteBound, telem.TimeStamp, bool, error) {
	db.idx.mu.RLock()
	position, exact := db.idx.unprotectedSearch(ts.SpanRange(0))
	if exact {
		ptr := db.idx.mu.pointers[position]
		db.idx.mu.RUnlock()
		offset, ts, err := calculateOffset(ctx, ptr.Start, ts)
		if err != nil {
			return deleteBound{}, 0, false, err
		}
		return deleteBound{position: position, ptr: ptr, offset: offset}, ts, true, nil
	}
	// Non-exact: ts is not contained within any domain. Add 1 since we want the first
	// domain greater than ts.
	position += 1
	if position == len(db.idx.mu.pointers) {
		db.idx.mu.RUnlock()
		return deleteBound{}, 0, false, nil
	}
	ptr := db.idx.mu.pointers[position]
	db.idx.mu.RUnlock()
	return deleteBound{position: position, ptr: ptr}, ptr.Start, true, nil
}

// findDeleteEnd finds the last domain less than or containing ts and the offset of ts
// from the domain's end. It returns ts snapped to the domain, and false if no domain
// precedes ts.
func (db *DB) findDeleteEnd(
	ctx context.Context,
	ts telem.TimeStamp,
	calculateOffset offsetResolver,
) (deleteBound, telem.TimeStamp, bool, error) {
	db.idx.mu.RLock()
	position, exact := db.idx.unprotectedSearch(ts.SpanRange(0))
	if exact {
		ptr := db.idx.mu.pointers[position]
		db.idx.mu.RUnlock()
		offset, ts, err := calculateOffset(ctx, ptr.Start, ts)
		if err != nil {
			return deleteBound{}, 0, false, err
		}
		return deleteBound{
			position: position,
			ptr:      ptr,
			offset:   telem.Size(ptr.size) - offset,
		}, ts, true, nil
	}
	// Non-exact: ts is not contained within any domain.
	if position == -1 {
		db.idx.mu.RUnlock()
		return deleteBound{}, 0, false, nil
	}
	ptr := db.idx.mu.pointers[position]
	db.idx.mu.RUnlock()
	return deleteBound{position: position, ptr: ptr}, ptr.End, true, nil
}

// unprotectedRelocate searches the index again for the domains of start and end when
// they moved since the lookup. The caller must hold idx.mu.
func (idx *index) unprotectedRelocate(start, end *deleteBound) {
	if idx.mu.pointers[start.position] != start.ptr {
		var exact bool
		start.position, exact = idx.unprotectedSearch(start.ptr.TimeRange)
		// Edge cases such as startDomain is after the end must have been already
		// handled before: a time range that existed in the domain before must not cease
		// to exist.
		if !exact {
			start.position += 1
		}
	}
	if idx.mu.pointers[end.position] != end.ptr {
		end.position, _ = idx.unprotectedSearch(end.ptr.TimeRange)
	}
}

// unprotectedSplice replaces the domains from start to end with the parts of them that
// lie outside tr. The caller must hold idx.mu.
func (idx *index) unprotectedSplice(start, end deleteBound, tr telem.TimeRange) {
	newPointers := make([]pointer, 0)
	// Calculate size of removed pointers.
	var removedSize int64
	for i := start.position; i <= end.position; i++ {
		removedSize += int64(idx.mu.pointers[i].size)
		// Readers see what the delete leaves at once, also of an unpublished commit.
		idx.report(idx.unprotectedReveal(idx.mu.pointers[i].Start))
	}

	// Remove old pointers.
	idx.mu.pointers = append(
		idx.mu.pointers[:start.position],
		idx.mu.pointers[end.position+1:]...,
	)

	if start.offset != 0 {
		newPointers = append(newPointers, pointer{
			Start:   start.ptr.Start,
			End:     tr.Start,
			fileKey: start.ptr.fileKey,
			offset:  start.ptr.offset,
			size:    uint32(start.offset), // size from start.Start to tr.Start
		})
	}

	if end.offset != 0 {
		newPointers = append(newPointers, pointer{
			Start:   tr.End,
			End:     end.ptr.End,
			fileKey: end.ptr.fileKey,
			offset:  end.ptr.offset + end.ptr.size - uint32(end.offset),
			size:    uint32(end.offset), // size from tr.End to end.End
		})
	}

	// Calculate size of new partial pointers and update totalSize.
	var addedSize int64
	for _, p := range newPointers {
		addedSize += int64(p.size)
	}
	idx.totalSize.Add(addedSize - removedSize)

	if len(newPointers) != 0 {
		idx.mu.pointers = append(
			idx.mu.pointers[:start.position],
			append(newPointers, idx.mu.pointers[start.position:]...)...,
		)
	}
}

// GarbageCollect rewrites files that have accumulated enough tombstones to warrant
// compaction, as defined by Threshold.
func (db *DB) GarbageCollect(ctx context.Context) error {
	_, span := db.cfg.T.Bench(ctx, "garbage_collect")
	defer span.End()

	// Pin before the check, so a Drop that runs in between waits for this call.
	db.resourceCount.Add(1)
	defer db.unpin()
	if db.closed.Load() {
		return ErrDBClosed
	}

	if err := db.fc.gcWriters(); err != nil {
		return span.Error(err)
	}

	// There also cannot be any readers open on the file, since any iterators that
	// acquire those readers will be symlinked to the old file, causing them to read bad
	// data since the new pointers no longer correspond to the old file.
	//
	// WE ARE BLOCKING ALL READ OPERATIONS ON THE FILE DURING THE ENTIRE DURATION OF GC:
	// this is a behavior that we ideally change in the future to reduce downtime, but
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

	var collected bool
	for fileKey := 1; fileKey <= int(db.fc.counter.Value()); fileKey++ {
		s, err := db.cfg.FS.Stat(fileKeyToName(uint16(fileKey)))
		if err != nil {
			return span.Error(err)
		}
		fileCollected, err := db.garbageCollectFile(uint16(fileKey), s.Size())
		if err != nil {
			return span.Error(err)
		}
		collected = collected || fileCollected
	}

	// No file moved, so the index still matches what is on disk.
	if !collected {
		return nil
	}

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()
	write := db.idx.indexPersist.prepare(0)
	// We choose to keep the mutex locked while persisting pointers: the time sacrifice
	// should not be substantial, and this ensures that the order of index persists are
	// in order of garbage collect so that the index does reflect the correct indexes.
	return write.run()
}

// garbageCollectFile rewrites key's file to drop its tombstones, reporting whether it
// rewrote anything.
func (db *DB) garbageCollectFile(key uint16, size int64) (bool, error) {
	// Atomically remove this file from the writer pool so that no writer can open
	// on it during GC. If the file has an active writer, skip it.
	canGC, wasUnopened := db.fc.prepareForGC(key)
	if !canGC {
		return false, nil
	}

	var (
		name          = fileKeyToName(key)
		copyName      = name + "_gc"
		tombstoneSize = size
		ptrs          []pointer
	)

	restore := func() {
		if wasUnopened {
			db.fc.restoreUnopened(key)
		}
	}

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
			restore()
			return false, nil
		}
		// Otherwise, we continue with garbage collection while holding the mutex lock
		// to prevent more readers from being created.
	}

	// Find all pointers using the file. There can be fewer pointers than the file's
	// data accounts for due to deletion (these gaps are tombstones).
	db.idx.mu.RLock()
	for _, ptr := range db.idx.mu.pointers {
		if ptr.fileKey == key {
			ptrs = append(ptrs, ptr)
			tombstoneSize -= int64(ptr.size)
		}
	}
	db.idx.mu.RUnlock()

	// Decide whether we should GC. A file without tombstones has nothing to reclaim, so
	// rewriting it is pure churn.
	if tombstoneSize <= 0 ||
		tombstoneSize < int64(db.cfg.GCThreshold*float32(db.cfg.FileSize)) {
		restore()
		return false, nil
	}

	// Copy the live pointers into the copy file. Both handles must be closed before the
	// rename below: Windows refuses to rename a file that is still open.
	offsetDeltaMap, err := db.copyPointers(name, copyName, ptrs)
	if err != nil {
		return false, err
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
	if err := db.swapCollectedFile(key, name, copyName, offsetDeltaMap); err != nil {
		return false, err
	}

	if err := db.fc.rejuvenate(key); err != nil {
		return false, err
	}

	return true, db.cfg.FS.Remove(name + "_temp")
}

// copyPointers copies the data of ptrs from the file name into the file copyName, back
// to back. It returns a map from the time range of each moved pointer to the
// difference between its old and new offset. Time ranges are unique within a domain.
func (db *DB) copyPointers(
	name, copyName string,
	ptrs []pointer,
) (offsetDeltaMap map[telem.TimeRange]uint32, err error) {
	// Runs after the deferred closes, which can set err.
	defer func() {
		if err != nil {
			offsetDeltaMap = nil
		}
	}()
	offsetDeltaMap = make(map[telem.TimeRange]uint32)
	r, err := db.cfg.FS.Open(name, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()

	w, err := db.cfg.FS.Open(copyName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, w.Close()) }()

	var newOffset uint32
	for _, ptr := range ptrs {
		buf := make([]byte, ptr.size)
		if _, err = r.ReadAt(buf, int64(ptr.offset)); err != nil {
			return nil, err
		}

		n, err := w.Write(buf)
		if err != nil {
			return nil, err
		}

		if newOffset != ptr.offset {
			offsetDeltaMap[ptr.TimeRange] = ptr.offset - newOffset
		}
		newOffset += uint32(n)
	}
	return offsetDeltaMap, nil
}

// swapCollectedFile replaces the file name with the collected file copyName and moves
// the offsets of the file's pointers by offsetDeltaMap.
func (db *DB) swapCollectedFile(
	key uint16,
	name, copyName string,
	offsetDeltaMap map[telem.TimeRange]uint32,
) error {
	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()
	if err := db.cfg.FS.Rename(name, name+"_temp"); err != nil {
		return err
	}
	if err := db.cfg.FS.Rename(copyName, name); err != nil {
		return errors.Combine(err, db.cfg.FS.Rename(name+"_temp", name))
	}
	for i, ptr := range db.idx.mu.pointers {
		if ptr.fileKey != key {
			continue
		}
		if deltaOffset, ok := resolvePointerOffset(
			ptr.TimeRange,
			offsetDeltaMap,
		); ok {
			db.idx.mu.pointers[i].offset = ptr.offset - deltaOffset
			db.idx.persistHead = min(db.idx.persistHead, i)
		}
	}
	return nil
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
func validateDelete(start, end *deleteBound, idx *index) (bool, error) {
	var (
		startPosition, endPosition = start.position, end.position
		startOffset, endOffset     = &start.offset, &end.offset
	)
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

	startPtrLen, endPtrLen := telem.Size(
		idx.mu.pointers[startPosition].size,
	), telem.Size(
		idx.mu.pointers[endPosition].size,
	)
	if *startOffset > startPtrLen {
		*startOffset = startPtrLen
	}

	if *endOffset > endPtrLen {
		*endOffset = endPtrLen
	}

	// If the startPosition is greater than end position and there are samples in
	// between.
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

	if startPosition > endPosition ||
		(startPosition == endPosition-1 &&
			*startOffset == startPtrLen &&
			*endOffset == endPtrLen) ||
		startPosition == endPosition && *startOffset+*endOffset == startPtrLen {
		return false, nil
	}

	return true, nil
}
