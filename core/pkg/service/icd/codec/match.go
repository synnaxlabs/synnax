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

// router finds the message that a document belongs to. Every match rule tests the value
// at one pointer, so one lookup on that value finds the message.
type router struct {
	// ruled is true when a message has a match rule.
	ruled    bool
	pointer  xjson.Pointer
	strings  map[string]int
	booleans map[bool]int
	integers map[integer]int
	floats   map[float64]int
	// fallback is the message with no match rule, or -1 when every message has one.
	fallback int
	// text is scratch space for the text of a string value.
	text []byte
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
		strings:  make(map[string]int),
		booleans: make(map[bool]int),
		integers: make(map[integer]int),
		floats:   make(map[float64]int),
		fallback: -1,
	}
	var (
		// first is the first message with a match rule.
		first = -1
		// rounded holds an integer rule for each float that one rounds to.
		rounded = make(map[float64]integer)
	)
	for i, m := range messages {
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
	return r, nil
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

// find returns the message of a document whose value at the match pointer is value,
// which is nil when there is none. ok is false when the document belongs to no message.
func (r *router) find(value jsontext.Value) (message int, ok bool, err error) {
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
	if !ok && r.fallback >= 0 {
		return r.fallback, true, nil
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
