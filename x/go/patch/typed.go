// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

import (
	"encoding/json/v2"
	"math"
	"slices"
	"unicode/utf8"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// Begin returns the cursor at the start of the action's path, for the Patch method of a
// typed document. It returns an error wrapping validate.ErrValidation for an empty
// path, an insert or move at the document, a path that names __proto__, or an action
// with guards.
func Begin[A Action](a A) (Cursor, error) {
	if gs := a.guards(); len(gs) > 0 {
		return Cursor{}, errors.Wrapf(
			validate.ErrValidation,
			"%s: the guard names no discriminator on the path",
			format(gs[0].Path),
		)
	}
	c := a.cursor()
	if err := checkPath(c.path); err != nil {
		return Cursor{}, err
	}
	if c.Done() {
		if c.kind == kindSet {
			return Cursor{}, errors.Wrap(validate.ErrValidation, "the path is empty")
		}
		return Cursor{}, c.notList()
	}
	return c, nil
}

type kind uint8

const (
	kindSet kind = iota
	kindRemove
	kindInsert
	kindMove
)

// Cursor is a path action on its way down a typed document. A generated Patch method
// switches on Segment and passes Next to the field that the segment names.
type Cursor struct {
	value any
	path  []string
	depth int
	kind  kind
}

func (s Set) cursor() Cursor {
	return Cursor{kind: kindSet, path: s.Path, value: s.Value}
}

func (r Remove) cursor() Cursor {
	return Cursor{kind: kindRemove, path: slices.Concat(r.Path, []string{r.Key})}
}

func (ins Insert) cursor() Cursor {
	return Cursor{kind: kindInsert, path: ins.Path, value: ins.Value}
}

func (m Move) cursor() Cursor { return Cursor{kind: kindMove, path: m.Path} }

// Segment returns the path segment at the cursor. It panics when the cursor is done.
func (c Cursor) Segment() string { return c.path[c.depth] }

// Next returns the cursor at the next segment of the path.
func (c Cursor) Next() Cursor {
	c.depth++
	return c
}

// Done reports whether the cursor is at the end of the path, where the action applies.
func (c Cursor) Done() bool { return c.depth == len(c.path) }

// UnknownField returns an error wrapping validate.ErrValidation for a segment that
// names no editable field.
func (c Cursor) UnknownField() error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the path names no editable field",
		format(c.path[:c.depth+1]),
	)
}

// Value applies the action at c to a required field that the path ends at, or that the
// path cannot go into because it holds a value.
func Value[T any](dst *T, c Cursor) error {
	if !c.Done() {
		return intoValue(c.path[:c.depth])
	}
	return whole(dst, c)
}

// OptionalValue applies the action at c to an optional field that the path ends at, or
// that the path cannot go into because it holds a value or is absent. A remove or move
// through an absent field does nothing.
func OptionalValue[T any](dst **T, c Cursor) error {
	if c.Done() {
		return optional(dst, c)
	}
	if *dst != nil {
		return intoValue(c.path[:c.depth])
	}
	if c.kind == kindRemove || c.kind == kindMove {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the value is absent",
		format(c.path[:c.depth]),
	)
}

// whole applies the action at c to a required field that the path ends at.
func whole[T any](dst *T, c Cursor) error {
	switch c.kind {
	case kindSet:
		if c.value == nil {
			return required(c.path)
		}
		return decode(dst, c)
	case kindRemove:
		return required(c.path)
	default:
		return c.notList()
	}
}

// optional applies the action at c to an optional field that the path ends at.
func optional[T any](dst **T, c Cursor) error {
	switch c.kind {
	case kindSet:
		if c.value == nil {
			*dst = nil
			return nil
		}
		next := new(T)
		if err := decode(next, c); err != nil {
			return err
		}
		*dst = next
		return nil
	case kindRemove:
		*dst = nil
		return nil
	default:
		return c.notList()
	}
}

// decode stores the action's value in dst as a JSON round trip would, failing for a
// value that the round trip would reject or fill with zero values.
func decode[T any](dst *T, c Cursor) error {
	if setExact(dst, c.value) {
		return nil
	}
	b, err := json.Marshal(c.value)
	if err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: encoding value: %s",
			format(c.path),
			err,
		)
	}
	var next T
	if err = json.Unmarshal(b, &next, json.RejectUnknownMembers(true)); err != nil {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s: decoding value: %s",
			format(c.path),
			err,
		)
	}
	// Only an object can lack a member that its type holds.
	if b[0] == '{' {
		path, err := missingFrom(next, b)
		if err != nil {
			return errors.Wrapf(
				validate.ErrValidation,
				"%s: encoding value: %s",
				format(c.path),
				err,
			)
		}
		if path != nil {
			return required(slices.Concat(c.path, path))
		}
	}
	*dst = next
	return nil
}

// missingFrom returns the path of a member that the JSON form of v holds and the object
// b lacks, or nil when b holds every member.
func missingFrom(v any, b []byte) ([]string, error) {
	var got, want map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(encoded, &want); err != nil {
		return nil, err
	}
	return missing(want, got), nil
}

// maxExact bounds the integers whose float64 and JSON forms are both exact.
const maxExact = 1 << 53

// setExact stores v in dst and returns true when dst is a builtin type and v is the
// value that a JSON round trip into dst would produce. It returns false for every other
// value, which then takes the round trip.
func setExact[T any](dst *T, v any) bool {
	switch d := any(dst).(type) {
	case *string:
		s, ok := v.(string)
		if ok = ok && utf8.ValidString(s); ok {
			*d = s
		}
		return ok
	case *bool:
		b, ok := v.(bool)
		if ok {
			*d = b
		}
		return ok
	case *float64:
		f, ok := v.(float64)
		if ok = ok && !math.IsInf(f, 0) && !math.IsNaN(f); ok {
			*d = f
		}
		return ok
	case *int:
		return setInteger(d, v, math.MinInt, math.MaxInt)
	case *int8:
		return setInteger(d, v, math.MinInt8, math.MaxInt8)
	case *int16:
		return setInteger(d, v, math.MinInt16, math.MaxInt16)
	case *int32:
		return setInteger(d, v, math.MinInt32, math.MaxInt32)
	case *int64:
		return setInteger(d, v, math.MinInt64, math.MaxInt64)
	case *uint:
		return setInteger(d, v, 0, math.MaxUint)
	case *uint8:
		return setInteger(d, v, 0, math.MaxUint8)
	case *uint16:
		return setInteger(d, v, 0, math.MaxUint16)
	case *uint32:
		return setInteger(d, v, 0, math.MaxUint32)
	case *uint64:
		return setInteger(d, v, 0, math.MaxUint64)
	}
	return false
}

func setInteger[I types.Integer](dst *I, v any, lo, hi float64) bool {
	f, ok := v.(float64)
	if !ok || f != math.Trunc(f) || (f == 0 && math.Signbit(f)) ||
		f < max(lo, -maxExact) || f > min(hi, maxExact) {
		return false
	}
	*dst = I(f)
	return true
}

// intoValue returns the error for a path that goes into a field that only a whole set
// changes.
func intoValue(path []string) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the value takes only a whole set",
		format(path),
	)
}

func required(path []string) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"%s: the document requires a value",
		format(path),
	)
}

func (c Cursor) notList() error { return notList(c.path) }
