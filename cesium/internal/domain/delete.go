package domain

import (
	"context"
	"errors"
	"github.com/synnaxlabs/x/telem"
	"io"
	"os"
	"strconv"
)

// Delete adds all pointers ranging from
// [db.get(startPosition).offset + startOffset, db.get(endPosition).offset + endOffset)
// into tombstone.
// Note that the deletion timerange includes the sample at startOffset, and ends at
// the sample right before endOffset
//
// The following invariants are placed on the variables:
// 0 <= startPosition <= endPosition < len(db.mu.idx.pointers), and must both be valid
// positions in the index.
//
// If an endOffset overshoots the length of that pointer, then it is "clipped" to the end
// i.e. any offset >= length will delete the whole pointer
//
// Similarly, if a startOffset undershoots, then it is "clipped" to the start.
//
// If endOffset = 0, the entirety of endPosition pointer is excluded.
// If endOffset = len(db.get(endPosition)), the entirety of endPosition pointer is included.
// If startOffset = 0, the entirety of startPosition pointer is included.
// If startOffset = len(db.get(endPosition)), the operation is invalid.
//
// ** Special case: if endOffset = -1, then it is equal to the length of the end pointer.
func (db *DB) Delete(ctx context.Context, startPosition int, endPosition int, startOffset int64, endOffset int64, tr telem.TimeRange) error {
	start, ok := db.idx.get(startPosition)
	if !ok {
		return errors.New("[cesium] Deletion starting at invalid position")
	}
	end, ok := db.idx.get(endPosition)
	if !ok {
		return errors.New("[cesium] Deletion ending at invalid position")
	}

	if startPosition > endPosition {
		if startPosition == endPosition+1 && startOffset == 0 && (uint32(endOffset) == end.length || endOffset == -1) {
			// Deleting nothing.
			return nil
		}

		return errors.New("[cesium] Deletion starting position cannot be greater than ending position")
	}

	if endOffset == -1 {
		endOffset = int64(end.length)
	}

	if startOffset >= int64(start.length) {
		return errors.New("[cesium] Deletion start offset cannot be greater than or equal to the length of that pointer")
	}

	if startOffset < 0 {
		return errors.New("[cesium] Deletion start offset cannot be less than 0")
	}

	if endOffset > int64(end.length) {
		return errors.New("[cesium] Deletion end offset cannot be greater than the length of that pointer")
	}

	if startPosition != endPosition {
		// Remove end of start pointer
		db.idx.insertTombstone(ctx, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.Start,
				End:   start.End,
			},
			fileKey: start.fileKey,
			offset:  start.offset + uint32(startOffset),
			length:  start.length - uint32(startOffset), // length of {tr.Start, start.End}
		})

		for _, p := range db.idx.mu.pointers[startPosition+1 : endPosition] {
			db.idx.insertTombstone(ctx, p)
		}

		// Remove start of end pointer.
		if endOffset != 0 {
			db.idx.insertTombstone(ctx, pointer{
				TimeRange: telem.TimeRange{
					Start: end.Start,
					End:   tr.End,
				},
				fileKey: end.fileKey,
				offset:  end.offset,
				length:  uint32(endOffset), // length of {end.Start, tr.End}
			})
		}
	} else {
		if startOffset > endOffset {
			return errors.New("[cesium] Deletion start offset cannot be greater than end offset in same pointer")
		}

		if startOffset == endOffset {
			return nil
		}

		db.idx.insertTombstone(ctx, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.Start,
				End:   tr.End,
			},
			fileKey: start.fileKey,
			offset:  start.offset + uint32(startOffset),
			length:  uint32(endOffset - startOffset),
		})
	}

	// Remove old pointers.
	db.idx.mu.pointers = append(db.idx.mu.pointers[:startPosition+1], db.idx.mu.pointers[endPosition+1:]...)

	newPointers := make([]pointer, 0)

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

	if uint32(endOffset) != end.length {
		newPointers = append(newPointers, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.End,
				End:   end.End,
			},
			fileKey: end.fileKey,
			offset:  end.offset + uint32(endOffset),
			length:  end.length - uint32(endOffset), // length from tr.End to end.End
		})
	}

	temp := make([]pointer, len(db.idx.mu.pointers)+len(newPointers)-1)
	copy(temp, db.idx.mu.pointers[:startPosition])

	if len(newPointers) != 0 {
		copy(temp[startPosition:startPosition+len(newPointers)], newPointers)
	}

	copy(temp[startPosition+len(newPointers):], db.idx.mu.pointers[startPosition+1:])

	db.idx.mu.pointers = temp

	return nil
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

func minInt(a uint32, b uint32) int {
	if a > b {
		return int(b)
	} else {
		return int(a)
	}
}
