package index

import (
	"encoding/binary"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/telem"
)

func newReaderIndex(reader storage.Reader[any]) *readerIndex {
	return &readerIndex{reader: reader}
}

//type ReaderIndex struct {
//	Reader storage.Reader[]
//	Iter   core.MDIterator
//	mu     sync.Mutex
//}
//
//func (r *ReaderIndex) SearchP(stamp telem.TimeStamp, guess position.Range) (position.Range, error) {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//	r.Iter.SetBounds(guess)
//	var (
//		ltOk  = r.Iter.SeekFirst()
//		first = position.Position(-1)
//		last  = position.Position(0)
//	)
//	if ltOk {
//		for r.Iter.Next(guess.Span()) {
//			md := r.Iter.Value()
//			segments, err := r.Reader.Read(md)
//			if err != nil {
//				return position.RangeZero, err
//			}
//			if first == -1 {
//				first = segments[0].Alignment
//			}
//			for _, seg := range segments {
//				for i := 0; i < len(seg.SData); i += 8 {
//					ts := DecodeTimeStamp(seg.SData[i : i+8])
//					if ts.AfterEq(stamp) {
//						pos := seg.Alignment.Add(position.Span(i / 8))
//						return pos.SpanRange(0), nil
//					}
//				}
//				last = seg.Alignment.Add(position.Span(len(seg.SData) / 8))
//			}
//		}
//	}
//	if !r.Iter.SeekGE(guess.Start) {
//		if ltOk {
//			return last.SpanRange(0), nil
//		}
//		return position.RangeZero, errors.New("index has no data")
//	}
//	return first.SpanRange(0), nil
//}
//
//func (r *ReaderIndex) SeekS(pos position.Position) (telem.TimeRange, error) {
//	r.mu.Lock()
//	defer r.mu.Unlock()
//	r.Iter.SetBounds(position.Range{Start: pos, End: pos})
//	ltOk := r.Iter.SeekLE(pos)
//	if ltOk {
//		for r.Iter.Next(position.Span(1)) {
//			md := r.Iter.Value()
//			segments, err := r.Reader.Read(md)
//			if err != nil {
//				return telem.TimeRangeZero, err
//			}
//			for _, seg := range segments {
//				if seg.Alignment.AfterEq(pos) {
//					return DecodeTimeStamp(seg.SData[pos.Sub(position.Span(seg.Alignment))*8:]).SpanRange(0), nil
//				}
//			}
//		}
//	}
//	return telem.TimeRangeZero, errors.New("index has no data")
//}
//

func DecodeTimeStamp(data []byte) telem.TimeStamp {
	return telem.TimeStamp(binary.BigEndian.Uint64(data))
}
