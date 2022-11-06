package cesium

import (
	"github.com/synnaxlabs/cesium/internal/index"
)

func (d *db) newStreamWriter(keys []ChannelKey) (StreamWriter, error) {
	channels, err := d.RetrieveChannels(keys...)
	if err != nil {
		return nil, err
	}

	if !d.channelLock.TryLock(keys...) {
		return nil, ErrChannelLocked
	}

	// Check whether we need to maintain any indexes.
	writeIndexes := make(map[ChannelKey]index.Writer)
	for _, ch := range channels {
		if ch.IsIndex {
			writeIndexes[ch.Key], err = d.indexes.acquireWriter(ch.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	w := &streamWriter{
		kv:              d.kv,
		kvWriter:        d.kv.NewWriter(),
		alloc:           d.allocator,
		storageWriter:   d.storage.NewWriter(),
		rateAligners:    make(map[ChannelKey]*rateAligner),
		indexAligners:   make(map[ChannelKey]*indexAligner),
		indexedAligners: make(map[ChannelKey]*indexedAligner),
	}

	searchIndexes, err := d.groupIndexesByChannelKey(channels)
	if err != nil {
		return nil, err
	}

	for _, ch := range channels {
		if ch.IsIndex {
			w.indexAligners[ch.Key] = &indexAligner{
				hwm:      0,
				channel:  ch,
				buffer:   &index.Buffered{Wrapped: writeIndexes[ch.Key]},
				searcher: searchIndexes[ch.Key],
			}
		}
	}
	for _, ch := range channels {
		if ch.Index != 0 {
			buf, ok := w.indexAligners[ch.Index]
			searcher := searchIndexes[ch.Key]
			if ok {
				searcher = index.CompoundSearcher{buf.buffer, searcher}
			}
			idxCh, err := d.RetrieveChannel(ch.Index)
			if err != nil {
				return nil, err
			}
			w.indexedAligners[ch.Key] = &indexedAligner{
				hwm:      0,
				channel:  ch,
				searcher: searcher,
				iter:     w.kvWriter.NewIterator(idxCh),
			}
		} else if !ch.IsIndex {
			w.rateAligners[ch.Key] = &rateAligner{
				channel:  ch,
				hwm:      0,
				searcher: searchIndexes[ch.Key],
			}
		}
	}

	return w, nil
}
