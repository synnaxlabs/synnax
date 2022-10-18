package core

import (
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/cesium/internal/allocate"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

// SegmentMD represents the metadata for a segment.
type SegmentMD struct {
	// ChannelKey tracks the channel the segment belongs to.
	ChannelKey ChannelKey
	// FileKey is the key of the file where the segment's data is stored.
	FileKey FileKey
	// Alignment is the start position of the segment.
	Alignment position.Position
	// Offset is the offset of the segment's data in the file.
	Offset telem.Offset
	// Size is the size of the segment's data in the file.
	Size telem.Size
}

// GorpKey implements the gorp.Entry interface.
func (s SegmentMD) GorpKey() []byte {
	key := make([]byte, 11)
	WriteSegmentKey(s.ChannelKey, s.Alignment, key)
	return key
}

// SetOptions implements the gorp.Entry interface.
func (s SegmentMD) SetOptions() []interface{} { return []interface{}{pebble.NoSync} }

// End returns the end position of the segment given its density.
func (s SegmentMD) End(density telem.Density) position.Position {
	return s.Alignment.Add(position.Span(s.Size / telem.Size(density)))
}

// EndOffset returns the end offset of the segment.
func (s SegmentMD) EndOffset() telem.Offset { return s.Offset + s.Size }

// Range returns the position range occupied by the segment.
func (s SegmentMD) Range(density telem.Density) position.Range {
	return position.Range{Start: s.Alignment, End: s.End(density)}
}

const prefix = 's'

// WriteSegmentKeyPrefix generates and writes a segment key prefix to the given byte slice.
func WriteSegmentKeyPrefix(chKey ChannelKey, into []byte) {
	into[0] = prefix
	binary.BigEndian.PutUint16(into[1:], uint16(chKey))
}

// WriteSegmentKey writes a segment key to the given byte slice.
func WriteSegmentKey(chKey ChannelKey, pos position.Position, into []byte) {
	WriteSegmentKeyPrefix(chKey, into)
	binary.BigEndian.PutUint64(into[3:], uint64(pos))
}

// SugaredSegment is a partitioned region of a channel's data aligned to a start timestamp.
// It is used to represent a segment's metadata and data.
type SugaredSegment struct {
	// Start is the timestamp of the first sample in the segment.
	Start telem.TimeStamp
	// Data is the segment's data. It is assumed to:
	//
	// 		1. Be aligned to the start timestamp.
	// 		2. Contain no gaps.
	//		3. Contain no overlaps.
	//		4. Be sorted by timestamp.
	//		5. Have a consistent sample density.
	Data []byte
	// MD is the segment's metadata.
	SegmentMD
}

func (s SugaredSegment) AItem() allocate.Item[ChannelKey] {
	return allocate.Item[ChannelKey]{Size: s.Size(), Key: s.ChannelKey}
}

func (s SugaredSegment) Size() telem.Size {
	if s.Data == nil {
		return s.SegmentMD.Size
	}
	return telem.Size(len(s.Data))
}
