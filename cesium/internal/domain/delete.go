package domain

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
	"io"
	"os"
	"strconv"
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
		temp := make([]pointer, len(db.idx.mu.pointers)+len(newPointers))
		copy(temp, db.idx.mu.pointers[:startPosition])
		copy(temp[startPosition:startPosition+len(newPointers)], newPointers)
		copy(temp[startPosition+len(newPointers):], db.idx.mu.pointers[startPosition:])
		db.idx.mu.pointers = temp
	}

	return db.idx.persist(ctx, startPosition)
}

func (db *DB) CollectTombstones(ctx context.Context, maxSizeRead uint32) error {
	db.idx.mu.Lock()
	defer db.idx.mu.Unlock()

	for fileKey, tombstones := range db.idx.mu.tombstones {
		var (
			// priorTombstoneLength tracks the number of lengths of all tombstone
			// pointers before the current pointer during the course of rewriting
			// pointers.
			priorTombstoneLength uint32 = 0
			pointerPtr                  = 0
			tombstonePtr                = 0
		)

		r, err := db.files.acquireReader(ctx, fileKey)
		if err != nil {
			return err
		}
		f, err := db.FS.Open(strconv.Itoa(int(fileKey))+"_temp.domain", os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}

		for pointerPtr < len(db.idx.mu.pointers) {
			currentPointer := &db.idx.mu.pointers[pointerPtr]
			if currentPointer.fileKey != fileKey {
				pointerPtr++
				continue
			}

			for tombstonePtr < len(tombstones) && currentPointer.offset > tombstones[tombstonePtr].offset {
				priorTombstoneLength += db.idx.mu.tombstones[fileKey][tombstonePtr].length
				tombstonePtr++
			}

			n := 0

			for n < int(currentPointer.length) {
				buf := make([]byte, minInt(maxSizeRead, currentPointer.length))
				_n, err := r.ReadAt(buf, int64(currentPointer.offset)+int64(n))
				if err != nil && err != io.EOF {
					return err
				}
				_, err = f.WriteAt(buf, int64(currentPointer.offset)+int64(n)-int64(priorTombstoneLength))
				if err != nil {
					return err
				}
				n += _n
			}
			currentPointer.offset -= priorTombstoneLength
			pointerPtr += 1
		}

		err = db.files.removeReadersWriters(ctx, fileKey)
		if err != nil {
			return err
		}

		err = db.FS.Remove(strconv.Itoa(int(fileKey)) + ".domain")
		if err != nil {
			return err
		}
		err = db.FS.Rename(strconv.Itoa(int(fileKey))+"_temp.domain", strconv.Itoa(int(fileKey))+".domain")
		if err != nil {
			return err
		}
	}

	return nil
}

func validateDelete(startPosition int, endPosition int, startOffset *int64, endOffset *int64, idx *index) error {
	if startPosition < 0 || startPosition >= len(idx.mu.pointers) {
		return errors.Newf("[cesium] deletion starting at invalid domain position <%d>", startPosition)
	}

	if endPosition < 0 || endPosition >= len(idx.mu.pointers) {
		return errors.Newf("[cesium] deletion ending at invalid domain position <%d>", endPosition)
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
		return errors.Newf("[cesium] deletion start offset <%d> is greater than end offset <%d>", *startOffset, *endOffset)
	}

	return nil
}

func minInt(a uint32, b uint32) int {
	if a > b {
		return int(b)
	} else {
		return int(a)
	}
}
