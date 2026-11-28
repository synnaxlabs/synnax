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
	"encoding/binary"
	"math"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/payload"
	xstrconv "github.com/synnaxlabs/x/strconv"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// BinaryEncoder encodes samples into the payload of one binary message. It is not safe
// for concurrent use.
type BinaryEncoder struct {
	name   string
	fields []binaryEncodeField
	muxes  multiplexors
	// muxFields holds the index in fields of each multiplexor.
	muxFields []int
	// raw holds the raw value of each multiplexor in the last encode.
	raw []int64
	// bound holds the index in fields of the field of each binding.
	bound  []int
	length int
}

type binaryEncodeField struct {
	base     icd.BaseField
	bits     bitRange
	encoding encoding
	// scaled is true when the field's scale or offset changes its value.
	scaled bool
	// kind is the kind of the samples of the field's channel.
	kind payload.Kind
	// narrow is true for float32 samples.
	narrow bool
	// binding is the index of the field's sample, or -1 for a field that takes its
	// default.
	binding int
	// fallback holds the raw bits of the field's default when defaulted is true.
	fallback  uint64
	defaulted bool
	condition condition
}

// NewBinaryEncoder returns an encoder of the payloads of m. Each binding names a field
// that takes a sample, in sample order, and every other field takes its default. The
// identifier field takes the identifier's value, and a CAN identifier is not encoded.
// It returns an error wrapping validate.ErrValidation when m has an invalid identifier,
// a binding names no field, a field has two bindings, a binding names the identifier
// field, a field's layout, default, or multiplexor is invalid, a bound field has a zero
// scale, a field's channel does not hold numbers, a scaled field's channel does not
// hold floats, a field that the multiplexors can select has no binding and no default,
// or a fixed multiplexor value never selects a bound field.
func NewBinaryEncoder(m BinaryMessage) (*BinaryEncoder, error) {
	e, err := newBinaryEncoder(m)
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", m.Name)
	}
	return e, nil
}

func newBinaryEncoder(m BinaryMessage) (*BinaryEncoder, error) {
	bound, err := bindingIndexes(m.Payload.Fields, binaryBase, m.Bindings)
	if err != nil {
		return nil, err
	}
	byKey := make(map[icd.FieldKey]icd.BinaryField, len(m.Payload.Fields))
	for _, f := range m.Payload.Fields {
		byKey[binaryKey(f)] = f
	}
	var id *icd.FieldIdentifier
	if m.Payload.Identifier != nil {
		switch v := m.Payload.Identifier.Variant.(type) {
		case icd.FieldIdentifier:
			id = &v
		case icd.CanIdentifier:
			// A CAN task writes the identifier in the frame, not the payload.
		default:
			panic(errors.Newf("codec: unknown identifier %T", v))
		}
	}
	if id != nil {
		if _, err = identifierBits(*id, m.Payload); err != nil {
			return nil, errors.Wrap(err, "identifier")
		}
		if _, ok := bound[id.Field]; ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"field %s holds the identifier, so it cannot take a channel",
				byKey[id.Field].Base().Name,
			)
		}
	}
	e := &BinaryEncoder{
		name:   m.Name,
		fields: make([]binaryEncodeField, len(m.Payload.Fields)),
		bound:  make([]int, len(m.Bindings)),
	}
	for i, f := range m.Payload.Fields {
		b := f.Base()
		binding, ok := bound[b.Key]
		if !ok {
			binding = -1
		}
		if err = e.addField(i, f, binding, m, byKey, id); err != nil {
			return nil, errors.Wrapf(err, "field %s", b.Name)
		}
	}
	if m.Payload.Length != nil {
		e.length = int(*m.Payload.Length)
	}
	if err = e.checkSelection(); err != nil {
		return nil, err
	}
	return e, nil
}

func binaryBase(f icd.BinaryField) icd.BaseField { return f.Base().BaseField }

// addField sets field i of e to f, the field of binding, or -1 for none. id is the
// identifier of m, or nil.
func (e *BinaryEncoder) addField(
	i int,
	f icd.BinaryField,
	binding int,
	m BinaryMessage,
	byKey map[icd.FieldKey]icd.BinaryField,
	id *icd.FieldIdentifier,
) error {
	var dt telem.DataType
	if binding >= 0 {
		dt = m.Bindings[binding].DataType
		e.bound[binding] = i
	}
	field, err := newBinaryEncodeField(f, binding, dt)
	if err == nil {
		err = checkLength(field.bits, m.Payload)
	}
	if err != nil {
		return err
	}
	if id != nil && id.Field == binaryKey(f) {
		field.fallback, field.defaulted = uint64(int64(id.Value)), true
	} else if field.condition, err = e.muxes.resolve(f.Base(), byKey); err != nil {
		return err
	}
	e.length = max(e.length, field.bits.end)
	e.fields[i] = field
	return nil
}

// checkSelection refuses a bound field that a fixed multiplexor value never selects,
// and a field that the multiplexors can select with no binding and no default. A
// multiplexor is fixed when it has no binding and a default.
func (e *BinaryEncoder) checkSelection() error {
	fixed := make([]*int64, len(e.muxes.all))
	e.muxFields = make([]int, len(e.muxes.all))
	for i, mux := range e.muxes.all {
		e.muxFields[i] = slices.IndexFunc(e.fields, func(f binaryEncodeField) bool {
			return f.base.Key == mux.key
		})
		if f := e.fields[e.muxFields[i]]; f.binding < 0 && f.defaulted {
			fixed[i] = new(int64(f.fallback))
		}
	}
	e.raw = make([]int64, len(e.muxes.all))
	for _, f := range e.fields {
		never := e.neverSelects(f.condition, fixed)
		switch {
		case never != nil && f.binding >= 0:
			return errors.Wrapf(
				validate.ErrValidation,
				"field %s takes a channel, but multiplexor %s never selects it",
				f.base.Name,
				never.name,
			)
		case never == nil && f.binding < 0 && !f.defaulted:
			return errors.Wrapf(
				validate.ErrValidation,
				"field %s: needs a channel or a default",
				f.base.Name,
			)
		}
	}
	return nil
}

// neverSelects returns the multiplexor whose fixed value never selects c, or nil. fixed
// holds the fixed value of each multiplexor, or nil.
func (e *BinaryEncoder) neverSelects(c condition, fixed []*int64) *multiplexor {
	for ; c.mux >= 0; c = e.muxes.all[c.mux].condition {
		if v := fixed[c.mux]; v != nil && !slices.Contains(c.values, *v) {
			return &e.muxes.all[c.mux]
		}
	}
	return nil
}

// newBinaryEncodeField returns the encode field of f, the field of binding, whose
// channel holds dt. binding is -1, and dt is unused, for a field with no binding.
func newBinaryEncodeField(
	f icd.BinaryField,
	binding int,
	dt telem.DataType,
) (binaryEncodeField, error) {
	b := f.Base()
	bits, err := newBitRange(b)
	if err != nil {
		return binaryEncodeField{}, err
	}
	e, err := encodingOf(f)
	if err != nil {
		return binaryEncodeField{}, err
	}
	fallback, defaulted, err := defaultRaw(f, e)
	if err != nil {
		return binaryEncodeField{}, errors.Wrap(err, "default")
	}
	field := binaryEncodeField{
		base:      b.BaseField,
		bits:      bits,
		encoding:  e,
		scaled:    b.Scale != 1 || b.Offset != 0,
		binding:   binding,
		fallback:  fallback,
		defaulted: defaulted,
		condition: condition{mux: -1},
	}
	if binding < 0 {
		return field, nil
	}
	if err = icd.CheckScale(b.Scale); err != nil {
		return binaryEncodeField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	if err = CheckWrite(e.kind(), field.scaled, dt); err != nil {
		return binaryEncodeField{}, err
	}
	field.kind, _ = payload.KindOf(dt)
	field.narrow = dt == telem.Float32T
	return field, nil
}

// Encode appends the payload of samples to dst. samples holds one sample for each
// binding, in binding order, and a nil sample takes the default of its field. A field
// that the multiplexors do not select is not written, and bits that no field holds are
// zero. Encode does not check the min and max of a field. It returns an error wrapping
// validate.ErrValidation, naming the message and the field, when a selected field has
// no sample and no default, or a sample does not fit its field.
func (e *BinaryEncoder) Encode(dst []byte, samples [][]byte) ([]byte, error) {
	start := len(dst)
	dst = slices.Grow(dst, e.length)[:start+e.length]
	p := dst[start:]
	clear(p)
	for i, mux := range e.muxes.all {
		e.raw[i] = 0
		if !e.muxes.selected(mux.condition, e.raw) {
			continue
		}
		f := &e.fields[e.muxFields[i]]
		raw, err := f.value(samples)
		if err != nil {
			return nil, e.wrap(err, f)
		}
		e.raw[i] = int64(raw)
	}
	for i := range e.fields {
		f := &e.fields[i]
		if !e.muxes.selected(f.condition, e.raw) {
			continue
		}
		raw, err := f.value(samples)
		if err != nil {
			return nil, e.wrap(err, f)
		}
		f.bits.write(p, raw)
	}
	return dst, nil
}

// Ready reports whether each field that the multiplexors select in samples has a sample
// or a default, so Encode has a value for it. samples is as in Encode.
func (e *BinaryEncoder) Ready(samples [][]byte) bool {
	for i, mux := range e.muxes.all {
		e.raw[i] = 0
		if !e.muxes.selected(mux.condition, e.raw) {
			continue
		}
		f := &e.fields[e.muxFields[i]]
		if !f.hasValue(samples) {
			return false
		}
		raw, err := f.value(samples)
		// Encode reports err.
		if err != nil {
			return true
		}
		e.raw[i] = int64(raw)
	}
	for i := range e.fields {
		f := &e.fields[i]
		if !f.hasValue(samples) && e.muxes.selected(f.condition, e.raw) {
			return false
		}
	}
	return true
}

// Check returns the error that Encode returns when binding i holds sample and its field
// is selected, or an error wrapping validate.ErrValidation when the raw value of sample
// decodes outside the min and max of the field of binding i. When the sample or the
// field is float32, a value that decodes to a limit at float32 precision is at that
// limit.
func (e *BinaryEncoder) Check(i int, sample []byte) error {
	f := &e.fields[e.bound[i]]
	raw, err := f.raw(sample)
	if err == nil {
		err = f.checkRaw(raw, f.narrow || f.encoding == encodingFloat32)
	}
	if err != nil {
		return e.wrap(err, f)
	}
	return nil
}

func (e *BinaryEncoder) wrap(err error, f *binaryEncodeField) error {
	return errors.Wrapf(err, "message %s: field %s", e.name, f.base.Name)
}

func (f *binaryEncodeField) hasValue(samples [][]byte) bool {
	return (f.binding >= 0 && samples[f.binding] != nil) || f.defaulted
}

// value returns the raw bits of the field's sample in samples, or of its default.
func (f *binaryEncodeField) value(samples [][]byte) (uint64, error) {
	if f.binding >= 0 && samples[f.binding] != nil {
		return f.raw(samples[f.binding])
	}
	if f.defaulted {
		return f.fallback, nil
	}
	return 0, errors.Wrap(validate.ErrValidation, "no value and no default")
}

// raw returns the raw bits of sample. It returns an error wrapping
// validate.ErrValidation when sample does not fit the field.
func (f *binaryEncodeField) raw(sample []byte) (uint64, error) {
	switch f.kind {
	case payload.KindInteger:
		return integerRaw(f, signed(sample))
	case payload.KindUnsigned, payload.KindBoolean:
		return integerRaw(f, unsigned(sample))
	case payload.KindFloat:
		if f.narrow {
			return f.floatRaw(
				float64(math.Float32frombits(binary.LittleEndian.Uint32(sample))),
			)
		}
		return f.floatRaw(math.Float64frombits(binary.LittleEndian.Uint64(sample)))
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	panic(errors.Newf("codec: %s channels never write binary fields", f.kind))
}

// integerRaw returns the raw bits of v, an integer sample of f. It returns an error
// wrapping validate.ErrValidation when v does not fit f.
func integerRaw[T int64 | uint64](f *binaryEncodeField, v T) (uint64, error) {
	switch f.encoding {
	case encodingFloat32:
		return uint64(math.Float32bits(float32(v))), nil
	case encodingFloat64:
		return math.Float64bits(float64(v)), nil
	case encodingUnsigned, encodingSigned:
	}
	n := f.bits.width
	negative, magnitude := v < 0, uint64(v)
	if negative {
		magnitude = -magnitude
	}
	fits := !negative && magnitude <= ^uint64(0)>>(64-n)
	if f.encoding == encodingSigned {
		hi := ^uint64(0) >> (65 - n)
		fits = magnitude <= hi || (negative && magnitude == hi+1)
	}
	if !fits {
		return 0, errors.Wrapf(
			validate.ErrValidation, "%d is out of range for %d bits", v, n,
		)
	}
	return uint64(v), nil
}

// floatRaw returns the raw bits of v, a float sample, after its scale and offset come
// off. A field of integer bits takes the nearest integer.
func (f *binaryEncodeField) floatRaw(v float64) (uint64, error) {
	unscaled := v
	if f.scaled {
		unscaled = (v - f.base.Offset) / f.base.Scale
	}
	switch f.encoding {
	case encodingFloat64:
		return math.Float64bits(unscaled), nil
	case encodingFloat32:
		if xmath.Finite(unscaled) && math.Abs(unscaled) > math.MaxFloat32 {
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"%s overflows float32",
				xstrconv.FormatFloat(v, 64),
			)
		}
		return uint64(math.Float32bits(float32(unscaled))), nil
	case encodingUnsigned, encodingSigned:
	}
	n := f.bits.width
	r := math.Round(unscaled)
	lo, hi := 0.0, math.Ldexp(1, n)
	if f.encoding == encodingSigned {
		lo, hi = -math.Ldexp(1, n-1), math.Ldexp(1, n-1)
	}
	if math.IsNaN(r) || r < lo || r >= hi {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%s is out of range for %d bits",
			xstrconv.FormatFloat(v, 64),
			n,
		)
	}
	if f.encoding == encodingSigned {
		return uint64(int64(r)), nil
	}
	return uint64(r), nil
}

// checkRaw returns an error wrapping validate.ErrValidation when raw, the bits of a
// sample, decode outside the min and max of the field. When narrow, a value that
// decodes to a limit at float32 precision is at that limit.
func (f *binaryEncodeField) checkRaw(raw uint64, narrow bool) error {
	v := f.encoding.value(raw)
	decoded := v*f.base.Scale + f.base.Offset
	// atLimit moves a value that is a limit at float32 precision onto that limit.
	if narrow && xmath.Finite(decoded) && atLimit(f.base, true, decoded) != decoded {
		return nil
	}
	return icd.CheckRawRange(f.base, v)
}
