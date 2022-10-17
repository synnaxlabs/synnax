package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/kv"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	kvx "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/lock"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

var (
	// NotFound is returned when a channel or a range of data cannot be found in the DB.
	NotFound = query.NotFound
	// UniqueViolation is returned when a provided channel key already exists in the DB.
	UniqueViolation = errors.Wrap(query.UniqueViolation, "[cesium] - channel key already exists")
	// ErrChannelLocked is returned when a channel has been locked for writing by another
	// goroutine.
	ErrChannelLocked = errors.Wrap(lock.ErrLocked, "[cesium] - channel locked for writing")
)

type Segment = segment.Segment

// DB provides a persistent, concurrent store for reading and writing regular time-series
// data.
//
// Key DB works with two data types: Channels and segments. Key Channel is a named collection
// samples across a time range. Key Channel typically represents a single data source, such
// as a physical sensor, software sensor, metric, event, or other entity that
// emits regular, consistent, and time-ordered values. Key Channel has a pre-defined:
// data rate in Hz, which is the number of samples recorded per second. This data rate is
// fixed, and cannot be changed after the Channel has been created. Key Channel must also have
// a pre-defined density, which is the number of bytes occupied by each sample. Using these
// two properties, a DB can calculate the time stamp of any sample on disk without needing
// to store a timestamp or delta.
//
// Key Channel's data is partitioned into entities called segments, which are reasonably sized
// sub-ranges of a channel's data. Key Segment is defined by a start time, channel key,
// and binary data. Key segment's start time is the timestamp for the first sample in the segment.
// Segments must be written in time-order (append only), and cannot be modified once written,
// although it is possible to leave gaps between the end of one segment and the start of
// another.
//
// Key DB is safe for concurrent read and write use, although it is not possible to write
// data to a single channel concurrently. When writing data to a channel, the DB will
// acquire an exclusive lock for the duration of the request. If another goroutine
// attempts to write to the channel, a DB will return ErrChannelLocked.
type DB interface {
	// Read returns all segments in the provided time range for the given Channels.
	// The segments are returned in time-order on a per-channel basis.
	Read(tr telem.TimeRange, keys ...ChannelKey) ([]Segment, error)
	// NewIterator returns a new, unpositioned kvPositionIterator over the given time range
	// for the provided Channels. The iterator will be invalid until a positioning
	// call (First, Last, SeekFirst, SeekLE, SeekGE) is made.
	//
	// The provided iterator will NOT acknowledge any future mutations to the underlying
	// data, and provides a consistent snapshot of the data at the time of the call.
	//
	// An kvPositionIterator must be closed when it is no longer needed, or resource leaks may occur.
	NewIterator(tr telem.TimeRange, keys ...ChannelKey) (Iterator, error)
	// NewStreamIterator returns a new, unpositioned StreamIterator over the given time
	// range for the provided Channels. The iterator will be invalid until a positioning
	// request is issued (IterFirst, IterLast, IterSeekFirst, IterSeekLE, IterSeekGE).
	// The iterator must be closed, either by closing the Inlet or by cancelling
	// the Flow context.
	NewStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (StreamIterator, error)
	// Write writes the provided segments to the DB. Segments must meet the following
	// requirements:
	//
	//  1. They must be provided in time-order.
	//  2. Channel keys must be defined and exist in the database.
	//  3. SData must be valid i.e. it must have non-zero length and be a multiple of the
	//  channel's density.
	//
	// If any segments do not meet these requirements, no data will be written and the DB
	// will return a validation err. If another goroutine is currently writing to one
	// of the specified Channels, DB will return ErrChannelLocked.
	Write(segments []Segment) error
	// NewWriter returns a new Writer for the provided Channels, acquiring an exclusive
	// lock for the duration of the Writer's usage. The Writer must be closed by calling
	// Close in order to release the lock.
	NewWriter(keys ...ChannelKey) (Writer, error)
	// NewStreamWriter returns a new StreamWriter for the provided Channels, acquiring an
	// exclusive lock for the duration of the Writer's usage. The StreamWriter must be
	// closed in order to release the lock. This can be done by closing the Inlet or by
	// cancelling the Flow context.
	NewStreamWriter(keys ...ChannelKey) (StreamWriter, error)
	// CreateChannel creates a new channel in the DB. The provided channel must have a
	// positive data rate and density. The caller can provide an optional uint16 segmentKey
	// for the channel. If the key is not provided, the DB will automatically generate a
	// key. If a key is provided, the DB will validate that it is unique.
	CreateChannel(ch *Channel) error
	// RetrieveChannel retrieves Channels from the DB by their key. Returns a query.NotFound
	// err if any of the Channels cannot be found.
	RetrieveChannel(keys ...ChannelKey) ([]Channel, error)
	// Close closes persists all pending data to disk and closes the DB. Close is not
	// safe to call concurrently with any other DB methods.
	Close() error
}

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
)

type db struct {
	kv             *kv.DB
	externalKV     bool
	wg             signal.WaitGroup
	shutdown       context.CancelFunc
	channelCounter *kvx.PersistedCounter
	channelLock    lock.Keys[ChannelKey]
	logger         *zap.Logger
	indexes        *IndexingEngine
	channelSvc     core.ChannelService
	allocator      allocate.Allocator[ChannelKey, file.Key]
	storage        *storage.Storage
}

// Write implements DB.
func (d *db) Write(segments []Segment) error {
	keys := lo.Map(segments, func(s Segment, _ int) ChannelKey { return s.ChannelKey })
	w, err := d.NewWriter(keys...)
	if err != nil {
		return err
	}
	w.Write(segments)
	return w.Close()
}

// NewWriter implements DB.
func (d *db) NewWriter(keys ...ChannelKey) (Writer, error) {
	wrapped, err := d.newStreamWriter(keys)
	if err != nil {
		return nil, err
	}
	return wrapStreamWriter(wrapped), nil
}

// NewStreamWriter implements DB.
func (d *db) NewStreamWriter(keys ...ChannelKey) (StreamWriter, error) {
	return d.newStreamWriter(keys)
}

// Read implements DB.
func (d *db) Read(tr telem.TimeRange, keys ...ChannelKey) ([]Segment, error) {
	iter, err := d.NewIterator(tr, keys...)
	if err != nil {
		return nil, err
	}
	var segments []Segment
	for iter.SeekFirst(); iter.Next(100 * telem.Second); {
		segments = append(segments, iter.Value()...)
	}
	return segments, iter.Close()
}

// NewIterator implements DB.
func (d *db) NewIterator(tr telem.TimeRange, keys ...ChannelKey) (Iterator, error) {
	wrapped, err := d.newStreamIterator(tr, keys...)
	if err != nil {
		return nil, err
	}
	return wrapStreamIterator(wrapped), nil
}

func (d *db) NewStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (StreamIterator, error) {
	return d.newStreamIterator(tr, keys...)
}

// NewStreamIterator implements DB.
func (d *db) newStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (*streamIterator, error) {
	// first thing we need to do is retrieve all the channels we're going to be reading
	channels, err := d.RetrieveChannel(keys...)
	if err != nil {
		return nil, err
	}

	// now we need to construct our non-rate indexes
	nonRateIndexes := make(map[channel.Key]index.Searcher)
	nonRateChannels := make(map[channel.Key][]channel.Channel)
	rateChannels := make(map[telem.Rate][]channel.Channel)
	for _, ch := range channels {
		if ch.Index != 0 {
			_, ok := nonRateIndexes[ch.Key]
			if !ok {
				nonRateIndexes[ch.Key], err = d.indexes.AcquireSearcher(ch.Key)
				if err != nil {
					return nil, err
				}
			}
			nonRateChannels[ch.Key] = append(nonRateChannels[ch.Key], ch)
		} else {
			rateChannels[ch.Rate] = append(rateChannels[ch.Rate], ch)
		}
	}

	// no we need to construct our index iterators
	nonRatePositionIters := make(map[channel.Key]core.MDPositionIterator)
	for _, group := range nonRateChannels {
		var groupIters []core.MDPositionIterator
		for _, ch := range group {
			iter, err := d.kv.NewIterator(ch.Key)
			if err != nil {
				return nil, err
			}
			groupIters = append(groupIters, iter)
		}
		nonRatePositionIters[group[0].Key] = core.NewCompoundMDPositionIterator(groupIters...)
	}
	ratePositionIters := make(map[telem.Rate]core.MDPositionIterator)
	for _, group := range rateChannels {
		var groupIters []core.MDPositionIterator
		for _, ch := range group {
			iter, err := d.kv.NewIterator(ch.Key)
			if err != nil {
				return nil, err
			}
			groupIters = append(groupIters, iter)
		}
		ratePositionIters[group[0].Rate] = core.NewCompoundMDPositionIterator(groupIters...)
	}

	indexIters := make([]core.MDStampIterator, len(ratePositionIters)+len(nonRatePositionIters))
	for k, iter := range nonRatePositionIters {
		idx := nonRateIndexes[k]
		indexIters = append(
			indexIters,
			index.WrapMDIter(iter, idx, idx),
		)
	}
	for r, iter := range ratePositionIters {
		idx := index.RateSearcher(r)
		indexIters = append(
			indexIters,
			index.WrapMDIter(iter, idx, idx),
		)
	}

	wrappedIter := core.NewCompoundMDStampIterator(indexIters...)

	wrappedIter.SetBounds(tr)

	reader := storage.NewReader[Segment](d.storage)
	requests := confluence.NewStream[Segment](100)
	responses := confluence.NewStream[storage.ReadResponse[Segment]](100)
	reader.InFrom(requests)
	reader.OutTo(responses)
	return &streamIterator{
		mdIter:  wrappedIter,
		readReq: requests,
		readRes: responses,
	}, nil

}

// CreateChannel implements DB.
func (d *db) CreateChannel(ch *Channel) error {
	if ch.Index != 0 {
		found, err := d.channelSvc.Exists(ch.Index)
		if err != nil {
			return err
		}
		if !found {
			return errors.Wrapf(validate.ValidationError, "[cesium] - channel index %d does not exist", ch.Index)
		}
		ch.Rate = 1e9 * telem.Hz
	}
	if ch.Rate <= 0 {
		return errors.Wrap(
			validate.ValidationError,
			"[cesium] - channel data rate must be positive",
		)
	}
	if ch.Density == 0 {
		return errors.Wrap(
			validate.ValidationError,
			"[cesium] - channel density cannot be zero",
		)
	}
	if ch.Key != 0 {
		exists, err := d.channelSvc.Exists(ch.Key)
		if err != nil {
			return err
		}
		if exists {
			return UniqueViolation
		}
	} else {
		key, err := d.channelCounter.Add()
		if err != nil {
			return err
		}
		ch.Key = ChannelKey(key)
	}
	return d.channelSvc.Set(*ch)
}

// RetrieveChannel implements DB.
func (d *db) RetrieveChannel(keys ...ChannelKey) ([]Channel, error) {
	return d.channelSvc.GetMany(keys...)
}

// Close implements DB.
func (d *db) Close() error {
	d.shutdown()
	err := d.wg.Wait()
	if !d.externalKV {
		if kvErr := d.kv.Close(); kvErr != nil {
			return kvErr
		}
	}
	if err != context.Canceled {
		return err
	}
	return nil
}

func (d *db) newStreamWriter(keys []ChannelKey) (StreamWriter, error) {
	// first thing we need to do is retrieve all the Channels.
	channels, err := d.RetrieveChannel(keys...)
	if err != nil {
		return nil, err
	}

	// now we need to acquire a lock on all the Channels.
	if !d.channelLock.TryLock(keys...) {
		return nil, ErrChannelLocked
	}

	// now we need to check if there are any nonRateIndexes we need to maintain.
	writeIndexes := make(map[channel.Key]index.Writer)
	for _, ch := range channels {
		if ch.IsIndex {
			writeIndexes[ch.Key], err = d.indexes.AcquireWriter(ch.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	haveWriteIndexes := len(writeIndexes) > 0

	// now we need to construct our non-rate nonRateIndexes.
	nonRateIndexes := make(map[channel.Key]index.Searcher)
	for _, ch := range channels {
		if ch.Index != 0 {
			_, ok := nonRateIndexes[ch.Index]
			if !ok {
				nonRateIndexes[ch.Index], err = d.indexes.AcquireSearcher(ch.Index)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	// now we need to construct our index map
	searchIndexes := make(map[channel.Key]index.Searcher)
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
	ac := NewAllocator(d.allocator)
	plumber.SetSegment[WriteRequest, segment.Segment](pipe, "allocator", ac)

	// then we need to align our segments with the root index.
	ia := NewIndexAligner(searchIndexes)
	plumber.SetSegment[segment.Segment, segment.Segment](pipe, "indexAligner", ia)

	var routeIndexAlignerTo address.Address = "storage"
	if haveWriteIndexes {
		routeIndexAlignerTo = "indexFilter"
		// we need to route our segments to the maintainer conditionally
		indexFilter := NewIndexFilter(makeChannelMap(channels))
		plumber.SetSegment[segment.Segment, segment.Segment](pipe, "indexFilter", indexFilter)

		// we need to maintain our non-rate indexes.
		maintainer := NewIndexMaintainer(writeIndexes)
		plumber.SetSegment[segment.Segment, segment.Segment](pipe, "maintainer", maintainer)
	}

	// now we need to route our segments to be written to storage.
	sw := storage.NewWriter[segment.Segment](d.storage)
	plumber.SetSegment[segment.Segment, storage.WriteResponse[segment.Segment]](pipe, "sw", sw)

	// then we need to write our segment metadata to the index.
	mdw := NewMDWriter(d.kv.NewWriter())
	plumber.SetSegment[storage.WriteResponse[segment.Segment], WriteResponse](pipe, "mdw", mdw)

	// now it's time to connect everything together.
	seg := &plumber.Segment[WriteRequest, WriteResponse]{}
	lo.Must0(seg.RouteInletTo("allocator"))
	lo.Must0(seg.RouteOutletFrom("mdw"))

	plumber.UnaryRouter[segment.Segment]{
		SourceTarget: "allocator",
		SinkTarget:   "indexAligner",
	}.MustRoute(pipe)

	plumber.UnaryRouter[segment.Segment]{
		SourceTarget: "indexAligner",
		SinkTarget:   routeIndexAlignerTo,
	}.MustRoute(pipe)

	if haveWriteIndexes {
		plumber.UnaryRouter[segment.Segment]{
			SourceTarget: "indexFilter",
			SinkTarget:   "maintainer",
		}.MustRoute(pipe)
		plumber.MultiRouter[segment.Segment]{
			SourceTargets: []address.Address{"maintainer", "indexFilter"},
			SinkTargets:   []address.Address{"storage"},
			Stitch:        plumber.StitchUnary,
		}.MustRoute(pipe)
	}

	plumber.UnaryRouter[storage.WriteResponse[segment.Segment]]{
		SourceTarget: "storage",
		SinkTarget:   "mdw",
	}.MustRoute(pipe)

	return seg, nil
}

func makeChannelMap(channels []Channel) map[ChannelKey]Channel {
	m := make(map[ChannelKey]Channel, len(channels))
	for _, ch := range channels {
		m[ch.Key] = ch
	}
	return m
}
