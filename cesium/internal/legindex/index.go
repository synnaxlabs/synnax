package legindex

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// Releaser is an index with resources that need to be released when no longer in use.
type Releaser interface {
	// Release releases any resources held by the index. Release should not be called
	// concurrently with any other method. After Release is called, the index is no
	// longer usable.
	Release() error
}

// Keyed is an index that can be indexed by a key.
type Keyed interface {
	// Key returns the key of the index.
	Key() core.ChannelKey
}

// Alignment is an alignment between a position on the root index and a timestamp.
type Alignment struct {
	// Pos is the alignment position.
	Pos position.Position
	// Stamp is the alignment time stamp.
	Stamp telem.TimeStamp
}

// PositionSearcher seeks positions on the root index given a TimeStamp and Approximation.
type PositionSearcher interface {
	// SearchP seeks a position based on the given TimeStamp and Approximation. The
	// guess should ensure that the position value is somewhere within its range.
	// Returns an Approximation representing the index's best resolution.
	SearchP(s telem.TimeStamp, guess position.Approximation) (position.Approximation, error)
	Releaser
	Keyed
}

// StampSearcher seeks TimeStamps on the root index given a position.
type StampSearcher interface {
	// SearchTS seeks a TimeStamp based on a given Position. The returned approximation
	// represents the index's best resolution of the TimeStamp.
	SearchTS(p position.Position, guess telem.Approximation) (telem.Approximation, error)
	Releaser
	Keyed
}

// Searcher is a searchable index.
type Searcher interface {
	PositionSearcher
	StampSearcher
}

// Writer is a writable index.
type Writer interface {
	// Write writes the given alignments to the index. The alignments must be in ascending
	// order.
	Write([]Alignment) error
	Releaser
	Keyed
}

// Index is a readable and writable index that allows a caller to translate
// root positions to timestamps and vice versa.
type Index interface {
	Searcher
	Writer
}

type nopReleaser struct{}

// Release implements the Releaser interface.
func (n nopReleaser) Release() error { return nil }
