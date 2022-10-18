package index

import (
	"bytes"
	"encoding/binary"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/telem"
	"io"
)

// Reader is an index that uses a core.PositionIterator to resolve timestamps
// and positions from a storage.Reader.
type Reader struct {
	// Reader is the storage.Reader to read segments from.
	Reader storage.Reader
	// Iter is the core.PositionIterator to use to resolve positions. Control
	// over the iterator (including closure) should be relinquished to the index.
	Iter core.PositionIterator
}

var _ Searcher = (*Reader)(nil)

// SearchP implements Searcher.
func (r *Reader) SearchP(stamp telem.TimeStamp, guess position.Approximation) (position.Approximation, error) {
	r.Iter.SetBounds(guess.Range)
	var (
		firstPos = position.Position(-1)
		first    = telem.TimeStamp(-1)
		last     = telem.TimeStamp(0)
		lastPos  = position.Position(-1)
	)
	for r.Iter.SeekFirst(); r.Iter.Next(guess.Span()); {
		segments, err := r.Reader.Read(r.Iter.Value())
		if err != nil {
			return position.Uncertain, err
		}
		for _, seg := range segments {
			approx := position.Uncertain
			IterTimeStamps(seg.Data, func(j int, ts telem.TimeStamp) bool {
				if first == -1 {
					first = ts
					firstPos = calcPos(seg.Alignment, j)
				}
				last = ts
				lastPos = calcPos(seg.Alignment, j)
				if ts.AfterEq(stamp) {
					approx = position.ExactlyAt(calcPos(seg.Alignment, j))
					return false
				}
				return true
			})
			if approx.Exact() {
				return approx, nil
			}
		}
	}

	if firstPos == -1 || lastPos == -1 {
		return position.Uncertain, nil
	}
	firstDiff := stamp.Sub(telem.TimeSpan(first)).Abs()
	lastDiff := stamp.Sub(telem.TimeSpan(last)).Abs()
	if firstDiff < lastDiff {
		return position.ExactlyAt(firstPos - 1), nil
	} else {
		return position.ExactlyAt(lastPos + 1), nil
	}
}

// SearchTS implements Searcher.
func (r *Reader) SearchTS(pos position.Position, guess telem.Approximation) (telem.Approximation, error) {
	r.Iter.SetBounds(position.Range{Start: pos.Sub(10), End: pos.Add(10)})
	for r.Iter.SeekFirst(); r.Iter.Next(position.Span(20)); {
		segments, err := r.Reader.Read(r.Iter.Value())
		if err != nil {
			return telem.Uncertain, err
		}
		for _, seg := range segments {
			approx := telem.Uncertain
			IterTimeStamps(seg.Data, func(i int, ts telem.TimeStamp) bool {
				if calcPos(seg.Alignment, i).AfterEq(pos) {
					approx = telem.ExactlyAt(ts)
					return false
				}
				return true
			})
			if approx.Exact() {
				return approx, nil
			}
		}
	}
	return telem.Uncertain, nil
}

// Release implements Releaser.
func (r *Reader) Release() error { return r.Iter.Close() }

func DecodeTimeStamp(data []byte) telem.TimeStamp {
	return telem.TimeStamp(binary.BigEndian.Uint64(data))
}

func IterTimeStamps(data []byte, f func(int, telem.TimeStamp) bool) {
	var (
		reader = bytes.NewReader(data)
		b      = make([]byte, 8)
		i      = 0
	)
	for {
		_, err := reader.Read(b)
		if errors.Is(err, io.EOF) {
			return
		}
		ts := DecodeTimeStamp(b)
		if !f(i, ts) {
			return
		}
		i++
	}
}

func calcPos(base position.Position, i int) position.Position {
	return base.Add(position.Span(i))
}
