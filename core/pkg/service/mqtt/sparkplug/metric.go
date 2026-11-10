// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sparkplug

import (
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/proto"
)

const (
	// RebirthMetric is the name of the node control tag that requests a new birth.
	RebirthMetric = "Node Control/Rebirth"
	// BdSeqMetric is the name of the tag that ties a death message to its birth.
	BdSeqMetric = "bdSeq"
)

// Reserved reports whether name is a tag that every birth of an edge node declares, so
// that no other tag can take it.
func Reserved(name string) bool { return name == RebirthMetric || name == BdSeqMetric }

// DataType is the Sparkplug B data type of a tag.
type DataType uint32

// The data types that a Metric value can hold. Every other type, such as a template or
// a data set, decodes to a nil value.
const (
	Int8     = DataType(pb.DataType_Int8)
	Int16    = DataType(pb.DataType_Int16)
	Int32    = DataType(pb.DataType_Int32)
	Int64    = DataType(pb.DataType_Int64)
	UInt8    = DataType(pb.DataType_UInt8)
	UInt16   = DataType(pb.DataType_UInt16)
	UInt32   = DataType(pb.DataType_UInt32)
	UInt64   = DataType(pb.DataType_UInt64)
	Float    = DataType(pb.DataType_Float)
	Double   = DataType(pb.DataType_Double)
	Boolean  = DataType(pb.DataType_Boolean)
	String   = DataType(pb.DataType_String)
	DateTime = DataType(pb.DataType_DateTime)
	Text     = DataType(pb.DataType_Text)
	UUID     = DataType(pb.DataType_UUID)
)

// Supported reports whether a Metric can hold a value of the type.
func (t DataType) Supported() bool {
	return t >= Int8 && t <= UUID
}

// String returns the name that the Sparkplug B specification gives the type.
func (t DataType) String() string { return pb.DataType(t).String() }

// bits returns the width of an integer type.
func (t DataType) bits() int {
	switch t {
	case Int8, UInt8:
		return 8
	case Int16, UInt16:
		return 16
	case Int32, UInt32:
		return 32
	}
	return 64
}

// Metric is one value of one tag.
type Metric struct {
	// Value is nil for a null value, for a data type with no Value form, and for a
	// DateTime later than a telem.TimeStamp can hold.
	Value Value
	Name  string
	// Timestamp is the acquisition time. Zero when the message gave none, or gave one
	// later than a telem.TimeStamp can hold.
	Timestamp telem.TimeStamp
	DataType  DataType
	// Historical is true for a value that is not the current value of the tag.
	Historical bool
}

// tag is what an edge node declares about one tag.
type tag struct {
	name     string
	alias    uint64
	dataType DataType
}

// tagTable holds the tags of an edge node, or of one of its devices.
type tagTable struct {
	byName  map[string]tag
	byAlias map[uint64]tag
}

func makeTagTable(size int) tagTable {
	return tagTable{
		byName:  make(map[string]tag, size),
		byAlias: make(map[uint64]tag, size),
	}
}

// resolve returns the tag that m names: by its name when it has one, else by its
// alias.
func (t tagTable) resolve(m *pb.Payload_Metric) (tag, bool) {
	if name := m.GetName(); name != "" {
		tg, ok := t.byName[name]
		return tg, ok
	}
	if m.Alias != nil {
		tg, ok := t.byAlias[m.GetAlias()]
		return tg, ok
	}
	return tag{}, false
}

// decodeMetric returns the Go form of m, a metric of tg in payload p. A data type on m
// overrides the one that tg declares.
func decodeMetric(p *pb.Payload, m *pb.Payload_Metric, tg tag) Metric {
	dt := tg.dataType
	if m.Datatype != nil {
		dt = DataType(m.GetDatatype())
	}
	stamp := p.GetTimestamp()
	if m.Timestamp != nil {
		stamp = m.GetTimestamp()
	}
	ts, _ := fromMillis(stamp)
	return Metric{
		Name:       tg.name,
		DataType:   dt,
		Value:      decodeValue(m, dt),
		Timestamp:  ts,
		Historical: m.GetIsHistorical(),
	}
}

// maxMillis is the latest time, in milliseconds after the Unix epoch, that a
// telem.TimeStamp can hold.
const maxMillis = uint64(telem.TimeStampMax / telem.MillisecondTS)

// fromMillis returns the time ms milliseconds after the Unix epoch. ok is false when a
// telem.TimeStamp cannot hold it.
func fromMillis(ms uint64) (ts telem.TimeStamp, ok bool) {
	if ms > maxMillis {
		return 0, false
	}
	return telem.TimeStamp(ms) * telem.MillisecondTS, true
}

// decodeValue returns the value of m as the data type dt.
func decodeValue(m *pb.Payload_Metric, dt DataType) Value {
	if m.GetIsNull() {
		return nil
	}
	// Tahu puts integers of up to 32 bits in int_value and wider ones in long_value.
	// Some edge nodes do not, so an integer accepts both.
	var (
		bits    uint64
		integer bool
	)
	switch v := m.GetValue().(type) {
	case *pb.Payload_Metric_IntValue:
		bits, integer = uint64(v.IntValue), true
	case *pb.Payload_Metric_LongValue:
		bits, integer = v.LongValue, true
	}
	switch dt {
	case Int8:
		if integer {
			return IntValue(int8(bits))
		}
	case Int16:
		if integer {
			return IntValue(int16(bits))
		}
	case Int32:
		if integer {
			return IntValue(int32(bits))
		}
	case Int64:
		if integer {
			return IntValue(bits)
		}
	case UInt8, UInt16, UInt32, UInt64:
		if integer {
			return UintValue(bits)
		}
	case DateTime:
		if ts, ok := fromMillis(bits); integer && ok {
			return DateTimeValue(ts)
		}
	case Float:
		if v, ok := m.GetValue().(*pb.Payload_Metric_FloatValue); ok {
			return FloatValue(v.FloatValue)
		}
	case Double:
		if v, ok := m.GetValue().(*pb.Payload_Metric_DoubleValue); ok {
			return FloatValue(v.DoubleValue)
		}
	case Boolean:
		if v, ok := m.GetValue().(*pb.Payload_Metric_BooleanValue); ok {
			return BoolValue(v.BooleanValue)
		}
	case String, Text, UUID:
		if v, ok := m.GetValue().(*pb.Payload_Metric_StringValue); ok {
			return StringValue(v.StringValue)
		}
	}
	return nil
}

// encodeValue sets the value of m to v as the data type dt. A number or a Boolean
// converts to any numeric type or to Boolean when it fits.
func encodeValue(m *pb.Payload_Metric, dt DataType, v Value) error {
	n, isNumber := toNumber(v)
	switch dt {
	case Int8, Int16, Int32, Int64:
		i, ok := n.int64()
		if !isNumber || !ok || !fitsInt(i, dt.bits()) {
			return misfit(v, dt)
		}
		if dt == Int64 {
			m.Value = &pb.Payload_Metric_LongValue{LongValue: uint64(i)}
		} else {
			m.Value = &pb.Payload_Metric_IntValue{IntValue: uint32(int32(i))}
		}
	case UInt8, UInt16, UInt32, UInt64:
		u, ok := n.uint64()
		if !isNumber || !ok || !fitsUint(u, dt.bits()) {
			return misfit(v, dt)
		}
		if dt == UInt64 {
			m.Value = &pb.Payload_Metric_LongValue{LongValue: u}
		} else {
			m.Value = &pb.Payload_Metric_IntValue{IntValue: uint32(u)}
		}
	case Float:
		f, ok := n.float32()
		if !isNumber || !ok {
			return misfit(v, dt)
		}
		m.Value = &pb.Payload_Metric_FloatValue{FloatValue: f}
	case Double:
		if !isNumber {
			return misfit(v, dt)
		}
		m.Value = &pb.Payload_Metric_DoubleValue{DoubleValue: n.float64()}
	case Boolean:
		if !isNumber {
			return misfit(v, dt)
		}
		m.Value = &pb.Payload_Metric_BooleanValue{BooleanValue: n.float64() != 0}
	case String, Text, UUID:
		s, ok := v.(StringValue)
		if !ok {
			return misfit(v, dt)
		}
		m.Value = &pb.Payload_Metric_StringValue{StringValue: string(s)}
	case DateTime:
		ts, ok := v.(DateTimeValue)
		if !ok || ts < 0 {
			return misfit(v, dt)
		}
		m.Value = &pb.Payload_Metric_LongValue{
			LongValue: uint64(telem.TimeStamp(ts) / telem.MillisecondTS),
		}
	default:
		return errors.Wrapf(errValue, "%s is not supported", dt)
	}
	return nil
}

func toMillis(ts telem.TimeStamp) *uint64 {
	return new(uint64(ts / telem.MillisecondTS))
}

// EncodeCommand returns the NCMD message, or the DCMD message when device is not empty,
// that writes the value of m to the tag of node that m names. A command carries no
// sequence number.
func EncodeCommand(
	node NodeID,
	device string,
	m Metric,
	now telem.TimeStamp,
) (Publication, error) {
	metric := &pb.Payload_Metric{
		Name:      new(m.Name),
		Timestamp: toMillis(now),
		Datatype:  new(uint32(m.DataType)),
	}
	if err := encodeValue(metric, m.DataType, m.Value); err != nil {
		return Publication{}, errors.Wrapf(err, "tag %s", m.Name)
	}
	payload, err := proto.Marshal(&pb.Payload{
		Timestamp: toMillis(now),
		Metrics:   []*pb.Payload_Metric{metric},
	})
	if err != nil {
		return Publication{}, err
	}
	return Publication{
		Topic:   CommandTopic(node, device).String(),
		Payload: payload,
	}, nil
}

// EncodeRebirth returns the NCMD message that asks node to publish its birth messages
// again.
func EncodeRebirth(node NodeID, now telem.TimeStamp) (Publication, error) {
	return EncodeCommand(
		node,
		"",
		Metric{Name: RebirthMetric, DataType: Boolean, Value: BoolValue(true)},
		now,
	)
}
