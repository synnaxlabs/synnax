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
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// multiplexors holds the multiplexors that select the fields of one binary message.
type multiplexors struct {
	// all holds each multiplexor after the multiplexor that selects it.
	all []multiplexor
	// end is the number of payload bytes that the multiplexors need.
	end int
}

// multiplexor is a field that selects which other fields a payload holds.
type multiplexor struct {
	key       icd.FieldKey
	name      string
	bits      bitRange
	signed    bool
	condition condition
}

// condition selects a field when its multiplexor holds one of values.
type condition struct {
	// mux indexes multiplexors.all, or is -1 for a field that every payload holds.
	mux    int
	values []int64
}

// resolve returns the condition that selects f, and adds the multiplexors that it
// reads. fields holds every field of the message of f. It returns an error wrapping
// validate.ErrValidation when a multiplexor does not exist, is a float field, selects
// on no values, or is in a cycle of multiplexors.
func (m *multiplexors) resolve(
	f icd.BinaryBaseField,
	fields map[icd.FieldKey]icd.BinaryField,
) (condition, error) {
	return m.resolveAt(f, fields, 0)
}

// resolveAt is resolve for a field with depth multiplexors below it in its chain.
func (m *multiplexors) resolveAt(
	f icd.BinaryBaseField,
	fields map[icd.FieldKey]icd.BinaryField,
	depth int,
) (condition, error) {
	if f.Multiplexor == nil {
		return condition{mux: -1}, nil
	}
	if len(f.MultiplexValues) == 0 {
		return condition{}, errors.Wrap(
			validate.ErrValidation, "multiplex values are empty",
		)
	}
	c := condition{values: make([]int64, len(f.MultiplexValues))}
	for i, v := range f.MultiplexValues {
		c.values[i] = int64(v)
	}
	key := *f.Multiplexor
	if c.mux = slices.IndexFunc(m.all, func(mux multiplexor) bool {
		return mux.key == key
	}); c.mux >= 0 {
		return c, nil
	}
	field, ok := fields[key]
	if !ok {
		return condition{}, errors.Wrapf(
			validate.ErrValidation, "multiplexor %s does not exist", key,
		)
	}
	b := field.Base()
	if _, float := field.Variant.(icd.FloatBinaryField); float {
		return condition{}, errors.Wrapf(
			validate.ErrValidation, "multiplexor %s is a float field", b.Name,
		)
	}
	// A multiplexor joins all only after the ones above it, so a cycle never ends in
	// all. A chain longer than the fields is a cycle.
	if depth >= len(fields) {
		return condition{}, errors.Wrap(
			validate.ErrValidation, "multiplexors form a cycle",
		)
	}
	bits, err := newBitRange(b)
	if err != nil {
		return condition{}, errors.Wrapf(err, "multiplexor %s", b.Name)
	}
	above, err := m.resolveAt(b, fields, depth+1)
	if err != nil {
		return condition{}, err
	}
	_, signed := field.Variant.(icd.IntegerBinaryField)
	c.mux = len(m.all)
	m.all = append(m.all, multiplexor{
		key:       key,
		name:      b.Name,
		bits:      bits,
		signed:    signed,
		condition: above,
	})
	m.end = max(m.end, bits.end)
	return c, nil
}

// selected reports whether raw, the value of each multiplexor in the order of all,
// selects c.
func (m *multiplexors) selected(c condition, raw []int64) bool {
	for c.mux >= 0 {
		if !slices.Contains(c.values, raw[c.mux]) {
			return false
		}
		c = m.all[c.mux].condition
	}
	return true
}
