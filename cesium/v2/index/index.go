package index

import (
	"encoding/binary"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/telem"
	"io"
)

// Index is an index over a collection of timestamps.
type Index interface {
	// Distance calculates the number of samples between a reference timestamp
	// and a set of target timestamps.
	Distance(ref telem.TimeStamp, targets []telem.TimeStamp) ([]int64, error)
	// DefinedAndContiguous checks whether the index is defined and contiguous
	// over the provided time range.
	DefinedAndContiguous(tr telem.TimeRange) (bool, error)
	// Search
	End(ref telem.TimeStamp, dist int64) (telem.TimeStamp, error)
}

type rangerIndex struct {
	DB *ranger.DB
}

func (i *rangerIndex) Distance(start, end telem.TimeStamp) (int64, error) {
	iter := i.DB.NewIterator(&ranger.IteratorConfig{
		Bounds: telem.TimeRange{Start: start, End: end},
	})
	if !iter.SeekFirst() {
		return 0, errors.New("no data")
	}
	if !iter.Range().ContainsStamp(start) {
		return 0, errors.New("no data")
	}
	if iter.Range().ContainsStamp(end) {
		r, err := iter.NewReader()
		if err != nil {
			return 0, err
		}
		startPos, err := i.search(start, r)
		if err != nil {
			return 0, err
		}
		endPos, err := i.search(end, r)
		if err != nil {
			return 0, err
		}
		return endPos - startPos, nil
	}
	panic("not implemented")
}

func (i *rangerIndex) search(stamp telem.TimeStamp, r *ranger.Reader) (int64, error) {
	startI, endI := int64(0), r.Len()/8
	for startI <= endI {
		mid := (startI + endI) / 2
		t, err := readTimeStamp(r, mid*8, make([]byte, 8))
		if err != nil {
			return 0, err
		}
		if t < stamp {
			startI = mid + 1
		} else if t > stamp {
			endI = mid - 1
		} else {
			return mid, nil
		}
	}
	return startI, nil
}

func readTimeStamp(r io.ReaderAt, offset int64, buf []byte) (telem.TimeStamp, error) {
	_, err := r.ReadAt(buf, offset)
	return telem.TimeStamp(binary.LittleEndian.Uint64(buf)), err
}
