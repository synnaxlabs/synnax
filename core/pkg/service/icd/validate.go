// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

const (
	maxStandardCANID = 0x7FF
	maxExtendedCANID = 0x1FFFFFFF
	maxCANLength     = 8
	maxCANFDLength   = 64
	maxBitLength     = 64
)

// validator accumulates the path-scoped errors of one library.
type validator struct{ err error }

func (v *validator) addf(path []string, format string, args ...any) {
	v.err = errors.Join(v.err, validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		path...,
	))
}

func pathOf(parent []string, segments ...string) []string {
	return append(
		append(make([]string, 0, len(parent)+len(segments)), parent...),
		segments...,
	)
}

// validateEntries checks the rules that span entries and fields, which the generated
// per-field validation cannot see: unique keys and names, references to other entries
// and fields, and the layout constraints of each bus.
func validateEntries(entries []Entry) error {
	var (
		v     validator
		keys  = make(set.Set[EntryKey], len(entries))
		names = make(set.Set[string], len(entries))
		enums = make(set.Set[EntryKey])
	)
	for _, e := range entries {
		if en, ok := e.Variant.(EnumEntry); ok {
			enums.Add(en.Key)
		}
	}
	for i, e := range entries {
		path := []string{"entries", strconv.Itoa(i)}
		if e.Variant == nil {
			v.addf(path, "kind is required")
			continue
		}
		base := e.Base()
		if keys.Contains(base.Key) {
			v.addf(pathOf(path, "key"), "duplicate entry key %s", base.Key)
		}
		keys.Add(base.Key)
		if names.Contains(base.Name) {
			v.addf(pathOf(path, "name"), "duplicate entry name %q", base.Name)
		}
		names.Add(base.Name)
		switch variant := e.Variant.(type) {
		case EnumEntry:
			v.validateEnum(path, variant)
		case MessageEntry:
			v.validateMessage(path, variant, enums)
		}
	}
	return v.err
}

func (v *validator) validateEnum(path []string, e EnumEntry) {
	values := make(set.Set[int32], len(e.Values))
	names := make(set.Set[string], len(e.Values))
	for i, ev := range e.Values {
		p := pathOf(path, "values", strconv.Itoa(i))
		if values.Contains(ev.Value) {
			v.addf(pathOf(p, "value"), "duplicate value %d", ev.Value)
		}
		values.Add(ev.Value)
		if names.Contains(ev.Name) {
			v.addf(pathOf(p, "name"), "duplicate name %q", ev.Name)
		}
		names.Add(ev.Name)
	}
}

func (v *validator) validateMessage(
	path []string,
	m MessageEntry,
	enums set.Set[EntryKey],
) {
	path = pathOf(path, "payload")
	var (
		bases  = m.Payload.FieldBases()
		fields = make(map[FieldKey]BaseField, len(bases))
		names  = make(set.Set[string], len(bases))
		floats = make(set.Set[FieldKey])
		// unencoded holds the indices of text fields with no encoding, whose bases are
		// zero values.
		unencoded = make(set.Set[int])
	)
	switch p := m.Payload.Variant.(type) {
	case BinaryPayload:
		for i, f := range p.Fields {
			v.validateBinaryField(pathOf(path, "fields", strconv.Itoa(i)), p, f)
			if f.Float {
				floats.Add(f.Key)
			}
		}
	case TextPayload:
		for i, f := range p.Fields {
			if f.Variant == nil {
				v.addf(pathOf(path, "fields", strconv.Itoa(i)), "encoding is required")
				unencoded.Add(i)
			}
		}
	case nil:
		v.addf(path, "format is required")
		return
	}
	for i, b := range bases {
		if unencoded.Contains(i) {
			continue
		}
		p := pathOf(path, "fields", strconv.Itoa(i))
		if _, dup := fields[b.Key]; dup {
			v.addf(pathOf(p, "key"), "duplicate field key %s", b.Key)
		}
		fields[b.Key] = b
		if names.Contains(b.Name) {
			v.addf(pathOf(p, "name"), "duplicate field name %q", b.Name)
		}
		names.Add(b.Name)
	}
	if p, ok := m.Payload.Variant.(BinaryPayload); ok {
		v.validateIdentifier(path, p, fields)
	}
	for i, b := range bases {
		if unencoded.Contains(i) {
			continue
		}
		p := pathOf(path, "fields", strconv.Itoa(i))
		if b.Enumeration != nil && !enums.Contains(*b.Enumeration) {
			v.addf(
				pathOf(p, "enumeration"),
				"no enum entry with key %s in this library",
				*b.Enumeration,
			)
		}
		if b.Multiplexor == nil {
			continue
		}
		_, ok := fields[*b.Multiplexor]
		switch {
		case *b.Multiplexor == b.Key:
			v.addf(pathOf(p, "multiplexor"), "a field cannot multiplex itself")
		case !ok:
			v.addf(
				pathOf(p, "multiplexor"),
				"no field with key %s in this message",
				*b.Multiplexor,
			)
		case floats.Contains(*b.Multiplexor):
			v.addf(pathOf(p, "multiplexor"), "a multiplexor must be an integer field")
		}
		if len(b.MultiplexValues) == 0 {
			v.addf(
				pathOf(p, "multiplex_values"),
				"at least one value is required when the field has a multiplexor",
			)
		}
	}
}

func (v *validator) validateIdentifier(
	path []string,
	p BinaryPayload,
	fields map[FieldKey]BaseField,
) {
	if p.Identifier == nil {
		return
	}
	idPath := pathOf(path, "identifier")
	switch id := p.Identifier.Variant.(type) {
	case CanIdentifier:
		limit := uint32(maxStandardCANID)
		if id.Extended {
			limit = maxExtendedCANID
		}
		if id.ID > limit {
			v.addf(
				pathOf(idPath, "id"),
				"id 0x%X exceeds the maximum 0x%X",
				id.ID,
				limit,
			)
		}
		maxLength := uint16(maxCANLength)
		if id.Fd {
			maxLength = maxCANFDLength
		}
		if p.Length != nil && *p.Length > maxLength {
			v.addf(
				pathOf(path, "length"),
				"length %d exceeds the %d bytes a CAN frame carries",
				*p.Length,
				maxLength,
			)
		}
	case FieldIdentifier:
		if _, ok := fields[id.Field]; !ok {
			v.addf(
				pathOf(idPath, "field"),
				"no field with key %s in this message",
				id.Field,
			)
		}
	case nil:
		v.addf(idPath, "type is required")
	}
}

func (v *validator) validateBinaryField(path []string, p BinaryPayload, f BinaryField) {
	if f.BitLength == 0 || f.BitLength > maxBitLength {
		v.addf(
			pathOf(path, "bit_length"),
			"bit_length must be between 1 and %d",
			maxBitLength,
		)
		return
	}
	if f.Float && f.BitLength != 32 && f.BitLength != 64 {
		v.addf(pathOf(path, "bit_length"), "a float field must be 32 or 64 bits")
	}
	if f.Float && f.Signed {
		v.addf(pathOf(path, "signed"), "a float field cannot also be signed")
	}
	if p.Length != nil && lastByte(f) >= int(*p.Length) {
		v.addf(
			pathOf(path, "start_bit"),
			"field extends past the %d-byte payload",
			*p.Length,
		)
	}
}

// lastByte returns the index of the last payload byte a binary field touches. A
// little-endian start bit is the field's least significant bit; a big-endian start bit
// is its most significant bit, and the field continues toward higher byte indices.
func lastByte(f BinaryField) int {
	start, length := int(f.StartBit), int(f.BitLength)
	if f.ByteOrder == ByteOrderBigEndian {
		msb := 8*(start/8) + 7 - start%8
		return (msb + length - 1) / 8
	}
	return (start + length - 1) / 8
}
