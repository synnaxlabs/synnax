package index

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/telem"
)

var (
	ErrDiscontinuous = errors.New("discontinuous")
)

// Index implements an index over a time series.
type Index interface {
	// Distance calculates the number of samples between the start and end of the given time
	// range.
	Distance(tr telem.TimeRange, continuous bool) (int64, error)
	// Stamp calculates a timestamp sampleCount samples after the given reference timestamp.
	Stamp(ref telem.TimeStamp, distance int64) (telem.TimeStamp, error)
}
