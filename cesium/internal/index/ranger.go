package index

import (
	"encoding/binary"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"io"
)

type Ranger struct {
	DB     *ranger.DB
	Logger *zap.Logger
}

// Distance implements Index.
func (i *Ranger) Distance(tr telem.TimeRange, continuous bool) (count int64, err error) {
	i.Logger.Debug("idx distance",
		zap.Stringer("timeRange", tr),
		zap.Bool("continuous", continuous),
	)
	var startPos, endPos int64
	defer func() {
		i.Logger.Debug("idx distance done",
			zap.Stringer("timeRange", tr),
			zap.Bool("continuous", continuous),
			zap.Int64("count", count),
			zap.Int64("startPos", startPos),
			zap.Int64("endPos", endPos),
			zap.Error(err),
		)
	}()

	iter := i.DB.NewIterator(ranger.IteratorConfig{Bounds: tr})

	if !iter.SeekFirst() || (!iter.Range().ContainsRange(tr) && continuous) {
		err = ErrDiscontinuous
		return
	}

	if tr.IsZero() {
		count = 0
		return
	}

	r, err := iter.NewReader()
	if err != nil {
		return
	}

	startPos, err = i.searchGE(tr.Start, r)
	if err != nil {
		return
	}

	if iter.Range().ContainsStamp(tr.End) || tr.End == iter.Range().End {
		endPos, err = i.searchLT(tr.End, r)
		if err != nil {
			return 0, err
		}
		return endPos - startPos, nil
	} else if continuous {
		return 0, ErrDiscontinuous
	}

	startDist := (iter.Len() / 8) - startPos
	var gap int64 = 0

	for {
		if !iter.Next() {
			if continuous {
				return 0, ErrDiscontinuous
			}
			return startDist + (iter.Len() / 8) + gap, nil
		}
		if iter.Range().ContainsStamp(tr.End) {
			r, err = iter.NewReader()
			if err != nil {
				return 0, err
			}
			endPos, err = i.searchLT(tr.End, r)
			if err != nil {
				return 0, err
			}
			return startDist + endPos + gap, nil
		}
		gap += iter.Len()
	}
}

func (i *Ranger) Stamp(ref telem.TimeStamp, offset int64) (telem.TimeStamp, error) {
	iter := i.DB.NewIterator(ranger.IterRange(ref.SpanRange(telem.TimeSpanMax)))
	if !iter.SeekFirst() ||
		iter.Len()/int64(telem.TimeStampDensity) <= offset ||
		!iter.Range().ContainsStamp(ref) {
		return 0, ErrDiscontinuous
	}

	if offset == 0 {
		return ref, nil
	}

	r, err := iter.NewReader()
	if err != nil {
		return 0, err
	}
	pos, err := i.searchGE(ref, r)
	if err != nil {
		return 0, err
	}
	pos += offset
	ts, err := readTimeStamp(r, pos*8, make([]byte, 8))
	return ts, err
}

func (i *Ranger) searchLT(ts telem.TimeStamp, r io.Reader) (int64, error) {
	pos, _, err := i.searchInsert(ts, r.(*ranger.Reader))
	return pos - 1, err
}

func (i *Ranger) searchGE(ts telem.TimeStamp, r *ranger.Reader) (int64, error) {
	pos, _, err := i.searchInsert(ts, r)
	return pos, err
}

func (i *Ranger) searchInsert(ts telem.TimeStamp, r *ranger.Reader) (int64, bool, error) {
	var (
		start int64 = 0
		end   int64 = (r.Len() / 8) - 1
	)
	for start <= end {
		mid := (start + end) / 2
		midTs, err := readTimeStamp(r, mid*8, make([]byte, 8))
		if err != nil {
			return 0, false, err
		}
		if midTs == ts {
			return mid, true, nil
		} else if midTs < ts {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return end + 1, false, nil
}

func readTimeStamp(r io.ReaderAt, offset int64, buf []byte) (telem.TimeStamp, error) {
	_, err := r.ReadAt(buf, offset)
	return telem.TimeStamp(binary.LittleEndian.Uint64(buf)), err
}
