// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	"encoding/json/jsontext"
	"slices"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/validate"
)

// router finds the message that a document belongs to. Every value rule tests the value
// at one pointer, so one lookup on that value finds its message. Each present rule
// tests a pointer of its own.
type router struct {
	names []string
	// ruled is true when a message has a value rule.
	ruled    bool
	pointer  xjson.Pointer
	strings  map[string]int
	booleans map[bool]int
	integers map[integer]int
	floats   map[float64]int
	present  []presentRule
	// fallback is the message with no match rule, or -1 when every message has one.
	fallback int
	// text is scratch space for the text of a string value.
	text []byte
}

type presentRule struct {
	pointer xjson.Pointer
	message int
}

// integer is the value of an integer or unsigned match rule.
type integer struct {
	// bits holds the value in two's complement.
	bits     uint64
	negative bool
}

func signedInteger(v int64) integer { return integer{bits: uint64(v), negative: v < 0} }

func (k integer) float() float64 {
	if k.negative {
		return float64(int64(k.bits))
	}
	return float64(k.bits)
}

func (k integer) String() string {
	if k.negative {
		return strconv.FormatInt(int64(k.bits), 10)
	}
	return strconv.FormatUint(k.bits, 10)
}

func newRouter(messages []JSONMessage) (*router, error) {
	r := &router{
		names:    make([]string, len(messages)),
		strings:  make(map[string]int),
		booleans: make(map[bool]int),
		integers: make(map[integer]int),
		floats:   make(map[float64]int),
		fallback: -1,
	}
	var (
		// first is the first message with a value rule.
		first = -1
		// rounded holds an integer rule for each float that one rounds to.
		rounded = make(map[float64]integer)
	)
	for i, m := range messages {
		r.names[i] = m.Name
		if m.Payload.Match == nil {
			if r.fallback >= 0 {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					"messages %s and %s both have no match rule",
					messages[r.fallback].Name,
					m.Name,
				)
			}
			r.fallback = i
			continue
		}
		match := *m.Payload.Match
		pointer, err := xjson.ParsePointer(match.Base().Pointer)
		if err != nil {
			return nil, errors.Wrapf(err, "message %s: match", m.Name)
		}
		if _, ok := match.Variant.(icd.PresentJSONMatch); ok {
			r.present = append(r.present, presentRule{pointer: pointer, message: i})
			continue
		}
		if first < 0 {
			first, r.pointer = i, pointer
		} else if !slices.Equal(pointer, r.pointer) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s match at different pointers %q and %q",
				messages[first].Name,
				m.Name,
				messages[first].Payload.Match.Base().Pointer,
				match.Base().Pointer,
			)
		}
		taken, value, typed := r.claimMatch(match, i, rounded)
		if !typed {
			return nil, errors.Wrapf(
				validate.ErrValidation, "message %s: match type is required", m.Name,
			)
		}
		if taken >= 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s match the same value %s",
				messages[taken].Name,
				m.Name,
				value,
			)
		}
	}
	r.ruled = first >= 0
	// A present rule at or above the pointer of another rule meets every document that
	// the other rule meets.
	for k, p := range r.present {
		other, at := -1, xjson.Pointer(nil)
		if r.ruled && within(r.pointer, p.pointer) {
			other, at = first, r.pointer
		}
		for j, q := range r.present {
			if j != k && within(q.pointer, p.pointer) {
				other, at = q.message, q.pointer
			}
		}
		if other >= 0 {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s both match a value at %q",
				messages[min(other, p.message)].Name,
				messages[max(other, p.message)].Name,
				at.String(),
			)
		}
	}
	return r, nil
}

// within reports whether inner is outer or a pointer into the value at outer.
func within(inner, outer xjson.Pointer) bool {
	return len(outer) <= len(inner) && slices.Equal(outer, inner[:len(outer)])
}

// pointers returns the pointers that the rules test, in the order that find takes their
// values.
func (r *router) pointers() []xjson.Pointer {
	var pointers []xjson.Pointer
	if r.ruled {
		pointers = append(pointers, r.pointer)
	}
	for _, p := range r.present {
		pointers = append(pointers, p.pointer)
	}
	return pointers
}

// claimMatch maps the value of match to message. It returns the message that already
// matches that value, or -1, and the value as text. typed is false when match has no
// type. rounded holds an integer rule for each float that one rounds to.
func (r *router) claimMatch(
	match icd.JSONMatch,
	message int,
	rounded map[float64]integer,
) (taken int, value string, typed bool) {
	switch v := match.Variant.(type) {
	case icd.StringJSONMatch:
		return claim(r.strings, v.Value, message), strconv.Quote(v.Value), true
	case icd.BooleanJSONMatch:
		return claim(r.booleans, v.Value, message), strconv.FormatBool(v.Value), true
	case icd.IntegerJSONMatch:
		k := signedInteger(int64(v.Value))
		return r.claimInteger(k, message, rounded), k.String(), true
	case icd.UnsignedJSONMatch:
		k := integer{bits: uint64(v.Value)}
		return r.claimInteger(k, message, rounded), k.String(), true
	case icd.FloatJSONMatch:
		f := float64(v.Value)
		taken = claim(r.floats, f, message)
		if k, ok := rounded[f]; ok && taken < 0 {
			return r.integers[k], k.String(), true
		}
		return taken, strconv.FormatFloat(f, 'g', -1, 64), true
	}
	return 0, "", false
}

// claimInteger is claim for an integer rule, which also meets a float rule that it
// rounds to.
func (r *router) claimInteger(
	k integer,
	message int,
	rounded map[float64]integer,
) int {
	taken := claim(r.integers, k, message)
	if f, ok := r.floats[k.float()]; ok && taken < 0 {
		taken = f
	}
	if _, ok := rounded[k.float()]; !ok {
		rounded[k.float()] = k
	}
	return taken
}

// claim maps key to message in m. It returns the message that already held key, or -1.
func claim[K comparable](m map[K]int, key K, message int) int {
	if held, ok := m[key]; ok {
		return held
	}
	m[key] = message
	return -1
}

// find returns the message of a document whose values at the pointers of the router are
// values. A value is nil when the document holds none. ok is false when the document
// belongs to no message. find returns an error when the document meets two rules.
func (r *router) find(values []jsontext.Value) (message int, ok bool, err error) {
	if r.ruled {
		if message, ok, err = r.findValue(values[0]); err != nil {
			return 0, false, err
		}
		values = values[1:]
	}
	for i, p := range r.present {
		if values[i] == nil || values[i].Kind() == 'n' {
			continue
		}
		if ok {
			return 0, false, errors.Wrapf(
				validate.ErrValidation,
				"document matches messages %s and %s",
				r.names[min(message, p.message)],
				r.names[max(message, p.message)],
			)
		}
		message, ok = p.message, true
	}
	if !ok && r.fallback >= 0 {
		return r.fallback, true, nil
	}
	return message, ok, nil
}

// findValue returns the message of the value rule that value meets. value is nil when
// the document holds none.
func (r *router) findValue(value jsontext.Value) (message int, ok bool, err error) {
	switch value.Kind() {
	case '"':
		if r.text, err = jsontext.AppendUnquote(r.text[:0], value); err != nil {
			return 0, false, errors.Wrap(validate.ErrValidation, err.Error())
		}
		message, ok = r.strings[string(r.text)]
	case 't', 'f':
		message, ok = r.booleans[value.Kind() == 't']
	case '0':
		// With no number rules, any number is no match, even one that does not parse.
		if len(r.integers) == 0 && len(r.floats) == 0 {
			break
		}
		n, err := xjson.ParseNumber(value)
		if err != nil {
			return 0, false, errors.Wrap(err, "match")
		}
		if k, isInteger := integerOf(n); isInteger {
			message, ok = r.integers[k]
		}
		if !ok {
			message, ok = r.floats[n.Float64()]
		}
	}
	return message, ok, nil
}

// integerOf returns n as the value of an integer rule. ok is false when n is not an
// integer that an int64 or a uint64 holds.
func integerOf(n xjson.ParsedNumber) (integer, bool) {
	if v, ok := n.Int64(); ok {
		return signedInteger(v), true
	}
	v, ok := n.Uint64()
	return integer{bits: v}, ok
}
