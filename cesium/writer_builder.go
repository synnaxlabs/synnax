package cesium

import (
	"github.com/synnaxlabs/cesium/internal/index"
)

func (d *db) newStreamWriter(keys []ChannelKey) (StreamWriter, error) {
	channels, err := d.RetrieveChannels(keys...)
	if err != nil {
		return nil, err
	}

	locked, releaser := d.channelLock.TryLockWithReleaser(keys...)
	if !locked {
		return nil, ErrChannelLocked
	}

	mdBatch := d.kv.NewBatch()

	w := &streamWriter{
		lockReleaser:  releaser,
		mdBatch:       mdBatch,
		alloc:         d.allocator,
		storageWriter: d.storage.NewWriter(),
		idxProcessors: make(map[ChannelKey]indexProcessor, len(channels)),
	}

	batchIndexes := make(map[ChannelKey]*index.Batch)
	for _, ch := range channels {
		if ch.IsIndex {
			batchIndexes[ch.Key], err = d.indexes.wrapMDBatch(ch.Key, mdBatch)
			if err != nil {
				return nil, err
			}
			w.idxReleasers = append(w.idxReleasers, batchIndexes[ch.Key])
			w.idxProcessors[ch.Key] = &indexIndexProcessor{
				hwm:      0,
				channel:  ch,
				writer:   batchIndexes[ch.Key],
				searcher: index.RateSearcher(ch.Rate),
			}
		}
	}

	for _, ch := range channels {
		if ch.Index != 0 {
			var (
				searcher index.Searcher
				ok       bool
			)
			searcher, ok = batchIndexes[ch.Index]
			if !ok {
				searcher, err = d.indexes.newSearcher(ch.Index)
				w.idxReleasers = append(w.idxReleasers, searcher)
				if err != nil {
					return nil, err
				}
			}
			w.idxProcessors[ch.Key] = &indexedIndexProcessor{
				hwm:      0,
				ch:       ch,
				searcher: searcher,
				batch:    mdBatch,
			}
		} else if !ch.IsIndex {
			idx := index.RateSearcher(ch.Rate)
			w.idxReleasers = append(w.idxReleasers, idx)
			w.idxProcessors[ch.Key] = &rateIndexProcessor{
				hwm:      0,
				ch:       ch,
				searcher: idx,
			}
		}
	}

	return w, nil
}
