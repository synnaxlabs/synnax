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
	positionIters := d.buildPositionIters(channels)
	if err != nil {
		return nil, err
	}
	indexes, err := d.groupChannelsByIndexSearcher(channels)
	if err != nil {
		return nil, err
	}
	posIters := d.combinePositionIteratorsByIndex(indexes, positionIters)
	timeIters := d.buildTimeIterators(posIters)
	timeIter := d.combineTimeIterators(timeIters)
	timeIter.SetBounds(tr)
	reader := d.storage.NewReader()
	return &streamIterator{mdIter: timeIter, reader: reader}, nil
}

// buildPositionIters opens a position iterator for each provided ch.
func (d *db) buildPositionIters(channels []Channel) map[Channel]core.PositionIterator {
	iters := make(map[Channel]core.PositionIterator, len(channels))
	for _, ch := range channels {
		iters[ch] = d.kv.NewIterator(ch)
	}
	return iters
}

type searchPosIterPair struct {
	idx  index.Searcher
	iter core.PositionIterator
}

// combinePositionIteratorsByIndex take sa set of grouped indexes and position iterators
// and creates combined position iterators for each index.
func (d *db) combinePositionIteratorsByIndex(
	indexes []searcherChannels,
	iters map[Channel]core.PositionIterator,
) []searchPosIterPair {
	combined := make([]searchPosIterPair, len(indexes))
	for i, pair := range indexes {
		var _iters []core.PositionIterator
		for _, ch := range pair.channels {
			_iters = append(_iters, iters[ch])
		}
		combined[i] = searchPosIterPair{idx: pair.idx, iter: core.NewCompoundPositionIterator(_iters...)}
	}
	return combined
}

// buildTimeIterators takes a set of position iterators and their corresponding search
// index, and creates a time iterator for each.
func (d *db) buildTimeIterators(posIters []searchPosIterPair) []core.TimeIterator {
	iters := make([]core.TimeIterator, 0, len(posIters))
	for _, pair := range posIters {
		iters = append(iters, index.WrapPositionIter(pair.iter, pair.idx, d.logger))
	}
	return iters
}

// combineTimeIterators takes a set of time iterators and combines them into a single
// time iterator.
func (d *db) combineTimeIterators(iters []core.TimeIterator) core.TimeIterator {
	return core.NewCompoundTimeITerator(iters...)
}
