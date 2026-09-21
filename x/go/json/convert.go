// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// ErrConversion is returned when a JSON value cannot be converted to a telemetry
// sample, or a sample to a JSON value.
var ErrConversion = errors.Wrap(validate.ErrValidation, "json conversion failed")

// TimeFormat is the encoding of a timestamp within a JSON document.
type TimeFormat uint8

const (
	// ISO8601 is an RFC 3339 string such as "2026-01-02T03:04:05.678Z".
	ISO8601 TimeFormat = iota
	// UnixSecond is a count of seconds since the Unix epoch.
	UnixSecond
	// UnixMillisecond is a count of milliseconds since the Unix epoch.
	UnixMillisecond
	// UnixMicrosecond is a count of microseconds since the Unix epoch.
	UnixMicrosecond
	// UnixNanosecond is a count of nanoseconds since the Unix epoch.
	UnixNanosecond
)

// ParseTimeFormat parses the config form of a TimeFormat: "iso8601", "unix_sec",
// "unix_ms", "unix_us", or "unix_ns". It returns an error wrapping
// validate.ErrValidation for any other text.
func ParseTimeFormat(text string) (TimeFormat, error) {
	switch text {
	case "iso8601":
		return ISO8601, nil
	case "unix_sec":
		return UnixSecond, nil
	case "unix_ms":
		return UnixMillisecond, nil
	case "unix_us":
		return UnixMicrosecond, nil
	case "unix_ns":
		return UnixNanosecond, nil
	}
	return ISO8601, errors.Wrapf(
		validate.ErrValidation,
		`unknown time format %q: expected "iso8601", "unix_sec", "unix_ms", `+
			`"unix_us", or "unix_ns"`,
		text,
	)
}

// nanosPerUnit returns the number of nanoseconds in one unit of a numeric format.
func (f TimeFormat) nanosPerUnit() int64 {
	switch f {
	case UnixSecond:
		return int64(time.Second)
	case UnixMillisecond:
		return int64(time.Millisecond)
	case UnixMicrosecond:
		return int64(time.Microsecond)
	default:
		return 1
	}
}

// EnumMap maps string labels in a JSON document to numeric sample values.
type EnumMap map[string]float64

// SupportsSampleTarget reports whether AppendSample can produce samples of dt.
func SupportsSampleTarget(dt telem.DataType) bool {
	switch dt {
	case telem.Float64T, telem.Float32T,
		telem.Int64T, telem.Int32T, telem.Int16T, telem.Int8T,
		telem.Uint64T, telem.Uint32T, telem.Uint16T, telem.Uint8T,
		telem.TimestampT, telem.StringT:
		return true
	}
	return false
}

// AppendSample converts value to one sample of dt and appends its series encoding to
// dst, in the manner of strconv.AppendInt. value is a node of a document decoded by
// encoding/json; decode with Decoder.UseNumber to keep full 64-bit integer precision.
//
// Numbers, booleans (as 1 and 0), and numeric strings convert to numeric types. A
// string found in enums converts to its mapped number. Any value converts to a
// string, as its JSON text when it is not already a string. format applies only when dt
// is telem.TimestampT.
//
// AppendSample returns an error wrapping ErrConversion when the value has a fractional
// part or is out of bounds for an integer type, or when the value and dt are
// incompatible.
func AppendSample(
	dst []byte,
	dt telem.DataType,
	value any,
	format TimeFormat,
	enums EnumMap,
) ([]byte, error) {
	out, err := appendSample(dst, dt, value, format, enums)
	if err == nil {
		return out, nil
	}
	text, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		text = []byte("<unencodable>")
	}
	if msg := err.Error(); msg != "" {
		return nil, errors.Wrapf(
			ErrConversion, "cannot convert %s to %s: %s", text, dt, msg,
		)
	}
	return nil, errors.Wrapf(ErrConversion, "cannot convert %s to %s", text, dt)
}

// errIncompatible marks a value whose JSON type can never convert to the target.
var errIncompatible = errors.New("")

func appendSample(
	dst []byte,
	dt telem.DataType,
	value any,
	format TimeFormat,
	enums EnumMap,
) ([]byte, error) {
	if dt == telem.TimestampT {
		ts, err := toTimeStamp(value, format)
		if err != nil {
			return dst, err
		}
		return telem.ByteOrder.AppendUint64(dst, uint64(ts)), nil
	}
	if dt == telem.StringT {
		if s, ok := value.(string); ok {
			return append(dst, telem.MarshalVariableSample([]byte(s))...), nil
		}
		text, err := json.Marshal(value)
		if err != nil {
			return dst, err
		}
		return append(dst, telem.MarshalVariableSample(text)...), nil
	}
	n, err := toNumber(value, dt, enums)
	if err != nil {
		return dst, err
	}
	return appendNumber(dst, dt, n)
}

// number is a JSON number held in the widest form that represents it exactly.
type number struct {
	// kind selects which of i, u, and f holds the value.
	kind numberKind
	// i holds a negative integer.
	i int64
	// u holds a non-negative integer.
	u uint64
	// f holds a number with a fraction or an exponent.
	f float64
}

type numberKind uint8

const (
	kindInt numberKind = iota
	kindUint
	kindFloat
)

var errNotNumber = errors.New("not a valid number")

func isFloatType(dt telem.DataType) bool {
	return dt == telem.Float64T || dt == telem.Float32T
}

// parseNumber parses the text of a number. An integer target parses integer text
// without passing through float64, which loses precision above 2^53.
func parseNumber(text string, dt telem.DataType) (number, error) {
	if text == "" {
		return number{}, errNotNumber
	}
	if !isFloatType(dt) && isIntegerText(text) {
		if text[0] == '-' {
			i, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return number{}, errOutOfBounds
			}
			return number{kind: kindInt, i: i}, nil
		}
		u, err := strconv.ParseUint(strings.TrimPrefix(text, "+"), 10, 64)
		if err != nil {
			return number{}, errOutOfBounds
		}
		return number{kind: kindUint, u: u}, nil
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return number{}, errNotNumber
	}
	return number{kind: kindFloat, f: f}, nil
}

func isIntegerText(text string) bool {
	start := 0
	if text[0] == '-' || text[0] == '+' {
		start = 1
	}
	if start == len(text) {
		return false
	}
	for i := start; i < len(text); i++ {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func toNumber(value any, dt telem.DataType, enums EnumMap) (number, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return number{kind: kindUint, u: 1}, nil
		}
		return number{kind: kindUint}, nil
	case json.Number:
		return parseNumber(v.String(), dt)
	case float64:
		return number{kind: kindFloat, f: v}, nil
	case string:
		if mapped, ok := enums[v]; ok {
			return number{kind: kindFloat, f: mapped}, nil
		}
		return parseNumber(v, dt)
	}
	return number{}, errIncompatible
}

var (
	errFractional  = errors.New("value has a fractional component")
	errOutOfBounds = errors.New("value is out of bounds")
)

// two63 and two64 are the exclusive float64 bounds of int64 and uint64.
const (
	two63 = 9223372036854775808.0
	two64 = 18446744073709551616.0
)

func (n number) int64() (int64, error) {
	switch n.kind {
	case kindInt:
		return n.i, nil
	case kindUint:
		if n.u > math.MaxInt64 {
			return 0, errOutOfBounds
		}
		return int64(n.u), nil
	}
	if n.f != math.Trunc(n.f) {
		return 0, errFractional
	}
	if math.IsNaN(n.f) || n.f < -two63 || n.f >= two63 {
		return 0, errOutOfBounds
	}
	return int64(n.f), nil
}

func (n number) uint64() (uint64, error) {
	switch n.kind {
	case kindInt:
		return 0, errOutOfBounds
	case kindUint:
		return n.u, nil
	}
	if n.f != math.Trunc(n.f) {
		return 0, errFractional
	}
	if math.IsNaN(n.f) || n.f < 0 || n.f >= two64 {
		return 0, errOutOfBounds
	}
	return uint64(n.f), nil
}

func (n number) float64() float64 {
	switch n.kind {
	case kindInt:
		return float64(n.i)
	case kindUint:
		return float64(n.u)
	}
	return n.f
}

func appendNumber(dst []byte, dt telem.DataType, n number) ([]byte, error) {
	switch dt {
	case telem.Float64T:
		return telem.ByteOrder.AppendUint64(dst, math.Float64bits(n.float64())), nil
	case telem.Float32T:
		return telem.ByteOrder.AppendUint32(
			dst,
			math.Float32bits(float32(n.float64())),
		), nil
	case telem.Int64T, telem.Int32T, telem.Int16T, telem.Int8T:
		i, err := n.int64()
		if err != nil {
			return dst, err
		}
		bits := uint(dt.Density()) * 8
		if shifted := i >> (bits - 1); shifted != 0 && shifted != -1 {
			return dst, errOutOfBounds
		}
		return appendUint(dst, dt, uint64(i)), nil
	case telem.Uint64T, telem.Uint32T, telem.Uint16T, telem.Uint8T:
		u, err := n.uint64()
		if err != nil {
			return dst, err
		}
		if bits := uint(dt.Density()) * 8; bits < 64 && u>>bits != 0 {
			return dst, errOutOfBounds
		}
		return appendUint(dst, dt, u), nil
	}
	return dst, errIncompatible
}

// appendUint appends the low bytes of v at the width of dt. Two's complement makes
// this correct for signed types that already passed their bounds check.
func appendUint(dst []byte, dt telem.DataType, v uint64) []byte {
	switch dt.Density() {
	case telem.Bit64:
		return telem.ByteOrder.AppendUint64(dst, v)
	case telem.Bit32:
		return telem.ByteOrder.AppendUint32(dst, uint32(v))
	case telem.Bit16:
		return telem.ByteOrder.AppendUint16(dst, uint16(v))
	}
	return append(dst, byte(v))
}

func toTimeStamp(value any, format TimeFormat) (telem.TimeStamp, error) {
	var text string
	switch v := value.(type) {
	case json.Number:
		text = v.String()
	case float64:
		return scaleTimeStamp(number{kind: kindFloat, f: v}, format)
	case string:
		if format == ISO8601 {
			return parseISO8601(v)
		}
		text = v
	default:
		return 0, errIncompatible
	}
	n, err := parseNumber(text, telem.Int64T)
	if err != nil {
		return 0, err
	}
	return scaleTimeStamp(n, format)
}

func scaleTimeStamp(n number, format TimeFormat) (telem.TimeStamp, error) {
	if format == ISO8601 {
		return 0, errors.New("numeric values cannot be converted with ISO 8601 format")
	}
	scale := format.nanosPerUnit()
	if n.kind == kindFloat {
		nanos := n.f * float64(scale)
		if math.IsNaN(nanos) || nanos < -two63 || nanos >= two63 {
			return 0, errOutOfBounds
		}
		return telem.TimeStamp(nanos), nil
	}
	i, err := n.int64()
	if err != nil {
		return 0, err
	}
	if i > math.MaxInt64/scale || i < math.MinInt64/scale {
		return 0, errOutOfBounds
	}
	return telem.TimeStamp(i * scale), nil
}

// parseISO8601 parses an RFC 3339 timestamp. It also accepts a lowercase "t" or a
// space as the date and time separator and a lowercase "z", which RFC 3339 §5.6
// permits and time.Parse rejects.
func parseISO8601(text string) (telem.TimeStamp, error) {
	normalized := []byte(text)
	if len(normalized) > 10 && (normalized[10] == 't' || normalized[10] == ' ') {
		normalized[10] = 'T'
	}
	if last := len(normalized) - 1; last >= 0 && normalized[last] == 'z' {
		normalized[last] = 'Z'
	}
	t, err := time.Parse(time.RFC3339Nano, string(normalized))
	if err != nil {
		return 0, errors.Newf("not a valid ISO 8601 timestamp: %s", err.Error())
	}
	return telem.NewTimeStamp(t), nil
}

// Type is the JSON type that a sample converts to.
type Type uint8

const (
	// Number is a JSON number.
	Number Type = iota
	// String is a JSON string.
	String
	// Boolean is a JSON boolean.
	Boolean
)

// ParseType parses the config form of a Type: "number", "string", or "boolean". It
// returns an error wrapping validate.ErrValidation for any other text.
func ParseType(text string) (Type, error) {
	switch text {
	case "number":
		return Number, nil
	case "string":
		return String, nil
	case "boolean":
		return Boolean, nil
	}
	return Number, errors.Wrapf(
		validate.ErrValidation,
		`unknown JSON type %q: expected "number", "string", or "boolean"`,
		text,
	)
}

// String implements fmt.Stringer.
func (t Type) String() string {
	switch t {
	case String:
		return "string"
	case Boolean:
		return "boolean"
	}
	return "number"
}

// ZeroValue returns the zero value of t: 0, "", or false.
func ZeroValue(t Type) any {
	switch t {
	case String:
		return ""
	case Boolean:
		return false
	}
	return 0
}

// ReverseEnumMap maps numeric sample values to string labels in a JSON document.
type ReverseEnumMap map[float64]string

// CheckFromSample reports whether FromSample can convert samples of dt to target. It
// returns an error wrapping ErrConversion when it cannot: a string sample converts
// only to String, and timestamps and other non-numeric types do not convert.
func CheckFromSample(dt telem.DataType, target Type) error {
	if dt == telem.StringT {
		if target == String {
			return nil
		}
	} else if dt != telem.TimestampT && SupportsSampleTarget(dt) {
		return nil
	}
	return errors.Wrapf(ErrConversion, "cannot convert %s to a JSON %s", dt, target)
}

// FromSample converts one sample of dt to a value that encoding/json marshals as
// target. sample is the series encoding of the sample, as returned by Series.At.
//
// A numeric sample converts to a number, to a boolean that is true when the sample is
// not zero, or to a string. The string is the label in enums when one matches, and
// the shortest decimal text of the number otherwise. enums applies only when target
// is String.
//
// FromSample returns an error wrapping ErrConversion under the conditions of
// CheckFromSample, and for a NaN or infinite sample, which JSON cannot represent.
func FromSample(
	dt telem.DataType,
	sample []byte,
	target Type,
	enums ReverseEnumMap,
) (any, error) {
	if err := CheckFromSample(dt, target); err != nil {
		return nil, err
	}
	if dt == telem.StringT {
		return string(sample), nil
	}
	n := readNumber(dt, sample)
	if n.kind == kindFloat && (math.IsNaN(n.f) || math.IsInf(n.f, 0)) {
		return nil, errors.Wrapf(
			ErrConversion, "cannot convert %v to JSON: value is not finite", n.f,
		)
	}
	switch target {
	case Boolean:
		return n.float64() != 0, nil
	case String:
		if label, ok := enums[n.float64()]; ok {
			return label, nil
		}
		return n.text(dt), nil
	}
	switch n.kind {
	case kindInt:
		return n.i, nil
	case kindUint:
		return n.u, nil
	}
	return n.f, nil
}

// readNumber decodes a fixed-density numeric sample of dt.
func readNumber(dt telem.DataType, sample []byte) number {
	switch dt {
	case telem.Float64T:
		return number{
			kind: kindFloat,
			f:    math.Float64frombits(telem.ByteOrder.Uint64(sample)),
		}
	case telem.Float32T:
		return number{
			kind: kindFloat,
			f:    float64(math.Float32frombits(telem.ByteOrder.Uint32(sample))),
		}
	case telem.Int64T:
		return signedNumber(int64(telem.ByteOrder.Uint64(sample)))
	case telem.Int32T:
		return signedNumber(int64(int32(telem.ByteOrder.Uint32(sample))))
	case telem.Int16T:
		return signedNumber(int64(int16(telem.ByteOrder.Uint16(sample))))
	case telem.Int8T:
		return signedNumber(int64(int8(sample[0])))
	case telem.Uint64T:
		return number{kind: kindUint, u: telem.ByteOrder.Uint64(sample)}
	case telem.Uint32T:
		return number{kind: kindUint, u: uint64(telem.ByteOrder.Uint32(sample))}
	case telem.Uint16T:
		return number{kind: kindUint, u: uint64(telem.ByteOrder.Uint16(sample))}
	}
	return number{kind: kindUint, u: uint64(sample[0])}
}

func signedNumber(i int64) number {
	if i >= 0 {
		return number{kind: kindUint, u: uint64(i)}
	}
	return number{kind: kindInt, i: i}
}

// text returns the shortest decimal text that parses back to the sample.
func (n number) text(dt telem.DataType) string {
	switch n.kind {
	case kindInt:
		return strconv.FormatInt(n.i, 10)
	case kindUint:
		return strconv.FormatUint(n.u, 10)
	}
	bitSize := 64
	if dt == telem.Float32T {
		bitSize = 32
	}
	return strconv.FormatFloat(n.f, 'g', -1, bitSize)
}

// FromTimeStamp converts ts to a value that encoding/json marshals in format. ISO8601
// gives a UTC RFC 3339 string with no trailing fractional zeros. The numeric formats
// give a json.Number that holds the exact decimal count of their unit.
func FromTimeStamp(ts telem.TimeStamp, format TimeFormat) any {
	if format == ISO8601 {
		return time.Unix(0, int64(ts)).UTC().Format(time.RFC3339Nano)
	}
	scale := format.nanosPerUnit()
	whole, frac := int64(ts)/scale, int64(ts)%scale
	if frac == 0 {
		return json.Number(strconv.FormatInt(whole, 10))
	}
	sign := ""
	if ts < 0 {
		sign, whole, frac = "-", -whole, -frac
	}
	// Pad the fraction to the digit count of the unit, then drop trailing zeros.
	digits := strconv.FormatInt(scale+frac, 10)[1:]
	return json.Number(
		sign + strconv.FormatInt(whole, 10) + "." + strings.TrimRight(digits, "0"),
	)
}
