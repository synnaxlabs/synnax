package cesium

import (
	"context"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/segment"
	"github.com/arya-analytics/x/confluence"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/lock"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/signal"
	"github.com/arya-analytics/x/telem"
	"github.com/arya-analytics/x/validate"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

var (
	// NotFound is returned when a channel or a range of data cannot be found in the DB.
	NotFound = query.NotFound
	// UniqueViolation is returned when a provided channel key already exists in the DB.
	UniqueViolation = query.UniqueViolation
	// ErrChannelLocked is returned when a channel has been locked for writing by another
	// goroutine.
	ErrChannelLocked = errors.Wrap(lock.ErrLocked, "[cesium] - channel locked for writing")
	// RangeHasNoData is returned when a range of data is requested but no data exists in the
	// range.
	RangeHasNoData = kv.RangeHasNoData
)

// DB provides a persistent, concurrent store for reading and writing regular time-series
// data.
//
// A DB works with two data types: channels and segments. A Channel is a named collection
// samples across a time range. A Channel typically represents a single data source, such
// as a physical sensor, software sensor, metric, event, or other entity that
// emits regular, consistent, and time-ordered values. A Channel has a pre-defined:
// data rate in Hz, which is the number of samples recorded per second. This data rate is
// fixed, and cannot be changed after the Channel has been created. A Channel must also have
// a pre-defined density, which is the number of bytes occupied by each sample. Using these
// two properties, a DB can calculate the time stamp of any sample on disk without needing
// to store a timestamp or delta.
//
// A Channel's data is partitioned into entities called segments, which are reasonably sized
// sub-ranges of a channel's data. A Segment is defined by a start time, channel key,
// and binary data. A segment's start time is the timestamp for the first sample in the segment.
// Segments must be written in time-order (append only), and cannot be modified once written,
// although it is possible to leave gaps between the end of one segment and the start of
// another.
//
// A DB is safe for concurrent read and write use, although it is not possible to write
// data to a single channel concurrently. When writing data to a channel, the DB will
// acquire an exclusive lock for the duration of the request. If another goroutine
// attempts to write to the channel, a DB will return ErrChannelLocked.
type DB interface {
	// Read returns all segments in the provided time range for the given channels.
	// The segments are returned in time-order on a per-channel basis.
	Read(tr telem.TimeRange, keys ...ChannelKey) ([]Segment, error)
	// NewIterator returns a new, unpositioned Iterator over the given time range
	// for the provided channels. The iterator will be invalid until a positioning
	// call (First, Last, SeekFirst, SeekLE, SeekGE) is made.
	//
	// The provided iterator will NOT acknowledge any future mutations to the underlying
	// data, and provides a consistent snapshot of the data at the time of the call.
	//
	// An Iterator must be closed when it is no longer needed, or resource leaks may occur.
	NewIterator(tr telem.TimeRange, keys ...ChannelKey) (Iterator, error)
	// NewStreamIterator returns a new, unpositioned StreamIterator over the given time
	// range for the provided channels. The iterator will be invalid until a positioning
	// request is issued (IterFirst, IterLast, IterSeekFirst, IterSeekLT, IterSeekGE).
	// The iterator must be closed, either by closing the Inlet or by cancelling
	// the Flow context.
	NewStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (StreamIterator, error)
	// Write writes the provided segments to the DB. Segments must meet the following
	// requirements:
	//
	//  1. They must be provided in time-order.
	//  2. Channel keys must be defined and exist in the database.
	//  3. Data must be valid i.e. it must have non-zero length and be a multiple of the
	//  channel's density.
	//
	// If any segments do not meet these requirements, no data will be written and the DB
	// will return a validation error. If another goroutine is currently writing to one
	// of the specified channels, DB will return ErrChannelLocked.
	Write(segments []Segment) error
	// NewWriter returns a new Writer for the provided channels, acquiring an exclusive
	// lock for the duration of the Writer's usage. The Writer must be closed by calling
	// Close in order to release the lock.
	NewWriter(keys ...ChannelKey) (Writer, error)
	// NewStreamWriter returns a new StreamWriter for the provided channels, acquiring an
	// exclusive lock for the duration of the Writer's usage. The StreamWriter must be
	// closed in order to release the lock. This can be done by closing the Inlet or by
	// cancelling the Flow context.
	NewStreamWriter(keys ...ChannelKey) (StreamWriter, error)
	// CreateChannel creates a new channel in the DB. The provided channel must have a
	// positive data rate and density. The caller can provide an optional uint16 Key
	// for the channel. If the key is not provided, the DB will automatically generate a
	// key. If a key is provided, the DB will validate that it is unique.
	CreateChannel(ch *Channel) error
	// RetrieveChannel retrieves channels from the DB by their key. Returns a query.NotFound
	// error if any of the channels cannot be found.
	RetrieveChannel(keys ...ChannelKey) ([]Channel, error)
	// Close closes persists all pending data to disk and closes the DB. Close is not
	// safe to call concurrently with any other DB methods.
	Close() error
}

type (
	Channel    = channel.Channel
	ChannelKey = channel.Key
	Segment    = segment.Segment
)

type db struct {
	kv                 kvx.DB
	externalKV         bool
	wg                 signal.WaitGroup
	shutdown           context.CancelFunc
	channelCounter     *kvx.PersistedCounter
	channelLock        lock.Keys[ChannelKey]
	createMetrics      createMetrics
	retrieveMetrics    retrieveMetrics
	logger             *zap.Logger
	retrieveOperations confluence.Inlet[[]retrieveOperationUnary]
	createOperations   confluence.Inlet[[]createOperationUnary]
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
	for iter.First(); iter.Valid(); iter.Next() {
		segments = append(segments, iter.Value()...)
	}
	return segments, iter.Close()
}

// NewIterator implements DB.
func (d *db) NewIterator(tr telem.TimeRange, keys ...ChannelKey) (Iterator, error) {
	wrapped, err := d.newStreamIterator(tr, keys)
	if err != nil {
		return nil, err
	}
	return wrapStreamIterator(wrapped), nil
}

// NewStreamIterator implements DB.
func (d *db) NewStreamIterator(tr telem.TimeRange, keys ...ChannelKey) (StreamIterator, error) {
	return d.newStreamIterator(tr, keys)
}

// CreateChannel implements DB.
func (d *db) CreateChannel(ch *Channel) error {
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
	svc := kv.NewChannelService(d.kv)
	if ch.Key != 0 {
		exists, err := svc.Exists(ch.Key)
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
	return svc.Set(*ch)
}

// RetrieveChannel implements DB.
func (d *db) RetrieveChannel(keys ...ChannelKey) ([]Channel, error) {
	return kv.NewChannelService(d.kv).Get(keys...)
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

func (d *db) newStreamIterator(tr telem.TimeRange, keys []ChannelKey) (*streamIterator, error) {
	return newStreamIterator(
		tr,
		keys,
		d.retrieveOperations,
		d.logger,
		d.retrieveMetrics,
		d.kv,
	)
}

func (d *db) newStreamWriter(keys []ChannelKey) (*streamWriter, error) {
	return newStreamWriter(
		keys,
		d.channelLock,
		d.kv,
		d.createMetrics,
		d.logger,
		d.createOperations,
	)
}
