package index

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

type rateSeeker struct{ telem.Rate }

// RateSearcher returns a Searcher that uses the given rate to seek timestamps and positions.
// RateSearcher always returns positions with complete certainty.
func RateSearcher(rate telem.Rate) Searcher { return rateSeeker{rate} }

// SearchP implements Searcher.
func (r rateSeeker) SearchP(iPos telem.TimeStamp, _ position.Approximation) (position.Approximation, error) {
	pos := position.Position(telem.TimeSpan(iPos) / r.Period())
	return position.ExactlyAt(pos), nil
}

// SearchTS implements SeekTS.
func (r rateSeeker) SearchTS(iPos position.Position, _ telem.Approximation) (telem.Approximation, error) {
	ts := telem.TimeStamp(telem.TimeSpan(iPos) * r.Period())
	return telem.CertainlyAt(ts), nil
}
