// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"strconv"
	"time"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// slot is a value a template fills in for each record.
type slot uint8

const (
	slotNone slot = iota
	slotValue
	slotChannel
	slotTimestamp
	slotUUID
)

// templateNode is one value of the JSON document a template renders.
type templateNode struct {
	// names orders members, which is nil unless the node is an object.
	names   []string
	members map[string]*templateNode
	// literal is the node's JSON text when it is known ahead of time.
	literal []byte
	slot    slot
	// format encodes slotTimestamp.
	format http.TimeFormat
}

// set places leaf at pointer, creating the objects on the way. The pointer must be
// valid and must not overlap a pointer already set.
func (n *templateNode) set(pointer string, leaf *templateNode) error {
	tokens, err := pointerTokens(pointer)
	if err != nil {
		return err
	}
	last := len(tokens) - 1
	for _, t := range tokens[:last] {
		c, ok := n.members[t]
		if !ok {
			c = &templateNode{}
			n.add(t, c)
		}
		n = c
	}
	n.add(tokens[last], leaf)
	return nil
}

func (n *templateNode) add(name string, c *templateNode) {
	if n.members == nil {
		n.members = make(map[string]*templateNode)
	}
	n.names = append(n.names, name)
	n.members[name] = c
}

type segment struct {
	prefix []byte
	slot   slot
	format http.TimeFormat
}

// template renders the JSON document of a record. Everything known at configuration is
// rendered once, so a record only costs its slots.
type template struct {
	segments []segment
	// suffix is the JSON text after the last slot.
	suffix []byte
}

func newTemplate(rec Record) (template, error) {
	root := &templateNode{}
	err := root.set(rec.ValuePointer, &templateNode{slot: slotValue})
	if err != nil {
		return template{}, err
	}
	if p := rec.ChannelPointer; p != nil {
		if err := root.set(*p, &templateNode{slot: slotChannel}); err != nil {
			return template{}, err
		}
	}
	if p := rec.TimestampPointer; p != nil {
		leaf := &templateNode{slot: slotTimestamp, format: rec.TimeFormat}
		if err := root.set(*p, leaf); err != nil {
			return template{}, err
		}
	}
	for _, f := range rec.Fields {
		pointer, leaf, err := fieldNode(f)
		if err != nil {
			return template{}, err
		}
		if err := root.set(pointer, leaf); err != nil {
			return template{}, err
		}
	}
	var t template
	text, err := t.render(root, nil)
	if err != nil {
		return template{}, err
	}
	t.suffix = text
	return t, nil
}

func fieldNode(field http.WriteField) (string, *templateNode, error) {
	switch v := field.Variant.(type) {
	case http.StaticWriteField:
		literal, err := json.Marshal(v.Value)
		if err != nil {
			return "", nil, errors.Wrapf(err, "field %q", v.Key)
		}
		return v.Pointer, &templateNode{literal: literal}, nil
	case http.GeneratedWriteField:
		switch v.Generator {
		case http.GeneratorTypeUUID:
			return v.Pointer, &templateNode{slot: slotUUID}, nil
		case http.GeneratorTypeTimestamp:
			format := http.TimeFormatUnixNs
			if v.TimeFormat != nil {
				format = *v.TimeFormat
			}
			leaf := &templateNode{slot: slotTimestamp, format: format}
			return v.Pointer, leaf, nil
		default:
			return "", nil, errors.Wrapf(
				validate.ErrValidation, "unknown generator %q", v.Generator,
			)
		}
	default:
		return "", nil, errors.Newf("unknown write field variant %T", v)
	}
}

// render appends n to text, which holds the JSON text since the last slot. Each slot
// ends a segment. It returns the JSON text since the last slot.
func (t *template) render(n *templateNode, text []byte) ([]byte, error) {
	if n.slot != slotNone {
		t.segments = append(
			t.segments, segment{prefix: text, slot: n.slot, format: n.format},
		)
		return nil, nil
	}
	if n.members == nil {
		return append(text, n.literal...), nil
	}
	text = append(text, '{')
	var err error
	for i, name := range n.names {
		if i > 0 {
			text = append(text, ',')
		}
		if text, err = jsontext.AppendQuote(text, name); err != nil {
			return nil, err
		}
		text = append(text, ':')
		if text, err = t.render(n.members[name], text); err != nil {
			return nil, err
		}
	}
	return append(text, '}'), nil
}

// append appends the document for sample, a sample of dt on wc stamped ts, to dst.
func (t *template) append(
	dst []byte,
	wc *writeChannel,
	dt telem.DataType,
	sample []byte,
	ts telem.TimeStamp,
) ([]byte, error) {
	var err error
	for i := range t.segments {
		seg := &t.segments[i]
		dst = append(dst, seg.prefix...)
		switch seg.slot {
		case slotValue:
			dst, err = wc.appendSample(dst, dt, sample)
		case slotChannel:
			dst = append(dst, wc.name...)
		case slotTimestamp:
			dst, err = appendTimestamp(dst, ts, seg.format)
		case slotUUID:
			dst = append(dst, '"')
			dst, err = uuid.New().AppendText(dst)
			dst = append(dst, '"')
		}
		if err != nil {
			return nil, err
		}
	}
	return append(dst, t.suffix...), nil
}

// appendSample appends sample, a sample of dt, as the JSON value wc produces.
func (wc *writeChannel) appendSample(
	dst []byte,
	dt telem.DataType,
	sample []byte,
) ([]byte, error) {
	if dt == telem.TimestampT {
		ts := telem.TimeStamp(telem.ByteOrder.Uint64(sample))
		return appendTimestamp(dst, ts, *wc.cfg.TimeFormat)
	}
	if dt.IsVariable() || dt == telem.UUIDT {
		if wc.cfg.JSONType != http.JSONTypeString {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"data type %s only serializes as a string",
				dt,
			)
		}
		if dt == telem.UUIDT {
			dst = append(dst, '"')
			dst, err := uuid.UUID(sample).AppendText(dst)
			return append(dst, '"'), err
		}
		return jsontext.AppendQuote(dst, sample)
	}
	n, err := sampleNumber(dt, sample)
	if err != nil {
		return nil, err
	}
	switch wc.cfg.JSONType {
	case http.JSONTypeNumber:
		return appendJSONNumber(dst, n)
	case http.JSONTypeBoolean:
		return strconv.AppendBool(dst, n.float() != 0), nil
	case http.JSONTypeString:
		if len(wc.cfg.EnumValues) == 0 {
			dst = append(dst, '"')
			return append(n.appendText(dst), '"'), nil
		}
		f := n.float()
		for i, e := range wc.cfg.EnumValues {
			if e.Value == f {
				return append(dst, wc.labels[i]...), nil
			}
		}
		return nil, errors.Wrapf(
			validate.ErrValidation, "no enum label for value %s", n,
		)
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "unknown json_type %q", wc.cfg.JSONType,
		)
	}
}

func sampleNumber(dt telem.DataType, sample []byte) (number, error) {
	order := telem.ByteOrder
	switch dt {
	case telem.Float64T:
		f := math.Float64frombits(order.Uint64(sample))
		return number{kind: floatNumber, f: f}, nil
	case telem.Float32T:
		f := float64(math.Float32frombits(order.Uint32(sample)))
		return number{kind: floatNumber, f: f}, nil
	case telem.Int64T:
		return number{kind: intNumber, i: int64(order.Uint64(sample))}, nil
	case telem.Int32T:
		return number{kind: intNumber, i: int64(int32(order.Uint32(sample)))}, nil
	case telem.Int16T:
		return number{kind: intNumber, i: int64(int16(order.Uint16(sample)))}, nil
	case telem.Int8T:
		return number{kind: intNumber, i: int64(int8(sample[0]))}, nil
	case telem.Uint64T:
		return number{kind: uintNumber, u: order.Uint64(sample)}, nil
	case telem.Uint32T:
		return number{kind: uintNumber, u: uint64(order.Uint32(sample))}, nil
	case telem.Uint16T:
		return number{kind: uintNumber, u: uint64(order.Uint16(sample))}, nil
	case telem.Uint8T:
		return number{kind: uintNumber, u: uint64(sample[0])}, nil
	case telem.BooleanT:
		n := number{kind: intNumber}
		if sample[0] != 0 {
			n.i = 1
		}
		return n, nil
	default:
		return number{}, errors.Wrapf(
			validate.ErrValidation, "data type %s is not numeric", dt,
		)
	}
}

// appendJSONNumber appends n as a JSON number. A float takes the shortest form that
// round trips, and integers stay exact.
func appendJSONNumber(dst []byte, n number) ([]byte, error) {
	if n.kind != floatNumber {
		return n.appendText(dst), nil
	}
	if math.IsNaN(n.f) || math.IsInf(n.f, 0) {
		return nil, errors.Wrapf(
			validate.ErrValidation, "%s is not a JSON number", n,
		)
	}
	format := byte('f')
	if abs := math.Abs(n.f); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	dst = strconv.AppendFloat(dst, n.f, format, -1, 64)
	// A JSON number writes an exponent without padding: e-9, not e-09.
	if l := len(dst); format == 'e' && l >= 4 && dst[l-4] == 'e' && dst[l-3] == '-' &&
		dst[l-2] == '0' {
		dst[l-2] = dst[l-1]
		dst = dst[:l-1]
	}
	return dst, nil
}

// appendTimestamp appends ts as a JSON value encoded with format. Nanoseconds stay
// integral so no precision is lost on the round trip.
func appendTimestamp(
	dst []byte,
	ts telem.TimeStamp,
	format http.TimeFormat,
) ([]byte, error) {
	if format == http.TimeFormatISO8601 {
		dst = append(dst, '"')
		dst = ts.Time().UTC().AppendFormat(dst, time.RFC3339Nano)
		return append(dst, '"'), nil
	}
	unit, err := timeUnit(format)
	if err != nil {
		return nil, err
	}
	if unit == telem.Nanosecond {
		return strconv.AppendInt(dst, int64(ts), 10), nil
	}
	return appendJSONNumber(dst, number{
		kind: floatNumber,
		f:    float64(ts) / float64(unit),
	})
}
