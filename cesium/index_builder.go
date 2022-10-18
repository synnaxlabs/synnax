package cesium

import (
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
)

// groupChannelsByIndexSearcher takes a set of given channels, groups them by their index,
// and returns a map of index searchers to the channels that belong to them.
func (d *db) groupChannelsByIndexSearcher(channels []Channel) (map[index.Searcher][]Channel, error) {
	var (
		// map of channels to the key of the index they belong to
		nonRateChannels = make(map[ChannelKey][]Channel)
		// map of channels to the fixed rate index they belong to
		rateChannels = make(map[telem.Rate][]Channel)
	)

	// group channels into their category and acquire index searchers when necessary.
	for _, ch := range channels {
		if ch.Index == 0 {
			rateChannels[ch.Rate] = append(rateChannels[ch.Rate], ch)
		} else {
			nonRateChannels[ch.Index] = append(nonRateChannels[ch.Index], ch)
		}
	}

	// group channels by their index searcher
	indexes := make(map[index.Searcher][]Channel)
	// instantiate a new rate searcher for each rate
	for rate, chs := range rateChannels {
		indexes[index.RateSearcher(rate)] = chs
	}
	// acquire index searchers for each non-rate index
	for idxKey, chs := range nonRateChannels {
		idx, err := d.indexes.acquireSearcher(idxKey)
		if err != nil {
			return nil, err
		}
		indexes[idx] = chs
	}
	return indexes, nil
}
