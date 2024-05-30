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
func (db *DB) Delete(
	ctx context.Context,
	startOffset int64,
	endOffset int64,
	tr telem.TimeRange,
) error {
	ctx, span := db.T.Bench(ctx, "Delete")
	defer span.End()

	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	startPosition, exact := db.idx.unprotectedSearch(tr.Start.SpanRange(0))
	if !exact {
		startPosition += 1
	}

	endPosition, _ := db.idx.unprotectedSearch(tr.End.SpanRange(0))

	err, ok := validateDelete(startPosition, endPosition, &startOffset, &endOffset, db.idx)
	if err != nil || !ok {
		return span.Error(err)
	}

	var (
		start       = db.idx.mu.pointers[startPosition]
		end         = db.idx.mu.pointers[endPosition]
		newPointers = make([]pointer, 0)
	)

	// Remove old pointers.
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

	if len(newPointers) != 0 {
		db.idx.mu.pointers = append(db.idx.mu.pointers[:startPosition], append(newPointers, db.idx.mu.pointers[startPosition:]...)...)
	}

	persistPointers := db.idx.indexPersist.preparePointersPersist(startPosition)
	return span.Error(persistPointers())
}

// GarbageCollect rewrites all files that are over the size limit of a file and has
// enough tombstones to garbage collect, as defined by GCThreshold.
func (db *DB) GarbageCollect(ctx context.Context) error {
	ctx, span := db.T.Bench(ctx, "garbage_collect")
	defer span.End()

	_, err := db.files.gcWriters()
	if err != nil {
		return span.Error(err)
	}

	var fileKey uint16

	for fileKey = 1; fileKey <= uint16(db.files.counter.Value()); fileKey++ {
		if db.files.hasWriter(fileKey) {
			continue
		}
		s, err := db.FS.Stat(fileKeyToName(fileKey))
		if err != nil {
			return span.Error(err)
		}
		if s.Size() >= int64(db.FileSize) {
			err = db.garbageCollectFile(fileKey, s.Size())
			if err != nil {
				return span.Error(err)
			}
		}
	}

	db.idx.mu.RLock()
	persistPointers := db.idx.indexPersist.preparePointersPersist(0)
	db.idx.mu.RUnlock()
	return persistPointers()
}

func (db *DB) garbageCollectFile(key uint16, size int64) error {
	var (
		name                 = fileKeyToName(key)
		copyName             = name + "_gc"
		newOffset     uint32 = 0
		tombstoneSize        = size
		ptrs          []pointer
		// offsetMap maps the old offset to the new offset for any pointer. Note
		// that pointer offsets are unique within a file.
		offsetMap = make(map[uint32]uint32)
	)

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
	if tombstoneSize < int64(db.GCThreshold*float32(db.FileSize)) {
		return nil
	}

	// Open a reader on the old file.
	r, err := db.FS.Open(name, os.O_RDONLY)
	if err != nil {
		return err
	}

	// Open a writer to the copy file.
	w, err := db.FS.Open(copyName, os.O_WRONLY|os.O_CREATE)
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

		_, err = w.WriteAt(buf, int64(newOffset))
		if err != nil {
			return err
		}

		if newOffset != ptr.offset {
			offsetMap[ptr.offset] = newOffset
		}
		newOffset += ptr.length
	}

	if err = r.Close(); err != nil {
		return err
	}

	if err = w.Close(); err != nil {
		return err
	}

	// Update the file and index while holding the mutex lock.
	db.idx.mu.Lock()
	for i, ptr := range db.idx.mu.pointers {
		if ptr.fileKey == key {
			oldOffset := ptr.offset
			if o, ok := offsetMap[oldOffset]; ok {
				db.idx.mu.pointers[i].offset = o
			}
		}
	}

	err = db.FS.Rename(name, name+"_temp")
	if err != nil {
		db.idx.mu.Unlock()
		return err
	}
	err = db.FS.Rename(copyName, name)
	db.idx.mu.Unlock()

	if err = db.files.rejuvenate(key); err != nil {
		return err
	}

	return db.FS.Remove(name + "_temp")
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
		return errors.Newf("deletion start offset %d is after end offset %d", *startOffset, *endOffset), false
	}

	if (startPosition == endPosition-1 && *startOffset == int64(idx.mu.pointers[endPosition].length) && *endOffset == int64(idx.mu.pointers[endPosition].length)) ||
		startPosition == endPosition && *startOffset+*endOffset == int64(idx.mu.pointers[startPosition].length) {
		return nil, false
	}

	return nil, true
}
