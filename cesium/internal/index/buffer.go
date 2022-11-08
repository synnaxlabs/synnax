package index

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/array"
)

// ThresholdBuffer chunks index alignments and provides the ability to search
// through them.
type ThresholdBuffer struct {
	CompoundSearcher
	chunks [][]Alignment
}

// Add adds the given chunk of alignments to the buffer. The chunk is assumed to be
// sorted by position and time, and the first alignment in the chunk must be
// greater than the last alignment of the previously added chunk.
func (b *ThresholdBuffer) Add(chunk []Alignment) error {
	b.chunks = append(b.chunks, chunk)
	idx := &BinarySearch{Array: array.Searchable[Alignment]{Array: array.Wrap[Alignment](chunk)}}
	b.CompoundSearcher = append(b.CompoundSearcher, idx)
	return nil
}

// WriteTo writes all chunks to the given writer. The chunks are removed from the
// buffer.
func (b *ThresholdBuffer) WriteTo(w Writer) error {
	for _, buf := range b.chunks {
		if err := w.Write(buf); err != nil {
			return err
		}
	}
	b.chunks = nil
	b.CompoundSearcher = nil
	return nil
}

// WriteToBelowThreshold takes the given threshold position and writes any chunks whose
// last alignment is below the threshold to the given writer. The chunks are removed
// from the buffer.
func (b *ThresholdBuffer) WriteToBelowThreshold(threshold position.Position, w Writer) error {
	flushThreshold := -1
	for i := len(b.chunks) - 1; i >= 0; i-- {
		buf := b.chunks[i]
		if len(buf) == 0 {
			continue
		}
		last := buf[len(buf)-1]
		if last.Pos <= threshold {
			flushThreshold = i
			break
		}
	}
	if flushThreshold == -1 {
		return nil
	}
	for i := 0; i <= flushThreshold; i++ {
		buf := b.chunks[i]
		if len(buf) == 0 {
			continue
		}
		if err := w.Write(buf); err != nil {
			return err
		}
	}
	b.chunks = b.chunks[flushThreshold+1:]
	b.CompoundSearcher = b.CompoundSearcher[flushThreshold+1:]
	return nil
}
