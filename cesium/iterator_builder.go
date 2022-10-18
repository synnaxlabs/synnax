package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/telem"
)

func (d *db) newStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (*streamIterator, error) {
	channels, err := d.RetrieveChannels(keys...)
	if err != nil {
		return nil, err
	}

	// build a position iterator for each of our requested channels.
	positionIters := d.buildPositionIters(channels)
	if err != nil {
		return nil, err
	}

	// group channels by their index
	indexes, err := d.groupChannelsByIndexSearcher(channels)
	if err != nil {
		return nil, err
	}

	// coming position iterators by their index
	posIters := d.combinePositionIteratorsByIndex(indexes, positionIters)

	// convert combined position iterators to time iterators
	timeIters := d.buildTimeIterators(posIters)

	// combine time iterators into a single time iterator
	timeIter := d.combineTimeIterators(timeIters)

	// set the time range on the combined time iterator
	timeIter.SetBounds(tr)

	reader := d.storage.NewReader()

	return &streamIterator{mdIter: timeIter, reader: reader}, nil
}

// buildPositionIters opens a position iterator for each provided channel.
func (d *db) buildPositionIters(channels []Channel) map[Channel]core.PositionIterator {
	iters := make(map[Channel]core.PositionIterator, len(channels))
	for _, ch := range channels {
		iters[ch] = d.kv.NewIterator(ch)
	}
	return iters
}

// combinePositionIteratorsByIndex take sa set of grouped indexes and position iterators
// and creates combined position iterators for each index.
func (d *db) combinePositionIteratorsByIndex(
	indexes map[index.Searcher][]Channel,
	iters map[Channel]core.PositionIterator,
) map[index.Searcher]core.PositionIterator {
	combined := make(map[index.Searcher]core.PositionIterator, len(indexes))
	for idx, chs := range indexes {
		var _iters []core.PositionIterator
		for _, ch := range chs {
			_iters = append(_iters, iters[ch])
		}
		combined[idx] = core.NewCompoundPositionIterator(_iters...)
	}
	return combined
}

// buildTimeIterators takes a set of position iterators and their corresponding search
// index, and creates a time iterator for each.
func (d *db) buildTimeIterators(
	posIters map[index.Searcher]core.PositionIterator,
) []core.TimeIterator {
	iters := make([]core.TimeIterator, 0, len(posIters))
	for idx, posIter := range posIters {
		iters = append(iters, index.WrapPositionIter(posIter, idx))
	}
	return iters
}

// combineTimeIterators takes a set of time iterators and combines them into a single
// time iterator.
func (d *db) combineTimeIterators(iters []core.TimeIterator) core.TimeIterator {
	return core.NewCompoundTimeITerator(iters...)
}
