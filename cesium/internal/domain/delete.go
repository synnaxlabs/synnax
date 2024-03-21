package domain

import (
	"context"
	"errors"
	"github.com/synnaxlabs/x/telem"
	"io"
	"os"
	"strconv"
)

// Delete adds all pointers ranging from [db.get(startPosition).start + startOffset, db.get(endPosition).end - endOffset) into tombstone
func (db *DB) Delete(ctx context.Context, startPosition int, endPosition int, startOffset int64, endOffset int64, tr telem.TimeRange) error {
	start, ok := db.idx.get(startPosition, false)
	if !ok {
		return errors.New("Invalid starting position")
	}
	end, ok := db.idx.get(endPosition, false)
	if !ok {
		return errors.New("Invalid ending position")
	}

	if startPosition != endPosition {
		// remove end of start pointer
		db.idx.insertTombstone(ctx, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.Start,
				End:   start.End,
			},
			fileKey: start.fileKey,
			offset:  start.offset + uint32(startOffset),
			length:  start.length - uint32(startOffset), // length of {tr.Start, start.End}
		}, false)

		for _, p := range db.idx.mu.pointers[startPosition+1 : endPosition] {
			db.idx.insertTombstone(ctx, p, false)
		}

		// remove start of end pointer
		db.idx.insertTombstone(ctx, pointer{
			TimeRange: telem.TimeRange{
				Start: end.Start,
				End:   tr.End,
			},
			fileKey: end.fileKey,
			offset:  end.offset,
			length:  end.length - uint32(endOffset), // length of {end.Start, tr.End}
		}, false)
	} else {
		db.idx.insertTombstone(ctx, pointer{
			TimeRange: telem.TimeRange{
				Start: tr.Start,
				End:   tr.End,
			},
			fileKey: start.fileKey,
			offset:  uint32(startOffset),
			length:  start.length - uint32(startOffset) - uint32(endOffset),
		}, false)
	}

	// remove old pointers
	db.idx.mu.pointers = append(db.idx.mu.pointers[:startPosition+1], db.idx.mu.pointers[endPosition+1:]...)

	newPointers := []pointer{
		{
			TimeRange: telem.TimeRange{
				Start: start.Start,
				End:   tr.Start,
			},
			fileKey: start.fileKey,
			offset:  start.offset,
			length:  uint32(startOffset), // length from start.Start to tr.Start
		},
		{
			TimeRange: telem.TimeRange{
				Start: tr.End,
				End:   end.End,
			},
			fileKey: end.fileKey,
			offset:  end.offset + end.length - uint32(endOffset), // end.offset + length of from tr.End to tr.End
			length:  uint32(endOffset),                           // length from tr.End to tr.Start
		},
	}

	temp := make([]pointer, len(db.idx.mu.pointers)+1)
	copy(temp, db.idx.mu.pointers[:startPosition])
	copy(temp[startPosition:startPosition+2], newPointers)
	copy(temp[startPosition+2:], db.idx.mu.pointers[startPosition+1:])

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
			// pointers
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
