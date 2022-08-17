package segment

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/core"
	"github.com/arya-analytics/x/telem"
	"io"
)

type Sugared struct {
	segment Segment
	header  Header
	bound   telem.TimeRange
	channel channel.Channel
}

// UnboundedSize returns the size of the underlying segment's data in bytes. This value is not restricted
// by any bounds set.
func (s Sugared) UnboundedSize() telem.Size {
	if s.segment.Data != nil {
		return telem.Size(len(s.segment.Data))
	}
	return s.header.Size
}

// UnboundedSpan returns the TimeSpan of the underlying segment. This value is not restricted by any bounds set.
func (s Sugared) UnboundedSpan() telem.TimeSpan {
	return s.channel.DataRate.SizeSpan(s.UnboundedSize(), s.channel.DataType)
}

// UnboundedRange returns the time range of the underlying segment. This value is not restricted by any bounds set.
func (s Sugared) UnboundedRange() telem.TimeRange {
	return s.Start().SpanRange(s.UnboundedSpan())
}

func (s Sugared) Start() telem.TimeStamp {
	if s.segment.Start != 0 {
		return s.segment.Start
	}
	return s.header.Start
}

// FileKey returns the key of the file the segment belongs to.
func (s Sugared) FileKey() core.FileKey { return s.header.FileKey }

func (s *Sugared) SetFileKey(fk core.FileKey) { s.header.FileKey = fk }

// ChannelKey returns the key of the channel the segment belongs to.
func (s Sugared) ChannelKey() channel.Key {
	if s.channel.Key != 0 {
		return s.channel.Key
	}
	if s.header.ChannelKey != 0 {
		return s.header.ChannelKey
	}
	return s.segment.ChannelKey
}

// BoundedRange returns the TimeRange of the underlying segment after being restricted by the bounds set in
// SetBounds.
func (s Sugared) BoundedRange() telem.TimeRange { return s.UnboundedRange().BoundBy(s.bound) }

// BoundedSpan returns the TimeSpan of the underlying segment after being restricted by the bounds set in
func (s Sugared) BoundedSpan() telem.TimeSpan { return s.BoundedRange().Span() }

// BoundedSize returns the size of the underlying segment's data in bytes after being restricted by the bounds set in
// SetBounds.
func (s Sugared) BoundedSize() telem.Size {
	return s.BoundedSpan().ByteSize(s.channel.DataRate, s.channel.DataType)
}

// BoundedOffset returns the file offset of the underlying segment after being restricted by the bounds set in
// SetBounds.
func (s Sugared) BoundedOffset() telem.Offset {
	return telem.TimeSpan(s.BoundedRange().Start-s.Start()).ByteSize(s.channel.
		DataRate, s.channel.DataType)
}

// ReadDataFrom reads data from the reader into the underlying segment.
func (s *Sugared) ReadDataFrom(r io.ReaderAt) error {
	s.segment.Data = make([]byte, s.BoundedSize())
	s.segment.Start = s.BoundedRange().Start
	_, err := r.ReadAt(s.segment.Data, int64(s.BoundedOffset()))
	return err
}

// WriteDataTo writes the segment's data to the writer.
func (s *Sugared) WriteDataTo(w io.WriteSeeker) error {
	off, err := w.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	s.header.Offset = telem.Offset(off)
	_, err = w.Write(s.segment.Data)
	return err
}

// SetBounds restricts the segment to a particular time range. This is particularly useful for enabling partial segment
// reads.
func (s *Sugared) SetBounds(bounds telem.TimeRange) { s.bound = bounds }

// Header returns the 'header' metadata for the segment.
func (s *Sugared) Header() Header { s.copyAttributesToHeader(); return s.header }

// Segment returns the underlying segment for the Sugared segment.
func (s *Sugared) Segment() Segment { s.copyAttributesToSegment(); return s.segment }

func (s *Sugared) copyAttributesToHeader() {
	if s.header.ChannelKey == 0 {
		if s.channel.Key != 0 {
			s.header.ChannelKey = s.channel.Key
		} else if s.segment.ChannelKey != 0 {
			s.header.ChannelKey = s.segment.ChannelKey
		}
	}
	s.header.Start = s.Start()
	s.header.Size = telem.Size(len(s.segment.Data))
}

func (s *Sugared) copyAttributesToSegment() {
	if s.segment.ChannelKey == 0 {
		if s.channel.Key != 0 {
			s.segment.ChannelKey = s.channel.Key
		}
		s.segment.ChannelKey = s.header.ChannelKey
	}
	s.segment.Start = s.Start()
}
