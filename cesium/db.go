package cesium

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var (
	// ChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
	ChannelNotFound  = errors.Wrap(query.NotFound, "[cesium] - channel not found")
	ErrDiscontinuous = index.ErrDiscontinuous
)

type (
	Channel = core.Channel
	Frame   = core.Frame
)

func NewFrame(keys []string, arrays []telem.Array) Frame { return core.NewFrame(keys, arrays) }

// DB provides a persistent, concurrent store for reading and writing arrays of telemetry.
//
// A DB works with three data types: Channels, Arrays, and Frames. A Channel is a named
// collection of samples across a time range, and typically represents a single data source,
// such as a physical sensor, software sensor, metric, or event.
type DB interface {
	// CreateChannel creates the given channels in the DB.
	CreateChannel(channels ...Channel) error
	// RetrieveChannel retrieves the channel with the given key.
	RetrieveChannel(key string) (Channel, error)
	// RetrieveChannels retrieves the channels with the given keys.
	RetrieveChannels(keys ...string) ([]Channel, error)
	Write(start telem.TimeStamp, frame Frame) error
	WriteArray(start telem.TimeStamp, key string, arr telem.Array) error
	NewWriter(cfg WriterConfig) (Writer, error)
	NewStreamWriter(cfg WriterConfig) (StreamWriter, error)
	Read(tr telem.TimeRange, keys ...string) (Frame, error)
	NewIterator(cfg IteratorConfig) (Iterator, error)
	NewStreamIterator(cfg IteratorConfig) (StreamIterator, error)
	Close() error
}

type cesium struct {
	*options
	dbs map[string]unary.DB
}

// Write implements DB.
func (db *cesium) Write(start telem.TimeStamp, frame Frame) error {
	w, err := db.NewWriter(WriterConfig{Start: start, Channels: frame.Keys()})
	if err != nil {
		return err
	}
	w.Write(frame)
	w.Commit()
	return w.Close()
}

// WriteArray implements DB.
func (db *cesium) WriteArray(start telem.TimeStamp, key string, arr telem.Array) error {
	return db.Write(start, core.NewFrame([]string{key}, []telem.Array{arr}))
}

// Read implements DB.
func (db *cesium) Read(tr telem.TimeRange, keys ...string) (frame Frame, err error) {
	iter, err := db.NewIterator(IteratorConfig{Channels: keys, Bounds: tr})
	if err != nil {
		return
	}
	defer func() { err = iter.Close() }()
	if !iter.SeekFirst() {
		return
	}
	for iter.Next(telem.TimeSpanMax) {
		frame = frame.AppendFrame(iter.Value())
	}
	return
}

// Close implements DB.
func (db *cesium) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, u := range db.dbs {
		c.Exec(u.Close)
	}
	return nil
}
