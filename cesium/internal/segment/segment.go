package segment

import (
	"encoding/binary"
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/pebble"
)

type Segment struct {
	ChannelKey channel.Key
	Start      telem.TimeStamp
	Data       []byte
}

func (s Segment) Sugar(ch channel.Channel) *Sugared {
	return &Sugared{segment: s, channel: ch, bound: telem.TimeRangeMax}
}

type Header struct {
	ChannelKey channel.Key
	Start      telem.TimeStamp
	FileKey    core.FileKey
	Offset     telem.Offset
	Size       telem.Size
}

func (h Header) Sugar(ch channel.Channel) *Sugared { return &Sugared{header: h, channel: ch} }

func (h Header) Key() Key { return NewKey(h.ChannelKey, h.Start) }

// GorpKey implements the gorp.Entry interface.
func (h Header) GorpKey() []byte { return h.Key().Bytes() }

// SetOptions implements the gorp.Entry interface.
func (h Header) SetOptions() []interface{} { return []interface{}{pebble.NoSync} }

func (h Header) End(dr telem.DataRate, dt telem.DataType) telem.TimeStamp {
	return h.Start.Add(dr.SizeSpan(h.Size, dt))
}

const prefix = 's'

type Key [11]byte

func (k Key) Bytes() []byte { return k[:] }

func NewKeyPrefix(channelKey channel.Key) []byte {
	keyPrefix := make([]byte, 3)
	keyPrefix[0] = prefix
	binary.BigEndian.PutUint16(keyPrefix[1:], uint16(channelKey))
	return keyPrefix
}

func NewKey(channelKey channel.Key, stamp telem.TimeStamp) (key Key) {
	key[0] = prefix
	binary.BigEndian.PutUint16(key[1:3], uint16(channelKey))
	binary.BigEndian.PutUint64(key[3:], uint64(stamp))
	return key
}

type Range struct {
	Channel channel.Channel
	Bounds  telem.TimeRange
	Headers []Header
}

func (r *Range) Range() telem.TimeRange { return r.UnboundedRange().BoundBy(r.Bounds) }

func (r *Range) Empty() bool { return r.UnboundedRange().IsZero() }

func (r *Range) UnboundedRange() telem.TimeRange {
	if len(r.Headers) == 0 {
		return telem.TimeRangeZero
	}
	return telem.TimeRange{
		Start: r.Headers[0].Start,
		End:   r.Headers[len(r.Headers)-1].End(r.Channel.DataRate, r.Channel.DataType),
	}
}
