package domain

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
	"os"
	"sort"
)

const tombstoneByteSize = 6

// Delete adds all pointers ranging from
// [db.get(startPosition).offset + startOffset, db.get(endPosition).offset + length - endOffset)
// into tombstone.
// Note that the deletion timerange includes the sample at startOffset, and ends at
// the sample immediately before endOffset. Therefore, endOffset=0 denotes the sample past
// the pointer at endPosition.
//
// The following requirements are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
func (db *DB) Delete(
	ctx context.Context,
	startPosition int,
	endPosition int,
	startOffset int64,
	endOffset int64,
	tr telem.TimeRange,
) error {
	ctx, span := db.T.Bench(ctx, "Delete")
	defer span.End()

	// TODO: think about defer unlock here
	db.idx.mu.Lock()

	if err := validateDelete(startPosition, endPosition, &startOffset, &endOffset, db.idx); err != nil {
		db.idx.mu.Unlock()
		return span.Error(err)
	}

	var (
		start       = db.idx.mu.pointers[startPosition]
		end         = db.idx.mu.pointers[endPosition]
		newPointers = make([]pointer, 0)
	)

	// TODO: refactor
	if startPosition == endPosition-1 && startOffset == int64(end.length) && endOffset == int64(end.length) ||
		startPosition == endPosition && startOffset+endOffset == int64(start.length) {
		// delete nothing
		db.idx.mu.Unlock()
		return nil
	}

	// Remove old pointers.
	for i := startPosition + 1; i < endPosition; i++ {
		ptr := db.idx.mu.pointers[i]
		db.idx.mu.tombstones[ptr.fileKey] += ptr.length
	}
	db.idx.mu.pointers = append(db.idx.mu.pointers[:startPosition], db.idx.mu.pointers[endPosition+1:]...)

	if startOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{
				Start: start.Start,
				End:   tr.Start,
			},
			fileKey: start.fileKey,
			offset:  start.offset,
			length:  uint32(startOffset), // length from start.Start to tr.Start
		})
	}
	db.idx.mu.tombstones[start.fileKey] += start.length - uint32(startOffset)

	if endOffset != 0 {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.End,
				End:   end.End,
			},
			fileKey: end.fileKey,
			offset:  end.offset + end.length - uint32(endOffset),
			length:  uint32(endOffset), // length from tr.End to end.End
		})
	}
	db.idx.mu.tombstones[end.fileKey] += end.length - uint32(endOffset)

	if startPosition == endPosition {
		// If start and end are in the same domain, then we only keep their intersection
		// as the tombstone. Calculated via the PIE.
		db.idx.mu.tombstones[end.fileKey] -= end.length
	}

	if len(newPointers) != 0 {
		temp := make([]pointer, len(db.idx.mu.pointers)+len(newPointers))
		copy(temp, db.idx.mu.pointers[:startPosition])
		copy(temp[startPosition:startPosition+len(newPointers)], newPointers)
		copy(temp[startPosition+len(newPointers):], db.idx.mu.pointers[startPosition:])
		db.idx.mu.pointers = temp
	}

	persistPointers := db.idx.indexPersist.preparePointersPersist(startPosition)
	persistTombstones := db.idx.indexPersist.prepareTombstonePersist()
	db.idx.mu.Unlock()
	if err := persistPointers(); err != nil {
		return span.Error(err)
	}
	return span.Error(persistTombstones())
}

// GarbageCollect rewrites all files that have tombstones in them to remove the garbage
// data, close all file handles on them, and add them to the unopened files set if they
// have space for writing after garbage collect.
func (db *DB) GarbageCollect(ctx context.Context) error {
	ctx, span := db.T.Bench(ctx, "GCTombstone")
	defer span.End()

	db.idx.mu.Lock()

	// TODO: need tombstones?
	for fileKey, tombstoneSize := range db.idx.mu.tombstones {
		if tombstoneSize >= uint32(db.GCThreshold*float32(db.FileSize)) {
			var (
				fileName         = fileKeyToName(fileKey)
				newOffset uint32 = 0
				pointers         = make([]*pointer, 0)
			)

			f, err := db.FS.Open(fileName, os.O_RDWR)
			if err != nil {
				db.idx.mu.Unlock()
				return span.Error(err)
			}

			// Find all pointers stored in the file, and sort them in order of offset.
			for i, ptr := range db.idx.mu.pointers {
				if ptr.fileKey == fileKey {
					pointers = append(pointers, &db.idx.mu.pointers[i])
				}
			}

			sort.Slice(pointers, func(i, j int) bool {
				return pointers[i].offset < pointers[j].offset
			})

			for _, ptr := range pointers {
				buf := make([]byte, int(ptr.length))
				_, err = f.ReadAt(buf, int64(ptr.offset))
				if err != nil {
					db.idx.mu.Unlock()
					return err
				}
				n, err := f.WriteAt(buf, int64(newOffset))
				if err != nil {
					db.idx.mu.Unlock()
					return err
				}

				ptr.offset = newOffset
				newOffset += uint32(n)
			}

			if err = f.Truncate(int64(newOffset)); err != nil {
				db.idx.mu.Unlock()
				return span.Error(err)
			}

			if err = f.Close(); err != nil {
				db.idx.mu.Unlock()
				return span.Error(err)
			}

			// Remove entry from tombstones.
			delete(db.idx.mu.tombstones, fileKey)

			err = db.files.rejuvenate(fileKey)
			if err != nil {
				db.idx.mu.Unlock()
				return span.Error(err)
			}
		}
	}

	persistTombstones := db.idx.indexPersist.prepareTombstonePersist()
	db.idx.mu.Unlock()
	return persistTombstones()
}

func validateDelete(startPosition int, endPosition int, startOffset *int64, endOffset *int64, idx *index) error {
	if startPosition < 0 || startPosition >= len(idx.mu.pointers) {
		return errors.Newf("deletion starting at invalid domain position %d for length %d", startPosition, len(idx.mu.pointers))
	}

	if endPosition < 0 || endPosition >= len(idx.mu.pointers) {
		return errors.Newf("deletion ending at invalid domain position %d for length %d", endPosition, len(idx.mu.pointers))
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
		return errors.Newf("deletion start domain %d is greater than deletion end domain %d", startPosition, endPosition)
	}

	if startPosition == endPosition && *startOffset+*endOffset > int64(idx.mu.pointers[startPosition].length) {
		return errors.Newf("deletion start offset %d is after end offset %d", *startOffset, *endOffset)
	}

	return nil
}
