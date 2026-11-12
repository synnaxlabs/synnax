// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"fmt"
	"math"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Channel is the kind and the options of a channel field, which writes the samples of
// a channel into a payload. Only the entries of Kind are set, and TimeFormat only on
// KindTimestamp.
type Channel struct {
	Kind       Kind
	Float      []FloatEntry
	Integer    []IntegerEntry
	Unsigned   []UnsignedEntry
	String     []StringEntry
	Boolean    []BooleanEntry
	TimeFormat TimeFormat
}

// Field returns how c writes the samples of a channel of data type dt. It returns an
// error wrapping validate.ErrValidation when dt does not allow c's kind, or a string
// entry is not the text of a value of dt.
func (c Channel) Field(dt telem.DataType) (xjson.Field, error) {
	if kinds := ChannelKinds(dt); !slices.Contains(kinds, c.Kind) {
		return xjson.Field{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s channel cannot be written as %s, only as %v",
			dt, c.Kind, kinds,
		)
	}
	f := xjson.Field{DataType: dt, Type: xjson.String}
	var err error
	switch c.Kind {
	case KindFloat:
		f.Type = xjson.Number
		f.Enums = lo.Map(c.Float, func(e FloatEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, float64(e.Value))
		})
	case KindInteger:
		f.Type = xjson.Number
		f.Enums = lo.Map(c.Integer, func(e IntegerEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, int64(e.Value))
		})
	case KindUnsigned:
		f.Type = xjson.Number
		f.Enums = lo.Map(c.Unsigned, func(e UnsignedEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, uint64(e.Value))
		})
	case KindString:
		f.Enums = make([]xjson.Enum, len(c.String))
		for i, e := range c.String {
			if f.Enums[i], err = textEnum(e, dt); err != nil {
				return xjson.Field{}, err
			}
		}
	case KindBoolean:
		f.Type = xjson.Boolean
		f.Enums = lo.Map(c.Boolean, func(e BooleanEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, e.Value)
		})
	case KindTimestamp:
		if f.TimeFormat, err = xjson.ParseTimeFormat(string(c.TimeFormat)); err != nil {
			return xjson.Field{}, err
		}
	case KindJSON:
		f.Type = xjson.JSON
	}
	return f, nil
}

// textEnum returns the enum of e, whose value is the text of a sample of dt.
func textEnum(e StringEntry, dt telem.DataType) (xjson.Enum, error) {
	var err error
	switch kind, _ := KindOf(dt); kind {
	case KindFloat:
		var v float64
		if v, err = strconv.ParseFloat(e.Value, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	case KindInteger:
		var v int64
		if v, err = strconv.ParseInt(e.Value, 10, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	case KindUnsigned, KindBoolean:
		var v uint64
		if v, err = strconv.ParseUint(e.Value, 10, 64); err == nil {
			return xjson.NewEnum(e.Label, v), nil
		}
	default:
		return xjson.NewEnum(e.Label, e.Value), nil
	}
	return xjson.Enum{}, errors.Wrapf(
		validate.ErrValidation,
		"enum label %q: %q is not a %s value",
		e.Label, e.Value, dt,
	)
}

// Check returns an error wrapping validate.ErrValidation when c does not agree with a
// channel of data type dt: dt does not allow c's kind, an entry does not fit dt, or two
// entries match one sample of dt.
func (c Channel) Check(dt telem.DataType) error {
	f, err := c.Field(dt)
	if err != nil {
		return err
	}
	if _, err = xjson.NewEncoder(f); err != nil {
		return err
	}
	return xjson.CheckEnums(f)
}

// Resolve checks c as Check does, and returns it with its float entries narrowed to a
// float32 channel of data type dt.
func (c Channel) Resolve(dt telem.DataType) (Channel, error) {
	if err := c.Check(dt); err != nil {
		return Channel{}, err
	}
	if c.Kind == KindFloat && dt == telem.Float32T {
		c.Float = lo.Map(c.Float, func(e FloatEntry, _ int) FloatEntry {
			e.Value = Float(float32(e.Value))
			return e
		})
	}
	return c, nil
}

// Validate returns an error wrapping validate.ErrValidation when two entries of c have
// one stored value. Two NaN entries, or 0 and -0, collide.
func (c Channel) Validate() error {
	var dup int
	switch c.Kind {
	case KindFloat:
		dup = duplicate(c.Float, func(a, b FloatEntry) bool {
			x, y := float64(a.Value), float64(b.Value)
			return x == y || (math.IsNaN(x) && math.IsNaN(y))
		})
	case KindInteger:
		dup = duplicate(c.Integer, func(a, b IntegerEntry) bool {
			return a.Value == b.Value
		})
	case KindUnsigned:
		dup = duplicate(c.Unsigned, func(a, b UnsignedEntry) bool {
			return a.Value == b.Value
		})
	case KindString:
		dup = duplicate(c.String, func(a, b StringEntry) bool {
			return a.Value == b.Value
		})
	case KindBoolean:
		dup = duplicate(c.Boolean, func(a, b BooleanEntry) bool {
			return a.Value == b.Value
		})
	default:
		dup = -1
	}
	if dup >= 0 {
		return errors.Wrapf(
			validate.ErrValidation,
			"enum_values[%d]: value is already labeled",
			dup,
		)
	}
	return nil
}

func duplicate[E any](entries []E, equal func(a, b E) bool) int {
	for i, e := range entries {
		if slices.ContainsFunc(
			entries[:i],
			func(prev E) bool { return equal(prev, e) },
		) {
			return i
		}
	}
	return -1
}

// Reconcile returns c changed to agree with a channel of data type dt, and describes
// each change. A kind dt does not allow becomes KindOf(dt). Entries convert between
// numeric kinds when they stay exact, and drop on other kinds. An entry that does not
// fit dt, or whose value matches an earlier entry's there, drops. c is returned as it
// is when dt is not a known data type.
func (c Channel) Reconcile(dt telem.DataType) (Channel, []string) {
	kinds := ChannelKinds(dt)
	if len(kinds) == 0 {
		return c, nil
	}
	var changes []string
	if !slices.Contains(kinds, c.Kind) {
		changes = append(
			changes,
			fmt.Sprintf("changed the kind from %s to %s", c.Kind, kinds[0]),
		)
		c = c.convert(kinds[0], &changes)
	}
	c.Float = keep(c, dt, c.Float, &changes,
		func(e FloatEntry) string { return e.Label },
		func(c Channel, es []FloatEntry) Channel { c.Float = es; return c })
	c.Integer = keep(c, dt, c.Integer, &changes,
		func(e IntegerEntry) string { return e.Label },
		func(c Channel, es []IntegerEntry) Channel { c.Integer = es; return c })
	c.Unsigned = keep(c, dt, c.Unsigned, &changes,
		func(e UnsignedEntry) string { return e.Label },
		func(c Channel, es []UnsignedEntry) Channel { c.Unsigned = es; return c })
	c.String = keep(c, dt, c.String, &changes,
		func(e StringEntry) string { return e.Label },
		func(c Channel, es []StringEntry) Channel { c.String = es; return c })
	c.Boolean = keep(c, dt, c.Boolean, &changes,
		func(e BooleanEntry) string { return e.Label },
		func(c Channel, es []BooleanEntry) Channel { c.Boolean = es; return c })
	return c, changes
}

// keep returns the entries of c that fit a channel of data type dt, in order, where
// with sets entries on c. It describes each drop in changes.
func keep[E any](
	c Channel,
	dt telem.DataType,
	entries []E,
	changes *[]string,
	label func(E) string,
	with func(Channel, []E) Channel,
) []E {
	if len(entries) == 0 {
		return entries
	}
	kept := make([]E, 0, len(entries))
	for _, e := range entries {
		if err := with(c, append(slices.Clip(kept), e)).Check(dt); err != nil {
			*changes = append(
				*changes,
				fmt.Sprintf("dropped enum entry %q: %v", label(e), err),
			)
			continue
		}
		kept = append(kept, e)
	}
	return kept
}

// convert returns c as kind. Entries convert between numeric kinds when they stay
// exact, and drop otherwise. It describes each drop in changes.
func (c Channel) convert(kind Kind, changes *[]string) Channel {
	out := Channel{Kind: kind}
	drop := func(n number) {
		*changes = append(*changes, fmt.Sprintf(
			"dropped enum entry %q, whose value %v is not exact as %s",
			n.label, n.value, kind,
		))
	}
	from := c.numbers()
	switch kind {
	case KindFloat:
		for _, n := range from {
			if f, ok := toFloat64(n.value); ok {
				out.Float = append(
					out.Float,
					FloatEntry{Label: n.label, Value: Float(f)},
				)
			} else {
				drop(n)
			}
		}
	case KindInteger:
		for _, n := range from {
			if i, ok := toInt64(n.value); ok {
				out.Integer = append(
					out.Integer,
					IntegerEntry{Label: n.label, Value: Integer(i)},
				)
			} else {
				drop(n)
			}
		}
	case KindUnsigned:
		for _, n := range from {
			if u, ok := toUint64(n.value); ok {
				out.Unsigned = append(
					out.Unsigned,
					UnsignedEntry{Label: n.label, Value: Unsigned(u)},
				)
			} else {
				drop(n)
			}
		}
	default:
		if len(from) > 0 || len(c.String) > 0 || len(c.Boolean) > 0 {
			*changes = append(*changes, "dropped the enum entries")
		}
		if kind == KindTimestamp {
			out.TimeFormat = lo.CoalesceOrEmpty(c.TimeFormat, TimeFormatISO8601)
		}
	}
	return out
}

type number struct {
	label string
	// value is a float64, int64, or uint64.
	value any
}

func (c Channel) numbers() []number {
	var out []number
	for _, e := range c.Float {
		out = append(out, number{label: e.Label, value: float64(e.Value)})
	}
	for _, e := range c.Integer {
		out = append(out, number{label: e.Label, value: int64(e.Value)})
	}
	for _, e := range c.Unsigned {
		out = append(out, number{label: e.Label, value: uint64(e.Value)})
	}
	return out
}

// maxExactFloat is the largest magnitude up to which a float64 holds every integer.
const maxExactFloat = 1 << 53

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), x >= -maxExactFloat && x <= maxExactFloat
	case uint64:
		return float64(x), x <= maxExactFloat
	default:
		return v.(float64), true
	}
}

func toInt64(v any) (int64, bool) {
	switch x := v.(type) {
	case float64:
		ok := x == math.Trunc(x) && x >= math.MinInt64 && x < math.MaxInt64
		return int64(x), ok
	case uint64:
		return int64(x), x <= math.MaxInt64
	default:
		return v.(int64), true
	}
}

func toUint64(v any) (uint64, bool) {
	switch x := v.(type) {
	case float64:
		return uint64(x), x == math.Trunc(x) && x >= 0 && x < math.MaxUint64
	case int64:
		return uint64(x), x >= 0
	default:
		return v.(uint64), true
	}
}
