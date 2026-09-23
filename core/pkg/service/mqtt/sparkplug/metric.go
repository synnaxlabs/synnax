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
	"math"

	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/proto"
)

const (
	// RebirthMetric is the name of the node control tag that requests a new birth.
	RebirthMetric = "Node Control/Rebirth"
	// BdSeqMetric is the name of the tag that ties a death message to its birth.
	BdSeqMetric = "bdSeq"
)

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

// Metric is one value of one tag.
type Metric struct {
	// Value is an int64, a uint64, a float64, a bool, a string, or, for DateTime, a
	// telem.TimeStamp. It is nil for a null value and for a data type with no Go form.
	Value any
	Name  string
	// Timestamp is the acquisition time. Zero when the message gave none.
	Timestamp telem.TimeStamp
	DataType  DataType
	// Historical is true for a value that is not the current value of the tag.
	Historical bool
}

const millisecond = telem.TimeStamp(1e6)

// decodeValue returns the Go form of the value of m as the data type dt.
func decodeValue(m *pb.Payload_Metric, dt DataType) any {
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
			return int64(int8(bits))
		}
	case Int16:
		if integer {
			return int64(int16(bits))
		}
	case Int32:
		if integer {
			return int64(int32(bits))
		}
	case Int64:
		if integer {
			return int64(bits)
		}
	case UInt8, UInt16, UInt32, UInt64:
		if integer {
			return bits
		}
	case DateTime:
		if integer {
			return telem.TimeStamp(bits) * millisecond
		}
	case Float:
		if v, ok := m.GetValue().(*pb.Payload_Metric_FloatValue); ok {
			return float64(v.FloatValue)
		}
	case Double:
		if v, ok := m.GetValue().(*pb.Payload_Metric_DoubleValue); ok {
			return v.DoubleValue
		}
	case Boolean:
		if v, ok := m.GetValue().(*pb.Payload_Metric_BooleanValue); ok {
			return v.BooleanValue
		}
	case String, Text, UUID:
		if v, ok := m.GetValue().(*pb.Payload_Metric_StringValue); ok {
			return v.StringValue
		}
	}
	return nil
}

// errValue is returned when a value cannot be sent as a data type.
var errValue = errors.Wrap(validate.ErrValidation, "value does not fit the data type")

// encodeValue sets the value of m to value as the data type dt. value has one of the
// Go forms of Metric.Value.
func encodeValue(m *pb.Payload_Metric, dt DataType, value any) error {
	switch dt {
	case Int8, Int16, Int32, Int64:
		i, ok := toInt64(value)
		lo, hi := intBounds(dt)
		if !ok || i < lo || i > hi {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		if dt == Int64 {
			m.Value = &pb.Payload_Metric_LongValue{LongValue: uint64(i)}
		} else {
			m.Value = &pb.Payload_Metric_IntValue{IntValue: uint32(int32(i))}
		}
	case UInt8, UInt16, UInt32, UInt64:
		u, ok := toUint64(value)
		if !ok || u > uintMax(dt) {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		if dt == UInt64 {
			m.Value = &pb.Payload_Metric_LongValue{LongValue: u}
		} else {
			m.Value = &pb.Payload_Metric_IntValue{IntValue: uint32(u)}
		}
	case DateTime:
		ts, ok := value.(telem.TimeStamp)
		if !ok || ts < 0 {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		m.Value = &pb.Payload_Metric_LongValue{LongValue: uint64(ts / millisecond)}
	case Float:
		f, ok := toFloat64(value)
		// A finite value past the float32 range would go out as an infinity.
		if !ok || (math.Abs(f) > math.MaxFloat32 && !math.IsInf(f, 0)) {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		m.Value = &pb.Payload_Metric_FloatValue{FloatValue: float32(f)}
	case Double:
		f, ok := toFloat64(value)
		if !ok {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		m.Value = &pb.Payload_Metric_DoubleValue{DoubleValue: f}
	case Boolean:
		b, ok := value.(bool)
		if !ok {
			f, isNumber := toFloat64(value)
			if !isNumber {
				return errors.Wrapf(errValue, "%v as %s", value, dt)
			}
			b = f != 0
		}
		m.Value = &pb.Payload_Metric_BooleanValue{BooleanValue: b}
	case String, Text, UUID:
		s, ok := value.(string)
		if !ok {
			return errors.Wrapf(errValue, "%v as %s", value, dt)
		}
		m.Value = &pb.Payload_Metric_StringValue{StringValue: s}
	default:
		return errors.Wrapf(errValue, "%s is not supported", dt)
	}
	return nil
}

func intBounds(dt DataType) (lo, hi int64) {
	switch dt {
	case Int8:
		return math.MinInt8, math.MaxInt8
	case Int16:
		return math.MinInt16, math.MaxInt16
	case Int32:
		return math.MinInt32, math.MaxInt32
	}
	return math.MinInt64, math.MaxInt64
}

func uintMax(dt DataType) uint64 {
	switch dt {
	case UInt8:
		return math.MaxUint8
	case UInt16:
		return math.MaxUint16
	case UInt32:
		return math.MaxUint32
	}
	return math.MaxUint64
}

// toInt64 converts a whole number to an int64.
func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case uint64:
		return int64(v), v <= math.MaxInt64
	case float64:
		return int64(v), v == math.Trunc(v) && v >= -(1<<63) && v < 1<<63
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// toUint64 converts a whole number that is not negative to a uint64.
func toUint64(value any) (uint64, bool) {
	switch v := value.(type) {
	case uint64:
		return v, true
	case int64:
		return uint64(v), v >= 0
	case float64:
		return uint64(v), v == math.Trunc(v) && v >= 0 && v < 1<<64
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func toMillis(ts telem.TimeStamp) *uint64 { return new(uint64(ts / millisecond)) }

// EncodeCommand returns the payload of an NCMD or DCMD message that writes the value
// of m to the tag that m names. A command carries no sequence number.
func EncodeCommand(m Metric, now telem.TimeStamp) ([]byte, error) {
	metric := &pb.Payload_Metric{
		Name:      new(m.Name),
		Timestamp: toMillis(now),
		Datatype:  new(uint32(m.DataType)),
	}
	if err := encodeValue(metric, m.DataType, m.Value); err != nil {
		return nil, errors.Wrapf(err, "tag %s", m.Name)
	}
	return proto.Marshal(&pb.Payload{
		Timestamp: toMillis(now),
		Metrics:   []*pb.Payload_Metric{metric},
	})
}

// EncodeRebirth returns the payload of the NCMD message that asks an edge node to
// publish its birth messages again.
func EncodeRebirth(now telem.TimeStamp) ([]byte, error) {
	return EncodeCommand(
		Metric{Name: RebirthMetric, DataType: Boolean, Value: true},
		now,
	)
}
