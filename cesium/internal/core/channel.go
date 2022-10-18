package core

import (
	"github.com/cockroachdb/pebble"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/telem"
)

// ChannelKey is a unique uint16 identifier for a channel.
type ChannelKey uint16

// Channel is a logical collection of telemetry samples across a time-range. The data
// within a channel typically arrives from a single source. This can be a physical sensor,
// metric, event, or other entity that emits regular, consistent, and time-order values.
// A channel can also be used for storing derived data, such as a moving average or signal
// processing result.
type Channel struct {
	// Key is a unique identifier for the channel within a cesium.DB. If not set when
	// creating a channel, a unique key will be generated.
	Key ChannelKey
	// Index is the channel used to index the channel's values. The Index is used to
	// associate a value with a timestamp. If zero, the channel's data will be indexed
	// using its rate. One of Index or Rate must be non-zero.
	Index ChannelKey
	// IsIndex is set to true if the channel is an index channel. Index channels must
	// be int64 values written in ascending order. Index channels are most commonly
	// unix nanosecond timestamps.
	IsIndex bool
	// Rate sets the rate at which the channels values are written. This is used to
	// determine the timestamp of each sample.
	Rate telem.Rate
	// Density is the number of bytes per sample. All values in a channel must be have
	// the same density.
	Density telem.Density
}

// GorpKey implements the gorp.Entry interface.
func (c Channel) GorpKey() ChannelKey { return c.Key }

// SetOptions implements the gorp.Entry interface.
func (c Channel) SetOptions() []interface{} { return []interface{}{pebble.NoSync} }

// ChannelKeys returns the keys of the given channels.
func ChannelKeys(channels []Channel) []ChannelKey {
	return lo.Map(channels, func(c Channel, i int) ChannelKey { return c.Key })
}

// ChannelReader is a readable channel store.
type ChannelReader interface {
	// GetChannel retrieves a channel by its key. Returns a query.NotFound error if the
	// channel does not exist.
	GetChannel(key ChannelKey) (Channel, error)
	// GetChannels retrieves multiple channels by their keys. Returns a query.NotFound error if
	// any of the channels do not exist.
	GetChannels(keys ...ChannelKey) ([]Channel, error)
	// ChannelsExist checks if the channels with the given keys exist. Returns false
	// if any of the channels are not found.
	ChannelsExist(keys ...ChannelKey) (bool, error)
}

// ChannelWriter is a writable channel store.
type ChannelWriter interface {
	// SetChannel sets a channel. Overwrites the channel if it already exists.
	SetChannel(c Channel) error
	// SetChannels sets multiple channels. Overwrites the channels if they already exist.
	SetChannels(chs ...Channel) error
	// DeleteChannel deletes a channel by its key. Does nothing if the channel does not
	// exist.
	DeleteChannel(key ChannelKey) error
	// DeleteChannels deletes multiple channels by their keys. Does nothing if the channels
	//do not exist.
	DeleteChannels(keys ...ChannelKey) error

	// NextChannelKey returns the next available channel key.
	NextChannelKey() (ChannelKey, error)
}

type ChannelEngine interface {
	ChannelReader
	ChannelWriter
}
