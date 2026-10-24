// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"math"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/x/set"
)

const (
	maxStandardCANID = 0x7FF
	maxExtendedCANID = 0x1FFFFFFF
	maxCANLength     = 8
	maxCANFDLength   = 64
	maxBitLength     = 64
)

// Check returns every problem in the ICD: the rules its schema declares and the rules
// that span entries and fields, such as unique names, references between entries and
// fields, and the limits of each bus. A save refuses an ICD with problems. Entries and
// fields with no key do not count as duplicates, as a save assigns their keys.
func Check(ic ICD) library.Problems {
	c := checker{problems: library.Problems{}}
	c.problems.AddError(ic.Validate())
	enums := make(set.Set[EntryKey])
	for _, e := range ic.Entries {
		if en, ok := e.Variant.(EnumEntry); ok {
			enums.Add(en.Key)
		}
	}
	keys := make(set.Set[EntryKey], len(ic.Entries))
	names := make(set.Set[string], len(ic.Entries))
	for i, e := range ic.Entries {
		path := []string{"entries", strconv.Itoa(i)}
		if e.Variant == nil {
			c.addf(path, "kind is required")
			continue
		}
		base := e.Base()
		if base.Key != uuid.Nil() && keys.Contains(base.Key) {
			c.addf(pathOf(path, "key"), "duplicate entry key %s", base.Key)
		}
		keys.Add(base.Key)
		if base.Name != "" && names.Contains(base.Name) {
			c.addf(pathOf(path, "name"), "duplicate entry name %q", base.Name)
		}
		names.Add(base.Name)
		switch v := e.Variant.(type) {
		case EnumEntry:
			c.checkEnum(path, v)
		case MessageEntry:
			c.checkMessage(path, v, enums)
		}
	}
	return c.problems
}

type checker struct{ problems library.Problems }

func (c *checker) addf(path []string, format string, args ...any) {
	c.problems.Addf(path, format, args...)
}

func pathOf(parent []string, segments ...string) []string {
	return append(
		append(make([]string, 0, len(parent)+len(segments)), parent...),
		segments...,
	)
}

func (c *checker) checkEnum(path []string, e EnumEntry) {
	values := make(set.Set[int32], len(e.Values))
	names := make(set.Set[string], len(e.Values))
	for i, ev := range e.Values {
		p := pathOf(path, "values", strconv.Itoa(i))
		if values.Contains(ev.Value) {
			c.addf(pathOf(p, "value"), "duplicate value %d", ev.Value)
		}
		values.Add(ev.Value)
		if ev.Name != "" && names.Contains(ev.Name) {
			c.addf(pathOf(p, "name"), "duplicate name %q", ev.Name)
		}
		names.Add(ev.Name)
	}
}

func (c *checker) checkMessage(path []string, m MessageEntry, enums set.Set[EntryKey]) {
	if m.Period != nil && *m.Period <= 0 {
		c.addf(pathOf(path, "period"), "period must be positive")
	}
	path = pathOf(path, "payload")
	var (
		bases = m.Payload.FieldBases()
		// binary holds the binary fields by key, and unencoded the indices of text
		// fields with no encoding, whose bases are zero values.
		binary    = make(map[FieldKey]BinaryField)
		unencoded = make(set.Set[int])
		isText    bool
	)
	switch p := m.Payload.Variant.(type) {
	case BinaryPayload:
		for i, f := range p.Fields {
			c.checkBinaryField(pathOf(path, "fields", strconv.Itoa(i)), p, f)
			binary[f.Key] = f
		}
	case TextPayload:
		isText = true
		for i, f := range p.Fields {
			if f.Variant == nil {
				c.addf(pathOf(path, "fields", strconv.Itoa(i)), "encoding is required")
				unencoded.Add(i)
			}
		}
	case nil:
		c.addf(path, "format is required")
		return
	}
	keys := make(set.Set[FieldKey], len(bases))
	names := make(set.Set[string], len(bases))
	for i, b := range bases {
		if unencoded.Contains(i) {
			continue
		}
		p := pathOf(path, "fields", strconv.Itoa(i))
		if b.Key != uuid.Nil() && keys.Contains(b.Key) {
			c.addf(pathOf(p, "key"), "duplicate field key %s", b.Key)
		}
		keys.Add(b.Key)
		if b.Name != "" && names.Contains(b.Name) {
			c.addf(pathOf(p, "name"), "duplicate field name %q", b.Name)
		}
		names.Add(b.Name)
		c.checkScaling(p, b)
		if b.Enumeration != nil && !enums.Contains(*b.Enumeration) {
			c.addf(pathOf(p, "enumeration"), "the enumeration is not in this ICD")
		}
		if b.Multiplexor == nil {
			continue
		}
		if isText {
			c.addf(pathOf(p, "multiplexor"), "text fields cannot be multiplexed")
			continue
		}
		c.checkMultiplexor(p, b, binary)
	}
	if p, ok := m.Payload.Variant.(BinaryPayload); ok {
		c.checkIdentifier(path, p, binary)
	}
}

func (c *checker) checkScaling(path []string, b BaseField) {
	if math.IsNaN(b.Scale) || math.IsInf(b.Scale, 0) || b.Scale == 0 {
		c.addf(pathOf(path, "scale"), "scale must be finite and non-zero")
	}
	if math.IsNaN(b.Offset) || math.IsInf(b.Offset, 0) {
		c.addf(pathOf(path, "offset"), "offset must be finite")
	}
	finite := func(name string, v *float64) bool {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
			c.addf(pathOf(path, name), "%s must be finite", name)
			return false
		}
		return v != nil
	}
	if minOK, maxOK := finite("min", b.Min), finite("max", b.Max); minOK && maxOK &&
		*b.Min > *b.Max {
		c.addf(pathOf(path, "max"), "max must be greater than or equal to min")
	}
}

func (c *checker) checkMultiplexor(
	path []string,
	b BaseField,
	binary map[FieldKey]BinaryField,
) {
	mux, ok := binary[*b.Multiplexor]
	switch {
	case *b.Multiplexor == b.Key:
		c.addf(pathOf(path, "multiplexor"), "a field cannot multiplex itself")
	case !ok:
		c.addf(pathOf(path, "multiplexor"), "the multiplexor is not in this message")
	case mux.Float:
		c.addf(pathOf(path, "multiplexor"), "a multiplexor must be an integer field")
	case inCycle(b.Key, binary):
		c.addf(pathOf(path, "multiplexor"), "multiplexors form a cycle")
	}
	if len(b.MultiplexValues) == 0 {
		c.addf(
			pathOf(path, "multiplex_values"),
			"at least one value is required when the field has a multiplexor",
		)
	}
	if !ok || mux.Float || !validBitLength(mux.BitLength) {
		return
	}
	for i, v := range b.MultiplexValues {
		if !fits(v, mux) {
			c.addf(
				pathOf(path, "multiplex_values", strconv.Itoa(i)),
				"value %d does not fit the multiplexor %q",
				v,
				mux.Name,
			)
		}
	}
}

// inCycle reports whether following multiplexors from the field with the given key
// leads back to it.
func inCycle(key FieldKey, binary map[FieldKey]BinaryField) bool {
	cur := key
	for range binary {
		f, ok := binary[cur]
		if !ok || f.Multiplexor == nil {
			return false
		}
		if cur = *f.Multiplexor; cur == key {
			return true
		}
	}
	return false
}

func (c *checker) checkIdentifier(
	path []string,
	p BinaryPayload,
	binary map[FieldKey]BinaryField,
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
			c.addf(
				pathOf(idPath, "id"),
				"id 0x%X exceeds the maximum 0x%X",
				id.ID,
				limit,
			)
		}
		if id.BitrateSwitched && !id.Fd {
			c.addf(
				pathOf(idPath, "bitrate_switched"),
				"only a CAN FD frame can switch bitrates",
			)
		}
		if maxLength := canLength(id); p.Length != nil && *p.Length > maxLength {
			c.addf(
				pathOf(path, "length"),
				"length %d exceeds the %d bytes a CAN frame carries",
				*p.Length,
				maxLength,
			)
		}
	case FieldIdentifier:
		f, ok := binary[id.Field]
		switch {
		case !ok:
			c.addf(
				pathOf(idPath, "field"),
				"the identifier field is not in this message",
			)
		case f.Float:
			c.addf(pathOf(idPath, "field"), "the identifier must be an integer field")
		case validBitLength(f.BitLength) && !fits(id.Value, f):
			c.addf(
				pathOf(idPath, "value"),
				"value %d does not fit the field %q",
				id.Value,
				f.Name,
			)
		}
	case nil:
		c.addf(idPath, "type is required")
	}
}

func (c *checker) checkBinaryField(path []string, p BinaryPayload, f BinaryField) {
	if !validBitLength(f.BitLength) {
		return
	}
	if f.Float && f.BitLength != 32 && f.BitLength != 64 {
		c.addf(pathOf(path, "bit_length"), "a float field must be 32 or 64 bits")
	}
	if f.Float && f.Signed {
		c.addf(pathOf(path, "signed"), "a float field cannot also be signed")
	}
	if p.Length != nil {
		if lastByte(f) >= int(*p.Length) {
			c.addf(
				pathOf(path, "start_bit"),
				"field extends past the %d-byte payload",
				*p.Length,
			)
		}
		return
	}
	if p.Identifier == nil {
		return
	}
	if id, ok := p.Identifier.Variant.(CanIdentifier); ok &&
		lastByte(f) >= int(canLength(id)) {
		c.addf(
			pathOf(path, "start_bit"),
			"field extends past the %d bytes a CAN frame carries",
			canLength(id),
		)
	}
}

func validBitLength(n uint8) bool { return n >= 1 && n <= maxBitLength }

// canLength returns the most bytes a frame with the given identifier carries.
func canLength(id CanIdentifier) uint16 {
	if id.Fd {
		return maxCANFDLength
	}
	return maxCANLength
}

// fits reports whether the integer field f can hold v.
func fits(v int32, f BinaryField) bool {
	n := int64(f.BitLength)
	if !f.Signed {
		return v >= 0 && (n >= 31 || int64(v) < int64(1)<<n)
	}
	if n >= 32 {
		return true
	}
	return int64(v) >= -(int64(1)<<(n-1)) && int64(v) < int64(1)<<(n-1)
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
