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
	"slices"
	"strconv"
	"strings"
	"uuid"

	xjson "github.com/synnaxlabs/x/json"
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

// Check returns every problem in the ICD: the rules its schema declares and the rules
// that span messages and fields, such as unique names, references between fields, and
// the limits of each bus. Problems do not stop a save. Messages and fields with no key
// do not count as duplicates, as a save assigns their keys.
func Check(ic ICD) validate.Problems {
	c := checker{Problems: validate.Problems{}}
	c.addSchema(ic.Validate())
	keys := make(set.Set[MessageKey], len(ic.Messages))
	names := make(set.Set[string], len(ic.Messages))
	for i, m := range ic.Messages {
		path := []string{"messages", strconv.Itoa(i)}
		if m.Key != uuid.Nil() && keys.Contains(m.Key) {
			c.Addf(pathOf(path, "key"), "duplicate message key %s", m.Key)
		}
		keys.Add(m.Key)
		if m.Name != "" && names.Contains(m.Name) {
			c.Addf(pathOf(path, "name"), "duplicate message name %q", m.Name)
		}
		names.Add(m.Name)
		c.checkMessage(path, m)
	}
	return c.Problems
}

type checker struct{ validate.Problems }

// addSchema adds each problem that err holds as a sentence that names its field, such
// as "bit length must be less than or equal to 64".
func (c *checker) addSchema(err error) {
	var schema validate.Problems
	schema.AddError(err)
	for _, p := range schema {
		field := p.Path[strings.LastIndexByte(p.Path, '.')+1:]
		field = strings.ReplaceAll(field, "_", " ")
		switch {
		case p.Message == "required":
			p.Message = field + " is required"
		case strings.HasPrefix(p.Message, "must "):
			p.Message = field + " " + p.Message
		default:
			p.Message = field + " is not valid"
		}
		c.Problems = append(c.Problems, p)
	}
}

func pathOf(parent []string, segments ...string) []string {
	return append(
		append(make([]string, 0, len(parent)+len(segments)), parent...),
		segments...,
	)
}

func (c *checker) checkMessage(path []string, m Message) {
	path = pathOf(path, "payload")
	var (
		bases = m.Payload.FieldBases()
		// untyped holds the indices of fields with no encoding or type, whose bases are
		// zero values, and unscaled the indices of fields that hold no number.
		untyped  = make(set.Set[int])
		unscaled = make(set.Set[int])
	)
	switch p := m.Payload.Variant.(type) {
	case BinaryPayload:
		binary := make(map[FieldKey]BinaryField, len(p.Fields))
		for i, f := range p.Fields {
			c.checkBinaryField(pathOf(path, "fields", strconv.Itoa(i)), p, f)
			binary[f.Key] = f
		}
		for i, f := range p.Fields {
			if f.Multiplexor != nil {
				c.checkMultiplexor(pathOf(path, "fields", strconv.Itoa(i)), f, binary)
			}
		}
		c.checkOverlaps(pathOf(path, "fields"), p.Fields)
		c.checkIdentifier(path, p, binary)
	case TextPayload:
		for i, f := range p.Fields {
			fieldPath := pathOf(path, "fields", strconv.Itoa(i))
			switch v := f.Variant.(type) {
			case nil:
				c.Addf(fieldPath, "encoding is required")
				untyped.Add(i)
			case TaggedTextField:
				if v.Tag == "" {
					c.Addf(pathOf(fieldPath, "tag"), "tag is required")
				}
			}
		}
	case JSONPayload:
		c.checkJSONFields(pathOf(path, "fields"), p, untyped, unscaled)
	case nil:
		c.Addf(path, "format is required")
		return
	}
	keys := make(set.Set[FieldKey], len(bases))
	names := make(set.Set[string], len(bases))
	for i, b := range bases {
		if untyped.Contains(i) {
			continue
		}
		p := pathOf(path, "fields", strconv.Itoa(i))
		if b.Key != uuid.Nil() && keys.Contains(b.Key) {
			c.Addf(pathOf(p, "key"), "duplicate field key %s", b.Key)
		}
		keys.Add(b.Key)
		if b.Name != "" && names.Contains(b.Name) {
			c.Addf(pathOf(p, "name"), "duplicate field name %q", b.Name)
		}
		names.Add(b.Name)
		if unscaled.Contains(i) {
			c.checkUnscaled(p, b)
		} else {
			c.checkScaling(p, b)
		}
	}
}

// checkJSONFields adds the indices of fields with no type to untyped, and of fields
// that hold no number to unscaled.
func (c *checker) checkJSONFields(
	path []string,
	p JSONPayload,
	untyped, unscaled set.Set[int],
) {
	type read struct {
		name    string
		pointer xjson.Pointer
	}
	reads := make([]read, 0, len(p.Fields))
	for i, f := range p.Fields {
		fieldPath := pathOf(path, strconv.Itoa(i))
		switch f.Variant.(type) {
		case nil:
			c.Addf(fieldPath, "type is required")
			untyped.Add(i)
			continue
		case NumberJSONField:
		default:
			unscaled.Add(i)
		}
		b := f.Base()
		pointer, err := xjson.ParsePointer(b.Pointer)
		if err != nil {
			c.Addf(pathOf(fieldPath, "pointer"), "invalid JSON Pointer %q", b.Pointer)
			continue
		}
		for _, r := range reads {
			if msg, ok := nested(pointer, r.pointer); ok {
				c.Addf(pathOf(fieldPath, "pointer"), msg, b.Pointer, r.name)
				break
			}
		}
		reads = append(reads, read{name: b.Name, pointer: pointer})
	}
}

// nested returns a problem format, taking a pointer and a field name, when a and b
// lead to the same value or one leads into the value of the other.
func nested(a, b xjson.Pointer) (string, bool) {
	switch {
	case slices.Equal(a, b):
		return "pointer %q duplicates the pointer of field %q", true
	case len(a) > len(b) && slices.Equal(a[:len(b)], b):
		return "pointer %q is inside the value of field %q", true
	case len(b) > len(a) && slices.Equal(b[:len(a)], a):
		return "pointer %q holds the value of field %q", true
	}
	return "", false
}

// checkUnscaled finds the scaling set on a field that holds no number.
func (c *checker) checkUnscaled(path []string, b BaseField) {
	if b.Scale != 1 {
		c.Addf(pathOf(path, "scale"), "only a number field can scale")
	}
	if b.Offset != 0 {
		c.Addf(pathOf(path, "offset"), "only a number field can have an offset")
	}
	if b.Min != nil {
		c.Addf(pathOf(path, "min"), "only a number field can have a min")
	}
	if b.Max != nil {
		c.Addf(pathOf(path, "max"), "only a number field can have a max")
	}
}

func (c *checker) checkScaling(path []string, b BaseField) {
	if math.IsNaN(b.Scale) || math.IsInf(b.Scale, 0) || b.Scale == 0 {
		c.Addf(pathOf(path, "scale"), "scale must be finite and non-zero")
	}
	if math.IsNaN(b.Offset) || math.IsInf(b.Offset, 0) {
		c.Addf(pathOf(path, "offset"), "offset must be finite")
	}
	finite := func(name string, v *float64) bool {
		if v != nil && (math.IsNaN(*v) || math.IsInf(*v, 0)) {
			c.Addf(pathOf(path, name), "%s must be finite", name)
			return false
		}
		return v != nil
	}
	if minOK, maxOK := finite("min", b.Min), finite("max", b.Max); minOK && maxOK &&
		*b.Min > *b.Max {
		c.Addf(pathOf(path, "max"), "max must be greater than or equal to min")
	}
}

func (c *checker) checkMultiplexor(
	path []string,
	b BinaryField,
	binary map[FieldKey]BinaryField,
) {
	mux, ok := binary[*b.Multiplexor]
	switch {
	case *b.Multiplexor == b.Key:
		c.Addf(pathOf(path, "multiplexor"), "a field cannot multiplex itself")
	case !ok:
		c.Addf(pathOf(path, "multiplexor"), "the multiplexor is not in this message")
	case mux.Float:
		c.Addf(pathOf(path, "multiplexor"), "a multiplexor must be an integer field")
	case inCycle(b.Key, binary):
		c.Addf(pathOf(path, "multiplexor"), "multiplexors form a cycle")
	}
	if len(b.MultiplexValues) == 0 {
		c.Addf(
			pathOf(path, "multiplex_values"),
			"at least one value is required when the field has a multiplexor",
		)
	}
	if !ok || mux.Float || !validBitLength(mux.BitLength) {
		return
	}
	for i, v := range b.MultiplexValues {
		if !fits(v, mux) {
			c.Addf(
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
			c.Addf(
				pathOf(idPath, "id"),
				"id 0x%X exceeds the maximum 0x%X",
				id.ID,
				limit,
			)
		}
		if id.BitrateSwitched && !id.Fd {
			c.Addf(
				pathOf(idPath, "bitrate_switched"),
				"only a CAN FD frame can switch bitrates",
			)
		}
		if maxLength := canLength(id); p.Length != nil && *p.Length > maxLength {
			c.Addf(
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
			c.Addf(
				pathOf(idPath, "field"),
				"the identifier field is not in this message",
			)
		case f.Float:
			c.Addf(pathOf(idPath, "field"), "the identifier must be an integer field")
		case validBitLength(f.BitLength) && !fits(id.Value, f):
			c.Addf(
				pathOf(idPath, "value"),
				"value %d does not fit the field %q",
				id.Value,
				f.Name,
			)
		}
	case nil:
		c.Addf(idPath, "type is required")
	}
}

func (c *checker) checkBinaryField(path []string, p BinaryPayload, f BinaryField) {
	if !validBitLength(f.BitLength) {
		return
	}
	if f.Float && f.BitLength != 32 && f.BitLength != 64 {
		c.Addf(pathOf(path, "bit_length"), "a float field must be 32 or 64 bits")
	}
	if f.Float && f.Signed {
		c.Addf(pathOf(path, "signed"), "a float field cannot also be signed")
	}
	if p.Length != nil {
		if lastByte(f) >= int(*p.Length) {
			c.Addf(
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
		c.Addf(
			pathOf(path, "start_bit"),
			"field extends past the %d bytes a CAN frame carries",
			canLength(id),
		)
	}
}

// checkOverlaps finds binary fields that read the same bits and can both be present in
// one frame. Each field is reported against the first earlier field it overlaps.
func (c *checker) checkOverlaps(path []string, fields []BinaryField) {
	// ids numbers the payload bits the fields read, so readers can be slices.
	ids := make(map[int]int)
	var all, plain readers
	found := make([]int, len(fields))
	bits := make([][]int, len(fields))
	groups := make(map[muxValue][]int)
	for j, f := range fields {
		found[j] = noField
		if !validBitLength(f.BitLength) {
			continue
		}
		for _, b := range coveredBits(f) {
			id, ok := ids[b]
			if !ok {
				id = len(ids)
				ids[b] = id
				all, plain = append(all, noField), append(plain, noField)
			}
			bits[j] = append(bits[j], id)
		}
		if f.Multiplexor == nil {
			found[j] = all.claim(j, bits[j])
			plain.claim(j, bits[j])
			continue
		}
		found[j] = plain.earliest(bits[j])
		all.claim(j, bits[j])
		for _, v := range f.MultiplexValues {
			k := muxValue{multiplexor: *f.Multiplexor, value: v}
			if g := groups[k]; len(g) == 0 || g[len(g)-1] != j {
				groups[k] = append(g, j)
			}
		}
	}
	// Fields under one multiplexor coexist only in the frames of a value they share.
	shared := slices.Repeat(readers{noField}, len(ids))
	for _, g := range groups {
		if len(g) < 2 {
			continue
		}
		for _, j := range g {
			found[j] = min(found[j], shared.claim(j, bits[j]))
		}
		for _, j := range g {
			shared.release(bits[j])
		}
	}
	for j, i := range found {
		if i != noField {
			c.Addf(
				pathOf(path, strconv.Itoa(j), "start_bit"),
				"field overlaps the bits of field %q",
				fields[i].Name,
			)
		}
	}
}

// noField marks the absence of a field index.
const noField = math.MaxInt

type muxValue struct {
	multiplexor FieldKey
	value       int32
}

// readers holds the first field that reads each bit.
type readers []int

// claim records field i as a reader of bits and returns the first earlier field that
// reads one of them, or noField.
func (r readers) claim(i int, bits []int) int {
	found := noField
	for _, b := range bits {
		if r[b] == noField {
			r[b] = i
		} else {
			found = min(found, r[b])
		}
	}
	return found
}

// earliest returns the first field that reads one of bits, or noField.
func (r readers) earliest(bits []int) int {
	found := noField
	for _, b := range bits {
		found = min(found, r[b])
	}
	return found
}

func (r readers) release(bits []int) {
	for _, b := range bits {
		r[b] = noField
	}
}

// coveredBits returns the payload bits the field reads, each numbered 8*byte + bit,
// where bit 0 is the least significant bit of its byte.
func coveredBits(f BinaryField) []int {
	start, length := int(f.StartBit), int(f.BitLength)
	bits := make([]int, 0, length)
	if f.ByteOrder != ByteOrderBigEndian {
		for i := start; i < start+length; i++ {
			bits = append(bits, i)
		}
		return bits
	}
	msb := 8*(start/8) + 7 - start%8
	for i := msb; i < msb+length; i++ {
		bits = append(bits, 8*(i/8)+7-i%8)
	}
	return bits
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
