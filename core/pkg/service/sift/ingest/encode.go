// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ingest

import (
	"slices"
	"time"
	"uuid"

	ingestv1 "github.com/sift-stack/sift/go/gen/sift/ingest/v1"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// column finds the samples of one data channel in a frame by alignment.
type column struct {
	series []telem.Series
	// cur is the index in series of the series that held the last sample found.
	cur int
	// variable holds each sample of series[cur] when its data type is variable.
	variable [][]byte
}

func newColumn(series []telem.Series) *column {
	return &column{series: series, cur: -1}
}

// encode calls send with one request per timestamp of f in fr at or after from that has
// a value. It returns the timestamp after the last one it read and the number of values
// it sent. It sets each element of last that has a value to the time after the last
// value of its channel.
func encode(
	configID string,
	f flow,
	fr framer.Frame,
	from telem.TimeStamp,
	last []telem.TimeStamp,
	send func(*ingestv1.IngestWithConfigDataStreamRequest) error,
) (telem.TimeStamp, int64, error) {
	var (
		next    = from
		samples int64
		columns = make([]*column, len(f.channels))
	)
	for i, ch := range f.channels {
		columns[i] = newColumn(fr.Get(ch.Key()).Series)
	}
	for _, idx := range fr.Get(f.index.Key()).Series {
		for j := range int(idx.Len()) {
			ts := idx.ValueAt[telem.TimeStamp](j)
			if ts < from {
				continue
			}
			a := idx.Alignment.AddSamples(uint32(j))
			req := &ingestv1.IngestWithConfigDataStreamRequest{
				IngestionConfigId: configID,
				Flow:              f.index.Name,
				Timestamp:         timestamppb.New(time.Unix(0, int64(ts))),
				ChannelValues: make(
					[]*ingestv1.IngestWithConfigDataChannelValue,
					len(columns),
				),
			}
			var n int64
			for i, c := range columns {
				v := c.value(a)
				if v != empty {
					n++
					last[i] = ts + 1
				}
				req.ChannelValues[i] = v
			}
			if n == 0 {
				next = max(next, ts+1)
				continue
			}
			if err := send(req); err != nil {
				return next, samples, err
			}
			samples += n
			next = max(next, ts+1)
		}
	}
	return next, samples, nil
}

var empty = &ingestv1.IngestWithConfigDataChannelValue{
	Type: &ingestv1.IngestWithConfigDataChannelValue_Empty{Empty: &emptypb.Empty{}},
}

// value returns the sample of the column at alignment a, or empty if the frame holds
// none.
func (c *column) value(a telem.Alignment) *ingestv1.IngestWithConfigDataChannelValue {
	if c.cur < 0 || !c.series[c.cur].AlignmentBounds().Contains(a) {
		c.cur = slices.IndexFunc(c.series, func(s telem.Series) bool {
			return s.AlignmentBounds().Contains(a)
		})
		if c.cur < 0 {
			return empty
		}
		c.variable = nil
		if s := c.series[c.cur]; s.DataType.IsVariable() {
			for sample := range s.Samples() {
				c.variable = append(c.variable, sample)
			}
		}
	}
	s := c.series[c.cur]
	i := int(a.SampleIndex() - s.Alignment.SampleIndex())
	switch s.DataType {
	case telem.Float64T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Double{
				Double: s.ValueAt[float64](i),
			},
		}
	case telem.Float32T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Float{
				Float: s.ValueAt[float32](i),
			},
		}
	case telem.Int8T:
		return int32Value(int32(s.ValueAt[int8](i)))
	case telem.Int16T:
		return int32Value(int32(s.ValueAt[int16](i)))
	case telem.Int32T:
		return int32Value(s.ValueAt[int32](i))
	case telem.Int64T:
		return int64Value(s.ValueAt[int64](i))
	case telem.TimestampT:
		return int64Value(int64(s.ValueAt[telem.TimeStamp](i)))
	case telem.Uint8T:
		return uint32Value(uint32(s.ValueAt[uint8](i)))
	case telem.Uint16T:
		return uint32Value(uint32(s.ValueAt[uint16](i)))
	case telem.Uint32T:
		return uint32Value(s.ValueAt[uint32](i))
	case telem.Uint64T:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Uint64{
				Uint64: s.ValueAt[uint64](i),
			},
		}
	case telem.BooleanT:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Bool{
				Bool: s.ValueAt[uint8](i) != 0,
			},
		}
	case telem.UUIDT:
		return stringValue(uuid.UUID(s.At(i)).String())
	case telem.StringT, telem.JSONT:
		return stringValue(string(c.variable[i]))
	case telem.BytesT:
		return &ingestv1.IngestWithConfigDataChannelValue{
			Type: &ingestv1.IngestWithConfigDataChannelValue_Bytes{
				Bytes: c.variable[i],
			},
		}
	default:
		return empty
	}
}

func int32Value(v int32) *ingestv1.IngestWithConfigDataChannelValue {
	return &ingestv1.IngestWithConfigDataChannelValue{
		Type: &ingestv1.IngestWithConfigDataChannelValue_Int32{Int32: v},
	}
}

func int64Value(v int64) *ingestv1.IngestWithConfigDataChannelValue {
	return &ingestv1.IngestWithConfigDataChannelValue{
		Type: &ingestv1.IngestWithConfigDataChannelValue_Int64{Int64: v},
	}
}

func uint32Value(v uint32) *ingestv1.IngestWithConfigDataChannelValue {
	return &ingestv1.IngestWithConfigDataChannelValue{
		Type: &ingestv1.IngestWithConfigDataChannelValue_Uint32{Uint32: v},
	}
}

func stringValue(v string) *ingestv1.IngestWithConfigDataChannelValue {
	return &ingestv1.IngestWithConfigDataChannelValue{
		Type: &ingestv1.IngestWithConfigDataChannelValue_String_{String_: v},
	}
}
