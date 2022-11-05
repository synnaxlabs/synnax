package cesium

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/storage"
)

type indexingEngine struct {
	channelReader core.ChannelReader
	memSearchers  map[ChannelKey]index.Searcher
	memWriters    map[ChannelKey]index.Writer
	storage       *storage.Storage
	kvDB          *kv.DB
}

func (i *indexingEngine) acquireSearcher(idxKey ChannelKey) (index.Searcher, error) {
	var idx index.CompoundSearcher

	memIdx, ok := i.memSearchers[idxKey]
	if !ok {
		panic("mem idx not found")
	}
	idx = append(idx, memIdx)

	ch, err := i.channelReader.GetChannel(idxKey)
	if err != nil {
		return nil, err
	}

	readIter := i.kvDB.NewIterator(ch)
	readIdx := &index.Reader{Reader: i.storage.NewReader(), Iter: readIter, ChannelKey: idxKey}
	idx = append(idx, readIdx)

	return idx, nil
}

func (i *indexingEngine) acquireWriter(key ChannelKey) (index.Writer, error) {
	var idx index.CompoundWriter
	memIdx, ok := i.memWriters[key]
	if ok {
		idx = append(idx, memIdx)
	}
	return idx, nil
}
