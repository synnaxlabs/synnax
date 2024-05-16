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
// Note that the deletion timerange includes the sample at startOffset, and ends at
// the sample immediately before endOffset. Therefore, endOffset=0 denotes the sample past
// the pointer at endPosition.
//
// The following requirements are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
func (db *DB) Delete(ctx context.Context, startPosition int, endPosition int, startOffset int64, endOffset int64, tr telem.TimeRange) error {
	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	if err := validateDelete(startPosition, endPosition, &startOffset, &endOffset, db.idx); err != nil {
		return err
	}

	var (
		start       = db.idx.mu.pointers[startPosition]
		end         = db.idx.mu.pointers[endPosition]
		newPointers = make([]pointer, 0)
	)

	if startPosition == endPosition-1 && startOffset == int64(end.length) && endOffset == int64(end.length) ||
		startPosition == endPosition && startOffset+endOffset == int64(start.length) {
		// delete nothing
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

	if startPosition == endPosition && startOffset != 0 && endOffset != 0 {
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

	return db.idx.persist(ctx, startPosition)
}

// CollectTombstones must only collect files that are oversize since no new writers may
// be acquired on them. However, deletes may still happen on them.
func (db *DB) CollectTombstones(ctx context.Context) error {
	ctx, span := db.T.Bench(ctx, "GCTombstone")
	db.idx.mu.Lock()

	defer func() {
		span.End()
		db.idx.mu.Unlock()
	}()

	for fileKey, tombstoneSize := range db.idx.mu.tombstones {
		if tombstoneSize >= uint32(db.GCThreshold*float32(db.FileSize)) {
			var (
				fileName         = fileKeyToName(fileKey)
				copyName         = fileName + "_gc"
				newOffset uint32 = 0
			)

			// Rename the old file.
			if err := db.FS.Rename(fileName, copyName); err != nil {
				return span.Error(err)
			}

			r, err := db.FS.Open(copyName, os.O_RDONLY)
			if err != nil {
				return span.Error(err)
			}

			// Open a new file.
			f, err := db.FS.Open(fileName, os.O_CREATE|os.O_WRONLY)
			if err != nil {
				return span.Error(err)
			}

			// Find all pointers stored in the old file, and write them to the new file.
			for i, ptr := range db.idx.mu.pointers {
				if ptr.fileKey == fileKey {
					buf := make([]byte, ptr.length)
					_, err = r.ReadAt(buf, int64(ptr.offset))
					if err != nil {
						return span.Error(err)
					}

					_, err = f.WriteAt(buf, int64(newOffset))
					if err != nil {
						return span.Error(err)
					}

					db.idx.mu.pointers[i].offset = newOffset
					newOffset += ptr.length
				}
			}

			// Delete the old file
			if err = r.Close(); err != nil {
				return span.Error(err)
			}

			if err = f.Close(); err != nil {
				return span.Error(err)
			}

			if err = db.FS.Remove(copyName); err != nil {
				return span.Error(err)
			}

			// Remove entry from tombstones.
			delete(db.idx.mu.tombstones, fileKey)
		}
	}

	return nil
}

func validateDelete(startPosition int, endPosition int, startOffset *int64, endOffset *int64, idx *index) error {
	if startPosition < 0 || startPosition >= len(idx.mu.pointers) {
		return errors.Newf("[cesium] deletion starting at invalid domain position <%d> for length %d", startPosition, len(idx.mu.pointers))
	}

	if endPosition < 0 || endPosition >= len(idx.mu.pointers) {
		return errors.Newf("[cesium] deletion ending at invalid domain position <%d> for length %d", endPosition, len(idx.mu.pointers))
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
		return errors.Newf("[cesium] deletion start domain <%d> is greater than deletion end domain <%d>", startPosition, endPosition)
	}

	if startPosition == endPosition && *startOffset+*endOffset > int64(idx.mu.pointers[startPosition].length) {
		return errors.Newf("[cesium] deletion start offset <%d> is after end offset <%d>", *startOffset, *endOffset)
	}

	return nil
}
