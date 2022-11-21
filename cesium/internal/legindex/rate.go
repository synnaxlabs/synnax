package legindex

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

const IrregularRate = 1e9 * telem.Hz

type rateSearcher struct {
	telem.Rate
	nopReleaser
}

// RateSearcher returns a Searcher that uses the given rate to seek timestamps and positions.
// RateSearcher always returns positions with complete certainty.
func RateSearcher(rate telem.Rate) Searcher { return rateSearcher{Rate: rate} }

func (r rateSearcher) Key() core.ChannelKey { return core.ChannelKey(0) }

// SearchP implements Searcher.
func (r rateSearcher) SearchP(iPos telem.TimeStamp, _ position.Approximation) (position.Approximation, error) {
	pos := position.Position(telem.TimeSpan(iPos) / r.Period())
	return position.ExactlyAt(pos), nil
}

// SearchTS implements Searcher.
func (r rateSearcher) SearchTS(iPos position.Position, _ telem.Approximation) (telem.Approximation, error) {
	ts := telem.TimeStamp(telem.TimeSpan(iPos) * r.Period())
	return telem.ExactlyAt(ts), nil
}
