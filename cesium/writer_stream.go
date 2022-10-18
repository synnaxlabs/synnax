package cesium

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence/plumber"
)

func (d *db) newStreamWriter(keys []ChannelKey) (StreamWriter, error) {
	// first thing we need to do is retrieve all the Channels.
	channels, err := d.RetrieveChannels(keys...)
	if err != nil {
		return nil, err
	}

	// now we need to acquire a lock on all the Channels.
	if !d.channelLock.TryLock(keys...) {
		return nil, ErrChannelLocked
	}

	// now we need to check if there are any nonRateIndexes we need to maintain.
	writeIndexes := make(map[ChannelKey]index.Writer)
	for _, ch := range channels {
		if ch.IsIndex {
			writeIndexes[ch.Key], err = d.indexes.acquireWriter(ch.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	haveWriteIndexes := len(writeIndexes) > 0

	// now we need to construct our non-rate nonRateIndexes.
	nonRateIndexes := make(map[ChannelKey]index.Searcher)
	for _, ch := range channels {
		if ch.Index != 0 {
			_, ok := nonRateIndexes[ch.Index]
			if !ok {
				nonRateIndexes[ch.Index], err = d.indexes.acquireSearcher(ch.Index)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// now we need to construct our index map
	searchIndexes := make(map[ChannelKey]index.Searcher)
	for _, ch := range channels {
		if ch.Index != 0 {
			searchIndexes[ch.Key] = nonRateIndexes[ch.Index]
		} else {
			searchIndexes[ch.Key] = index.RateSearcher(ch.Rate)
		}
	}

	// and now to construct our write pipeline.
	pipe := plumber.New()

	// first we need to allocate our segments to a file.
	ac := newAllocator(d.allocator)
	plumber.SetSegment[WriteRequest, []core.SugaredSegment](pipe, "allocator", ac)

	// then we need to align our segments with the root index.
	ia := newIndexAligner(searchIndexes)
	plumber.SetSegment[[]core.SugaredSegment, []core.SugaredSegment](pipe, "indexAligner", ia)

	var routeIndexAlignerTo address.Address = "storage"
	if haveWriteIndexes {
		routeIndexAlignerTo = "indexFilter"
		// we need to route our segments to the maintainer conditionally
		indexFilter := newIndexMaintenanceRouter(d.kv.ChannelReader)
		plumber.SetSegment[[]core.SugaredSegment, []core.SugaredSegment](pipe, "indexFilter", indexFilter)

		// we need to maintain our non-rate indexes.
		maintainer := newIndexMaintainer(writeIndexes)
		plumber.SetSink[[]core.SugaredSegment](pipe, "maintainer", maintainer)
	}

	// now we need to route our segments to be written to storage.
	plumber.SetSegment[[]core.SugaredSegment, []core.SugaredSegment](
		pipe,
		"storage",
		newStorageWriter(d.storage.NewWriter()),
	)

	// then we need to write our segment metadata to the index.
	kvW, err := d.kv.NewWriter()
	if err != nil {
		return nil, err
	}
	mdw := newMDWriter(kvW, keys, d.channelLock)
	plumber.SetSegment[[]core.SugaredSegment, WriteResponse](pipe, "mdWriter", mdw)

	// now it's time to connect everything together.
	seg := &plumber.Segment[WriteRequest, WriteResponse]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("allocator"))
	lo.Must0(seg.RouteOutletFrom("mdWriter"))

	plumber.MustConnect[[]core.SugaredSegment](pipe, "allocator", "indexAligner", 1)

	plumber.UnaryRouter[[]core.SugaredSegment]{
		SourceTarget: "indexAligner",
		SinkTarget:   routeIndexAlignerTo,
	}.MustRoute(pipe)

	if haveWriteIndexes {
		plumber.UnaryRouter[[]core.SugaredSegment]{
			SourceTarget: "indexFilter",
			SinkTarget:   "maintainer",
		}.MustRoute(pipe)
		plumber.UnaryRouter[[]core.SugaredSegment]{
			SourceTarget: "indexFilter",
			SinkTarget:   "storage",
		}.MustRoute(pipe)
	}

	plumber.UnaryRouter[[]core.SugaredSegment]{
		SourceTarget: "storage",
		SinkTarget:   "mdWriter",
	}.MustRoute(pipe)

	return seg, nil
}
