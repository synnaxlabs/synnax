// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"bytes"
	"context"
	"io"
	"math/bits"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	xhttp "github.com/synnaxlabs/x/http"
	"github.com/synnaxlabs/x/telem"
)

// TDMSEncoder encodes a read as an NI TDMS file. Channels are grouped by the index
// channel that timestamps them, and each group holds its index, then its data channels.
// The first segment declares every group and channel, and each frame then becomes one
// segment, so the file streams. Timestamps become TDMS timestamps, and UUID, bytes, and
// JSON samples become TDMS strings in their text form. Channels with no index are
// excluded. A read that fails after the first byte truncates the body.
var TDMSEncoder tdmsEncoder

type tdmsEncoder struct{}

var _ xhttp.Encoder = tdmsEncoder{}

// ContentType implements http.Encoder.
func (tdmsEncoder) ContentType() string { return "application/x-tdms" }

// Encode implements encoding.Encoder. It holds the whole read in memory, so prefer
// EncodeStream.
func (e tdmsEncoder) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := e.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeStream implements encoding.Encoder. value must be a ReadResponse.
func (tdmsEncoder) EncodeStream(_ context.Context, w io.Writer, value any) error {
	res, err := readResponse(value)
	if err != nil {
		return err
	}
	s, err := newTDMSSegmenter(res)
	if err == nil {
		err = s.writeHeader(w)
	}
	if err != nil {
		return errors.Combine(err, res.Close())
	}
	return drain(res, func(fr Frame) error { return s.writeFrame(w, fr) })
}

const (
	tdmsVersion = 4713
	// tdmsLeadInSize is the byte size of the lead-in that opens every segment.
	tdmsLeadInSize = 28
	// tdmsNoRawData is the raw data index of an object with no data in a segment.
	tdmsNoRawData uint32 = 0xFFFFFFFF
)

// Flags of a segment's table of contents.
const (
	tdmsTocMetaData   uint32 = 1 << 1
	tdmsTocNewObjList uint32 = 1 << 2
	tdmsTocRawData    uint32 = 1 << 3
)

// tdmsDataType is a TDMS data type code.
type tdmsDataType uint32

const (
	tdmsI8        tdmsDataType = 0x01
	tdmsI16       tdmsDataType = 0x02
	tdmsI32       tdmsDataType = 0x03
	tdmsI64       tdmsDataType = 0x04
	tdmsU8        tdmsDataType = 0x05
	tdmsU16       tdmsDataType = 0x06
	tdmsU32       tdmsDataType = 0x07
	tdmsU64       tdmsDataType = 0x08
	tdmsF32       tdmsDataType = 0x09
	tdmsF64       tdmsDataType = 0x0A
	tdmsString    tdmsDataType = 0x20
	tdmsBoolean   tdmsDataType = 0x21
	tdmsTimeStamp tdmsDataType = 0x44
)

// tdmsChannel is a channel of the file.
type tdmsChannel struct {
	key      channel.Key
	path     string
	dataType telem.DataType
	tdmsType tdmsDataType
}

// tdmsSegmenter writes the segments of a file, reusing its buffers across them.
type tdmsSegmenter struct {
	groups   []string
	channels []tdmsChannel
	meta     []byte
	raw      []byte
	text     []byte
}

func newTDMSSegmenter(res ReadResponse) (*tdmsSegmenter, error) {
	s := &tdmsSegmenter{}
	for _, group := range groupByIndex(res) {
		groupPath := "/" + tdmsName(group[0].Name)
		s.groups = append(s.groups, groupPath)
		for _, ch := range group {
			t, err := newTDMSDataType(ch.DataType)
			if err != nil {
				return nil, err
			}
			s.channels = append(s.channels, tdmsChannel{
				key:      ch.Key(),
				path:     groupPath + "/" + tdmsName(ch.Name),
				dataType: ch.DataType,
				tdmsType: t,
			})
		}
	}
	return s, nil
}

func newTDMSDataType(dt telem.DataType) (tdmsDataType, error) {
	switch dt {
	case telem.Int8T:
		return tdmsI8, nil
	case telem.Int16T:
		return tdmsI16, nil
	case telem.Int32T:
		return tdmsI32, nil
	case telem.Int64T:
		return tdmsI64, nil
	case telem.Uint8T:
		return tdmsU8, nil
	case telem.Uint16T:
		return tdmsU16, nil
	case telem.Uint32T:
		return tdmsU32, nil
	case telem.Uint64T:
		return tdmsU64, nil
	case telem.Float32T:
		return tdmsF32, nil
	case telem.Float64T:
		return tdmsF64, nil
	case telem.BooleanT:
		return tdmsBoolean, nil
	case telem.TimestampT:
		return tdmsTimeStamp, nil
	case telem.StringT, telem.JSONT, telem.UUIDT, telem.BytesT:
		return tdmsString, nil
	default:
		return 0, errors.Newf("data type %s has no TDMS form", dt)
	}
}

// tdmsName quotes name as one element of an object path.
func tdmsName(name string) string {
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

// writeHeader writes a segment that declares the file, its groups, and its channels.
func (s *tdmsSegmenter) writeHeader(w io.Writer) error {
	s.meta = telem.ByteOrder.AppendUint32(
		s.meta[:0],
		uint32(1+len(s.groups)+len(s.channels)),
	)
	s.meta = appendTDMSObject(s.meta, "/")
	for _, g := range s.groups {
		s.meta = appendTDMSObject(s.meta, g)
	}
	for _, c := range s.channels {
		s.meta = appendTDMSObject(s.meta, c.path)
	}
	s.raw = s.raw[:0]
	return s.write(w, tdmsTocMetaData|tdmsTocNewObjList)
}

func appendTDMSObject(dst []byte, path string) []byte {
	dst = appendTDMSString(dst, path)
	dst = telem.ByteOrder.AppendUint32(dst, tdmsNoRawData)
	return telem.ByteOrder.AppendUint32(dst, 0)
}

// writeFrame writes a segment that holds the samples fr carries for the file's
// channels. It writes nothing when fr carries none.
func (s *tdmsSegmenter) writeFrame(w io.Writer, fr Frame) error {
	s.meta, s.raw = s.meta[:4], s.raw[:0]
	var objects uint32
	for _, c := range s.channels {
		series := fr.Get(c.key).Series
		var count int64
		for _, ser := range series {
			count += ser.Len()
		}
		if count == 0 {
			continue
		}
		objects++
		s.meta = appendTDMSString(s.meta, c.path)
		indexSize := uint32(20)
		if c.tdmsType == tdmsString {
			indexSize = 28
		}
		s.meta = telem.ByteOrder.AppendUint32(s.meta, indexSize)
		s.meta = telem.ByteOrder.AppendUint32(s.meta, uint32(c.tdmsType))
		s.meta = telem.ByteOrder.AppendUint32(s.meta, 1)
		s.meta = telem.ByteOrder.AppendUint64(s.meta, uint64(count))
		switch c.tdmsType {
		case tdmsString:
			if err := s.appendText(c.dataType, series); err != nil {
				return err
			}
			// The total size covers the offsets and the text.
			s.meta = telem.ByteOrder.AppendUint64(
				s.meta,
				uint64(4*count)+uint64(len(s.text)),
			)
		case tdmsTimeStamp:
			for _, ser := range series {
				for i := 0; i+8 <= len(ser.Data); i += 8 {
					ts := telem.TimeStamp(telem.ByteOrder.Uint64(ser.Data[i:]))
					s.raw = appendTDMSTimeStamp(s.raw, ts)
				}
			}
		default:
			for _, ser := range series {
				s.raw = append(s.raw, ser.Data...)
			}
		}
		s.meta = telem.ByteOrder.AppendUint32(s.meta, 0)
	}
	if objects == 0 {
		return nil
	}
	telem.ByteOrder.PutUint32(s.meta, objects)
	return s.write(w, tdmsTocMetaData|tdmsTocNewObjList|tdmsTocRawData)
}

// appendText appends the raw data of a string channel: the end offset of every string,
// then the strings.
func (s *tdmsSegmenter) appendText(dt telem.DataType, series []telem.Series) error {
	s.text = s.text[:0]
	for _, ser := range series {
		for sample := range ser.Samples() {
			var err error
			if s.text, err = telem.AppendSampleText(s.text, dt, sample); err != nil {
				return err
			}
			s.raw = telem.ByteOrder.AppendUint32(s.raw, uint32(len(s.text)))
		}
	}
	s.raw = append(s.raw, s.text...)
	return nil
}

// tdmsEpochOffset is the number of seconds from the TDMS epoch, 1904-01-01 UTC, to the
// Unix epoch.
const tdmsEpochOffset = 2082844800

// appendTDMSTimeStamp appends ts as a TDMS timestamp: the fraction of the second in
// units of 2^-64 seconds, then the whole seconds since the TDMS epoch.
func appendTDMSTimeStamp(dst []byte, ts telem.TimeStamp) []byte {
	secs, nanos := int64(ts)/1e9, int64(ts)%1e9
	if nanos < 0 {
		secs, nanos = secs-1, nanos+1e9
	}
	fraction, _ := bits.Div64(uint64(nanos), 0, 1e9)
	dst = telem.ByteOrder.AppendUint64(dst, fraction)
	return telem.ByteOrder.AppendUint64(dst, uint64(secs+tdmsEpochOffset))
}

func appendTDMSString(dst []byte, v string) []byte {
	dst = telem.ByteOrder.AppendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}

// write writes the lead-in, the metadata, and the raw data of a segment.
func (s *tdmsSegmenter) write(w io.Writer, toc uint32) error {
	var leadIn [tdmsLeadInSize]byte
	copy(leadIn[:4], "TDSm")
	telem.ByteOrder.PutUint32(leadIn[4:], toc)
	telem.ByteOrder.PutUint32(leadIn[8:], tdmsVersion)
	telem.ByteOrder.PutUint64(leadIn[12:], uint64(len(s.meta)+len(s.raw)))
	telem.ByteOrder.PutUint64(leadIn[20:], uint64(len(s.meta)))
	for _, b := range [][]byte{leadIn[:], s.meta, s.raw} {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}
