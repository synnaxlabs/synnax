package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/legindex"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/storage"
)

type indexRegistry struct {
	channels core.ChannelReader
	indexes  map[ChannelKey]legindex.Index
	storage  *storage.Storage
	kvDB     *kv.DB
}

func (i *indexRegistry) newSearcher(idxKey ChannelKey) (legindex.Searcher, error) {
	var idx legindex.CompoundSearcher

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
	readIdx := &legindex.Reader{Reader: i.storage.NewReader(), Iter: readIter, ChannelKey: idxKey}
	idx = append(idx, readIdx)

	return idx, nil
}

func (i *indexRegistry) wrapMDBatch(key ChannelKey, kvb core.MDBatch) (*legindex.Batch, error) {
	var searchIdx legindex.CompoundSearcher
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
	readIdx := &legindex.Reader{Reader: i.storage.NewReader(), Iter: readIter, ChannelKey: key}
	searchIdx = append(searchIdx, readIdx)

	return &legindex.Batch{
		WrappedWriter:   memIdx,
		WrappedSearcher: searchIdx,
		Buf:             &legindex.ThresholdBuffer{},
	}, nil
}
