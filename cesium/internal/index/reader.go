package index

import (
	"encoding/binary"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/telem"
)

// Reader is an index that uses a core.PositionIterator to resolve timestamps
// and positions from a storage.Reader.
type Reader struct {
	// Reader is the storage.Reader to read segments from.
	Reader storage.Reader
	// Iter is the core.PositionIterator to use to resolve positions. Control
	// over the iterator (including closure) should be relinquished to the index.
	Iter        core.PositionIterator
	targetStamp telem.TimeStamp
	targetPos   position.Position
}

var _ Searcher = (*Reader)(nil)

// SearchP implements Searcher.
func (r *Reader) SearchP(stamp telem.TimeStamp, approx position.Approximation) (position.Approximation, error) {
	r.Iter.SetBounds(approx.Range)
	r.targetStamp = stamp
	nApprox := position.Uncertain
	for r.Iter.SeekFirst(); r.Iter.Next(approx.Span()); {
		segments, err := r.Reader.Read(r.Iter.Value())
		if err != nil {
			return position.Uncertain, err
		}
		alignments, err := decodeAlignments(segments)
		if err != nil {
			return position.Uncertain, err
		}
		nApprox = binarySearchP(stamp, approx, alignments)
		if nApprox.Start == position.Min || nApprox.Exact() {
			return nApprox, nil
		}
	}
	return nApprox, nil
}

// SearchTS implements Searcher.
func (r *Reader) SearchTS(pos position.Position, approx telem.Approximation) (telem.Approximation, error) {
	r.Iter.SetBounds(position.Range{Start: pos.Sub(10), End: pos.Add(10)})
	nApprox := telem.Uncertain
	for r.Iter.SeekFirst(); r.Iter.Next(position.Span(20)); {
		segments, err := r.Reader.Read(r.Iter.Value())
		if err != nil {
			return telem.Uncertain, err
		}
		alignments, err := decodeAlignments(segments)
		if err != nil {
			return telem.Uncertain, err
		}
		nApprox = binarySearchTS(pos, approx, alignments)
		if nApprox.Start == telem.TimeStampMin || nApprox.Exact() {
			return nApprox, nil
		}
	}
	return nApprox, nil
}

// Release implements Releaser.
func (r *Reader) Release() error { return r.Iter.Close() }

func DecodeTimeStamp(data []byte) telem.TimeStamp {
	return telem.TimeStamp(binary.BigEndian.Uint64(data))
}

func decodeAlignments(segments []core.SugaredSegment) ([]Alignment, error) {
	alignmentCount := 0
	for _, seg := range segments {
		alignmentCount += telem.TimeStampDensity.SampleCount(seg.Size())
	}
	alignments := make([]Alignment, 0, alignmentCount)
	for _, seg := range segments {
		for i := 0; i < len(seg.Data); i += 8 {
			alignments = append(alignments, Alignment{
				Pos:   seg.Alignment + position.Position(i/8),
				Stamp: DecodeTimeStamp(seg.Data[i : i+8]),
			})
		}
	}
	return alignments, nil
}
