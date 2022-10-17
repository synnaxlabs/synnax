package index

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

type Releaser interface {
	Release() error
}

type nopReleaser struct{}

func (n *nopReleaser) Release() error { return nil }

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
}

// StampSearcher seeks TimeStamps on the root index given a position.
type StampSearcher interface {
	// SearchTS seeks a TimeStamp based on a given Position. The returned approximation
	// represents the index's best resolution of the TimeStamp.
	SearchTS(p position.Position, guess telem.Approximation) (telem.Approximation, error)
	Releaser
}

// Searcher is a seekable index.
type Searcher interface {
	PositionSearcher
	StampSearcher
}

// Writer is a writable index.
type Writer interface {
	Write([]Alignment) error
	Releaser
}

// Index is a readable and writable index that allows a caller to translate
// root positions to timestamps and vice versa.
type Index interface {
	Searcher
	Writer
}
