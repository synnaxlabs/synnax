package legindex

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// Batch wraps a given index to provide a buffer for incoming alignments. The buffer
// holds all alignments until the caller calls either Commit or CommitBelowThreshold.
// The Batch provides the ability to search through both the buffer and the wrapped
// index.
type Batch struct {
	WrappedWriter   Writer
	WrappedSearcher Searcher
	Buf             *ThresholdBuffer
}

var _ Index = (*Batch)(nil)

// Key implements the Keyed interface.
func (b *Batch) Key() core.ChannelKey { return b.WrappedWriter.Key() }

// Release implements the Releaser interface.
func (b *Batch) Release() error {
	return b.WrappedSearcher.Release()
}

// Write implements the Writer interface.
func (b *Batch) Write(alignments []Alignment) error { return b.Buf.Add(alignments) }

// SearchP implements the Searcher interface.
func (b *Batch) SearchP(stamp telem.TimeStamp, approx position.Approximation) (position.Approximation, error) {
	return CompoundSearcher{b.Buf, b.WrappedSearcher}.SearchP(stamp, approx)
}

// SearchTS implements the Searcher interface.
func (b *Batch) SearchTS(pos position.Position, approx telem.Approximation) (telem.Approximation, error) {
	return CompoundSearcher{b.Buf, b.WrappedSearcher}.SearchTS(pos, approx)
}

// CommitBelowThreshold takes the given threshold position and commits any alignments
// whose last alignment is below the threshold to the wrapped index. The alignments are
// removed from the buffer.
func (b *Batch) CommitBelowThreshold(threshold position.Position) error {
	return b.Buf.WriteToBelowThreshold(threshold, b.WrappedWriter)
}

// Commit commits all alignments in the buffer to the wrapped index. The alignments are
// removed from the buffer.
func (b *Batch) Commit() error { return b.Buf.WriteTo(b.WrappedWriter) }
