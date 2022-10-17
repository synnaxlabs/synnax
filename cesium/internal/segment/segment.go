package segment

import (
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/file"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/telem"
)

// MD represents the key-value metadata for a segment.
type MD struct {
	// ChannelKey tracks the channel the segment belongs to.
	ChannelKey channel.Key
	// FileKey is the key of the file where the segment's data is stored.
	FileKey file.Key
	// Alignment is the start position of the segment.
	Alignment position.Position
	// Offset is the offset of the segment's data in the file.
	Offset telem.Offset
	// Size is the size of the segment's data in the file.
	Size telem.Size
}

func (s MD) GorpKey() []byte {
	key := make([]byte, 11)
	WriteKey(s.ChannelKey, s.Alignment, key)
	return key
}

func (s MD) SetOptions() []interface{} { return []interface{}{pebble.NoSync} }

func (s MD) End(density telem.Density) position.Position {
	return s.Alignment.Add(position.Span(s.Size / telem.Size(density)))
}

func (s MD) EndOffset() telem.Offset {
	return s.Offset + s.Size
}

func (s MD) Range(density telem.Density) position.Range {
	return position.Range{
		Start: s.Alignment,
		End:   s.End(density),
	}
}

const prefix = 's'

func WriteKeyPrefix(chKey channel.Key, into []byte) {
	into[0] = prefix
	binary.BigEndian.PutUint16(into[1:], uint16(chKey))
}

func WriteKey(chKey channel.Key, pos position.Position, into []byte) {
	WriteKeyPrefix(chKey, into)
	binary.BigEndian.PutUint64(into[3:], uint64(pos))
}

type Segment struct {
	ChannelKey channel.Key
	Start      telem.TimeStamp
	Data       []byte
	MD         MD
}

var _ storage.WriteRequest = Segment{}

func (s Segment) AItem() allocate.Item[channel.Key] {
	return allocate.Item[channel.Key]{Size: s.Size(), Key: s.ChannelKey}
}

func (s Segment) Size() telem.Size {
	return telem.Size(len(s.Data))
}

// STarget implements storage.WriteRequest.
func (s Segment) STarget() file.Key { return s.MD.FileKey }

// SData implements storage.WriteRequest.
func (s Segment) SData() []byte { return s.Data }

func (s Segment) SOffset() telem.Offset { return s.MD.Offset }

func (s Segment) SSize() telem.Size { return s.MD.Size }
