package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/storage"
)

type indexRegistry struct {
	channels core.ChannelReader
	indexes  map[ChannelKey]index.Index
	storage  *storage.Storage
	kvDB     *kv.DB
}

func (i *indexRegistry) newSearcher(idxKey ChannelKey) (index.Searcher, error) {
	var idx index.CompoundSearcher

	memIdx, ok := i.indexes[idxKey]
	if !ok {
		panic("mem idx not found")
	}
	idx = append(idx, memIdx)

	ch, err := i.channels.GetChannel(idxKey)
	if err != nil {
		return nil, err
	}

	readIter := i.kvDB.NewIterator(ch)
	readIdx := &index.Reader{Reader: i.storage.NewReader(), Iter: readIter, ChannelKey: idxKey}
	idx = append(idx, readIdx)

	return idx, nil
}

func (i *indexRegistry) wrapMDBatch(key ChannelKey, kvb core.MDBatch) (*index.Batch, error) {
	var searchIdx index.CompoundSearcher
	memIdx, ok := i.indexes[key]
	if !ok {
		panic("mem idx not found")
	}
	searchIdx = append(searchIdx, memIdx)
	ch, err := i.channels.GetChannel(key)
	if err != nil {
		return nil, err
	}
	readIter := kvb.NewIterator(ch)
	readIdx := &index.Reader{Reader: i.storage.NewReader(), Iter: readIter, ChannelKey: key}
	searchIdx = append(searchIdx, readIdx)

	return &index.Batch{
		WrappedWriter:   memIdx,
		WrappedSearcher: searchIdx,
		Buf:             &index.ThresholdBuffer{},
	}, nil
}
