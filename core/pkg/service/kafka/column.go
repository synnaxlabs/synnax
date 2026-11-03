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
	"math"
	"time"
	"unsafe"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// rejectDuplicateNames overrides the leniency of jsontext.Value.Compact.
var rejectDuplicateNames = []jsontext.Options{jsontext.AllowDuplicateNames(false)}

// variablePrefix is the size of the length prefix of a variable-density sample.
const variablePrefix = 4

// column accumulates one field's samples as series data, in arrival order. Its buffers
// survive across polls.
type column struct {
	dataType   telem.DataType
	timeFormat http.TimeFormat
	enums      []http.EnumEntry
	// density is the size of a sample, zero when samples vary in size.
	density int
	data    []byte
	// ends[i] is the end of sample i in data. It is only set when density is zero.
	ends []int
	// text is scratch space for a string sample that is not stored as text.
	text []byte
}

func newColumn(fld ReadField, dt telem.DataType) column {
	c := column{dataType: dt, enums: fld.EnumValues}
	if fld.TimeFormat != nil {
		c.timeFormat = *fld.TimeFormat
	}
	if !dt.IsVariable() {
		c.density = int(dt.Density())
	}
	return c
}

func (c *column) sample(i int) []byte {
	if c.density != 0 {
		return c.data[i*c.density : (i+1)*c.density]
	}
	start := 0
	if i > 0 {
		start = c.ends[i-1]
	}
	return c.data[start:c.ends[i]]
}

func (c *column) truncate(n int) {
	if c.density != 0 {
		c.data = c.data[:n*c.density]
		return
	}
	c.ends = c.ends[:n]
	end := 0
	if n > 0 {
		end = c.ends[n-1]
	}
	c.data = c.data[:end]
}

func (c *column) size(rows []row) int {
	if c.density != 0 {
		return len(rows) * c.density
	}
	n := 0
	for _, r := range rows {
		n += len(c.sample(r.pos))
	}
	return n
}

// append converts raw into a sample of the column's data type and appends it. It
// returns an error wrapping validate.ErrValidation when raw does not convert, which
// can leave a partial sample: truncate removes it.
func (c *column) append(raw jsontext.Value) error {
	switch c.dataType {
	case telem.TimestampT:
		ts, err := c.parseTimestamp(raw)
		if err != nil {
			return err
		}
		c.data = telem.ByteOrder.AppendUint64(c.data, uint64(ts))
		return nil
	case telem.StringT:
		if raw.Kind() != '"' {
			return errors.Wrapf(
				validate.ErrValidation, "expected a string, got %s", kindName(raw),
			)
		}
		start := c.beginVariable()
		var err error
		if c.data, err = jsontext.AppendUnquote(c.data, raw); err != nil {
			return err
		}
		c.endVariable(start)
		return nil
	case telem.JSONT:
		return c.appendJSON(raw)
	case telem.UUIDT:
		return c.appendUUID(raw)
	case telem.BytesT:
		return errors.Wrapf(
			validate.ErrValidation, "unsupported data type %s", c.dataType,
		)
	}
	switch raw.Kind() {
	case 't':
		return c.appendNumber(number{kind: intNumber, i: 1})
	case 'f':
		return c.appendNumber(number{kind: intNumber})
	case '0':
		n, err := parseNumber(raw)
		if err != nil {
			return err
		}
		if err := checkFits(n, c.dataType); err != nil {
			return err
		}
		return c.appendNumber(n)
	case '"':
		return c.appendEnum(raw)
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"expected a number or boolean, got %s",
			kindName(raw),
		)
	}
}

func (c *column) beginVariable() int {
	start := len(c.data)
	c.data = append(c.data, make([]byte, variablePrefix)...)
	return start
}

func (c *column) endVariable(start int) {
	size := len(c.data) - start - variablePrefix
	telem.ByteOrder.PutUint32(c.data[start:], uint32(size))
	c.ends = append(c.ends, len(c.data))
}

func (c *column) appendJSON(raw jsontext.Value) error {
	start := c.beginVariable()
	c.data = append(c.data, raw...)
	if kind := raw.Kind(); kind == '{' || kind == '[' {
		v := jsontext.Value(c.data[start+variablePrefix:])
		if err := v.Compact(rejectDuplicateNames...); err != nil {
			return err
		}
		c.data = c.data[:start+variablePrefix+len(v)]
	}
	c.endVariable(start)
	return nil
}

func (c *column) appendUUID(raw jsontext.Value) error {
	if raw.Kind() != '"' {
		return errors.Wrapf(
			validate.ErrValidation, "expected a UUID string, got %s", kindName(raw),
		)
	}
	if err := c.unquote(raw); err != nil {
		return err
	}
	var id uuid.UUID
	if err := id.UnmarshalText(c.text); err != nil {
		return errors.Wrapf(validate.ErrValidation, "invalid UUID %q", c.text)
	}
	c.data = append(c.data, id[:]...)
	return nil
}

func (c *column) appendEnum(raw jsontext.Value) error {
	if err := c.unquote(raw); err != nil {
		return err
	}
	for _, e := range c.enums {
		if e.Label == string(c.text) {
			n := number{kind: floatNumber, f: e.Value}
			if err := checkFits(n, c.dataType); err != nil {
				return err
			}
			return c.appendNumber(n)
		}
	}
	return errors.Wrapf(validate.ErrValidation, "unknown enum label %q", c.text)
}

func (c *column) appendNumber(n number) error {
	order := telem.ByteOrder
	switch c.dataType {
	case telem.Float64T:
		c.data = order.AppendUint64(c.data, math.Float64bits(n.float()))
	case telem.Float32T:
		c.data = order.AppendUint32(c.data, math.Float32bits(float32(n.float())))
	case telem.Int64T:
		c.data = order.AppendUint64(c.data, uint64(convertNumber[int64](n)))
	case telem.Int32T:
		c.data = order.AppendUint32(c.data, uint32(convertNumber[int32](n)))
	case telem.Int16T:
		c.data = order.AppendUint16(c.data, uint16(convertNumber[int16](n)))
	case telem.Int8T:
		c.data = append(c.data, byte(convertNumber[int8](n)))
	case telem.Uint64T:
		c.data = order.AppendUint64(c.data, convertNumber[uint64](n))
	case telem.Uint32T:
		c.data = order.AppendUint32(c.data, convertNumber[uint32](n))
	case telem.Uint16T:
		c.data = order.AppendUint16(c.data, convertNumber[uint16](n))
	case telem.Uint8T:
		c.data = append(c.data, convertNumber[uint8](n))
	case telem.BooleanT:
		var b byte
		if n.float() != 0 {
			b = 1
		}
		c.data = append(c.data, b)
	default:
		return errors.Wrapf(
			validate.ErrValidation, "unsupported data type %s", c.dataType,
		)
	}
	return nil
}

func (c *column) parseTimestamp(raw jsontext.Value) (telem.TimeStamp, error) {
	if c.timeFormat == http.TimeFormatISO8601 {
		if raw.Kind() != '"' {
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"expected an ISO 8601 string, got %s",
				kindName(raw),
			)
		}
		if err := c.unquote(raw); err != nil {
			return 0, err
		}
		// time.Parse keeps no reference to its input on success.
		text := unsafe.String(unsafe.SliceData(c.text), len(c.text))
		t, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			return 0, errors.Wrapf(
				validate.ErrValidation, "invalid ISO 8601 timestamp %q", c.text,
			)
		}
		return telem.NewTimeStamp(t), nil
	}
	if raw.Kind() != '0' {
		return 0, errors.Wrapf(
			validate.ErrValidation, "expected a number, got %s", kindName(raw),
		)
	}
	n, err := parseNumber(raw)
	if err != nil {
		return 0, err
	}
	return numericTimestamp(n, c.timeFormat)
}

func (c *column) unquote(raw jsontext.Value) (err error) {
	c.text, err = jsontext.AppendUnquote(c.text[:0], raw)
	return err
}

func kindName(raw jsontext.Value) string {
	switch raw.Kind() {
	case '"':
		return "a string"
	case '0':
		return "a number"
	case 't', 'f':
		return "a boolean"
	case 'n':
		return "null"
	case '{':
		return "an object"
	default:
		return "an array"
	}
}
