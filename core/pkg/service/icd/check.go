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

	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	xmath "github.com/synnaxlabs/x/math"
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

// Check returns every problem in the ICD as an error at the path of the bad value: the
// rules its schema declares and the rules that span messages and fields, such as unique
// names, references between fields, and the limits of each bus. Each error wraps
// validate.ErrValidation. Problems do not stop a save. Messages and fields with no key
// do not count as duplicates, as a save assigns their keys.
func Check(ic ICD) []validate.PathError {
	c := checker{problems: []validate.PathError{}}
	c.addSchema(ic.Validate())
	messages := newUniques("message", len(ic.Messages))
	for i, m := range ic.Messages {
		path := []string{"messages", strconv.Itoa(i)}
		c.checkUnique(path, messages, m.Key, m.Name)
		c.checkMessage(path, m, ic.Framing)
	}
	if ic.Framing != nil {
		c.checkSync(*ic.Framing)
	}
	return c.problems
}

type checker struct{ problems []validate.PathError }

func (c *checker) add(path []string, err error) {
	c.problems = append(
		c.problems,
		validate.PathError{Path: slices.Clone(path), Err: err},
	)
}

func (c *checker) addf(path []string, format string, args ...any) {
	c.add(path, validate.Errorf(format, args...))
}

// uniques holds the keys and names of the messages, or of the fields of one message,
// checked so far.
type uniques struct {
	noun  string
	keys  set.Set[uuid.UUID]
	names set.Set[string]
}

func newUniques(noun string, size int) uniques {
	return uniques{
		noun:  noun,
		keys:  make(set.Set[uuid.UUID], size),
		names: make(set.Set[string], size),
	}
}

// checkUnique adds a problem for a key or name that an earlier entry of u holds, then
// adds key and name to u. A nil key and an empty name never count, as a save assigns
// keys.
func (c *checker) checkUnique(path []string, u uniques, key uuid.UUID, name string) {
	if key != uuid.Nil() && u.keys.Contains(key) {
		c.addf(pathOf(path, "key"), "duplicate %s key %s", u.noun, key)
	}
	u.keys.Add(key)
	if name != "" && u.names.Contains(name) {
		c.addf(pathOf(path, "name"), "duplicate %s name %q", u.noun, name)
	}
	u.names.Add(name)
}

// addSchema adds each problem that err holds as a sentence that names its field, such
// as "bit length must be less than or equal to 64". err may join several errors, and an
// error that is not a validate.PathError is a problem with the whole ICD.
func (c *checker) addSchema(err error) {
	if err == nil {
		return
	}
	if members := errors.UnwrapMulti(err); members != nil {
		for _, m := range members {
			c.addSchema(m)
		}
		return
	}
	p, ok := errors.AsType[validate.PathError](err)
	if !ok {
		p = validate.PathError{Err: err}
	}
	var field string
	if len(p.Path) > 0 {
		field = strings.ReplaceAll(p.Path[len(p.Path)-1], "_", " ")
	}
	switch message := p.Err.Error(); {
	case message == "required":
		c.addf(p.Path, "%s is required", field)
	case strings.HasPrefix(message, "must "):
		c.addf(p.Path, "%s %s", field, message)
	default:
		c.addf(p.Path, "%s is not valid", field)
	}
}

func pathOf(parent []string, segments ...string) []string {
	return append(
		append(make([]string, 0, len(parent)+len(segments)), parent...),
		segments...,
	)
}

func (c *checker) checkMessage(path []string, m Message, framing *Framing) {
	if m.CycleTime != nil && *m.CycleTime <= 0 {
		c.addf(pathOf(path, "cycle_time"), "cycle time must be positive")
	}
	path = pathOf(path, "payload")
	switch p := m.Payload.Variant.(type) {
	case BinaryPayload:
		c.checkBinaryPayload(path, p)
		if framing != nil && p.Length != nil {
			c.checkFrameLength(pathOf(path, "length"), *p.Length, *framing)
		}
	case TextPayload:
		c.checkTextFields(path, p)
	case JSONPayload:
		if p.Match != nil {
			c.checkMatch(pathOf(path, "match"), *p.Match)
		}
		c.checkJSONFields(pathOf(path, "fields"), p)
	}
}

// checkBase checks the key, name, and scaling of a field with a variant. Only a number
// field can scale.
func (c *checker) checkBase(path []string, b BaseField, number bool, fields uniques) {
	c.checkUnique(path, fields, b.Key, b.Name)
	if number {
		c.checkScaling(path, b)
	} else {
		c.checkUnscaled(path, b)
	}
}

func (c *checker) checkBinaryPayload(path []string, p BinaryPayload) {
	binary := make(map[FieldKey]BinaryField, len(p.Fields))
	fields := newUniques("field", len(p.Fields))
	for i, f := range p.Fields {
		if f.Variant == nil {
			continue
		}
		fieldPath := pathOf(path, "fields", strconv.Itoa(i))
		c.checkBinaryField(fieldPath, p, f)
		c.checkBase(fieldPath, f.Base().BaseField, true, fields)
		binary[f.Base().Key] = f
	}
	for i, f := range p.Fields {
		if f.Variant != nil && f.Base().Multiplexor != nil {
			c.checkMultiplexor(pathOf(path, "fields", strconv.Itoa(i)), f, binary)
		}
	}
	c.checkOverlaps(pathOf(path, "fields"), p.Fields)
	c.checkIdentifier(path, p, binary)
}

// checkFrameLength checks a binary payload length against the frame that holds it.
func (c *checker) checkFrameLength(path []string, length uint16, framing Framing) {
	switch f := framing.Variant.(type) {
	case FixedFraming:
		if uint32(length) != f.Length {
			c.addf(path, "length %d must equal the frame length %d", length, f.Length)
		}
	case SyncFraming:
		if uint32(length) > f.MaxLength {
			c.addf(
				path, "length %d exceeds the max frame length %d", length, f.MaxLength,
			)
		}
	}
}

// checkSync finds the problems in a sync frame header that span two fields, which the
// schema cannot state.
func (c *checker) checkSync(framing Framing) {
	f, ok := framing.Variant.(SyncFraming)
	if !ok {
		return
	}
	if int(f.LengthOffset) < len(f.Sync) {
		c.addf(
			[]string{"framing", "length_offset"},
			"the length field overlaps the sync sequence",
		)
	}
	size, ok := lengthBytes[f.LengthSize]
	if header := uint64(f.LengthOffset) + size; ok && uint64(f.MaxLength) < header {
		c.addf(
			[]string{"framing", "max_length"},
			"the maximum length of %d is less than the %d bytes of the header",
			f.MaxLength,
			header,
		)
	}
}

// lengthBytes is the number of bytes in a length field of each size.
var lengthBytes = map[LengthSize]uint64{
	LengthSizeOne:  1,
	LengthSizeTwo:  2,
	LengthSizeFour: 4,
}

func (c *checker) checkTextFields(path []string, p TextPayload) {
	fields := newUniques("field", len(p.Fields))
	for i, f := range p.Fields {
		if f.Variant == nil {
			continue
		}
		fieldPath := pathOf(path, "fields", strconv.Itoa(i))
		if v, ok := f.Variant.(TaggedTextField); ok {
			c.checkTag(pathOf(fieldPath, "tag"), v.Tag, p.Delimiter)
		}
		if message, found := textDefaultProblem(f); found {
			c.addf(pathOf(fieldPath, "default"), "default %s", message)
		}
		c.checkBase(fieldPath, f.Base().BaseField, true, fields)
	}
	c.checkTagOverlaps(pathOf(path, "fields"), p.Fields)
}

// checkTagOverlaps refuses a tag that equals the tag of an earlier field, or that is
// the tag of another field and then text that can start a value. The Driver gives such
// a value to the longer tag. A field gets at most one such problem.
func (c *checker) checkTagOverlaps(path []string, fields []TextField) {
	for j, f := range fields {
		long, ok := f.Variant.(TaggedTextField)
		if !ok {
			continue
		}
		for i, g := range fields {
			if err := tagConflict(long, g, i < j); err != nil {
				c.add(pathOf(path, strconv.Itoa(j), "tag"), err)
				break
			}
		}
	}
}

// tagConflict returns an error when long has the tag of g and g is earlier, or when the
// tag of g and then a value of g read as the tag of long.
func tagConflict(long TaggedTextField, g TextField, earlier bool) error {
	short, ok := g.Variant.(TaggedTextField)
	if !ok || short.Tag == "" || !strings.HasPrefix(long.Tag, short.Tag) {
		return nil
	}
	switch rest := long.Tag[len(short.Tag):]; {
	case rest == "" && earlier:
		return validate.Errorf(
			"the tag %q is also the tag of field %s", long.Tag, short.Name,
		)
	case rest != "" && startsValue(rest):
		return validate.Errorf(
			"a value of field %s after its tag %q reads as this tag",
			short.Name,
			short.Tag,
		)
	}
	return nil
}

// startsValue reports whether rest, once the Driver trims textSpace from it, is empty
// or the start of a number that the Driver reads: an optional sign, digits, an optional
// point and digits, and an optional e or E with an optional sign and digits.
func startsValue(rest string) bool {
	s := strings.TrimLeft(rest, textSpace)
	i := 0
	skipSign := func() {
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
	}
	skipDigits := func() int {
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return i - start
	}
	skipSign()
	n := skipDigits()
	if i < len(s) && s[i] == '.' {
		i++
		n += skipDigits()
	}
	// An exponent needs a digit before it.
	if i < len(s) && n > 0 && (s[i] == 'e' || s[i] == 'E') {
		i++
		skipSign()
		skipDigits()
	}
	return i == len(s)
}

// textSpace is the text that the Driver trims from around each item of a line.
const textSpace = " \t\r"

// checkTag refuses a tag that the Driver cannot match in an item of a line split on
// delimiter.
func (c *checker) checkTag(path []string, tag, delimiter string) {
	switch {
	case tag == "":
		c.addf(path, "tag is required")
	case delimiter != "" && strings.Contains(tag, delimiter):
		c.addf(path, "the tag contains the delimiter %q", delimiter)
	case strings.IndexByte(textSpace, tag[0]) >= 0:
		c.addf(path, "the tag starts with whitespace")
	}
}

func (c *checker) checkJSONFields(path []string, p JSONPayload) {
	type read struct {
		name    string
		pointer xjson.Pointer
	}
	reads := make([]read, 0, len(p.Fields))
	fields := newUniques("field", len(p.Fields))
	for i, f := range p.Fields {
		if f.Variant == nil {
			continue
		}
		fieldPath := pathOf(path, strconv.Itoa(i))
		c.checkEnums(fieldPath, f)
		if message, bad := defaultProblem(f); bad {
			c.addf(pathOf(fieldPath, "default"), "default %s", message)
		}
		b := f.Base()
		var number bool
		switch f.Variant.(type) {
		case FloatJSONField, IntegerJSONField, UnsignedJSONField:
			number = true
		}
		c.checkBase(fieldPath, b.BaseField, number, fields)
		pointer, err := xjson.ParsePointer(b.Pointer)
		if err != nil {
			c.addf(pathOf(fieldPath, "pointer"), "invalid JSON Pointer %q", b.Pointer)
			continue
		}
		for _, r := range reads {
			if msg, ok := nested(pointer, r.pointer); ok {
				c.addf(pathOf(fieldPath, "pointer"), msg, b.Pointer, r.name)
				break
			}
		}
		reads = append(reads, read{name: b.Name, pointer: pointer})
	}
}

// checkEnums finds each enum label that is empty or that an earlier entry of the field
// uses, and each number value outside the min and max of the field. The scale and
// offset do not apply to an enum value.
func (c *checker) checkEnums(path []string, f JSONField) {
	var (
		labels []string
		values []float64
	)
	switch v := f.Variant.(type) {
	case FloatJSONField:
		for _, e := range v.EnumValues {
			labels = append(labels, e.Label)
			values = append(values, float64(e.Value))
		}
	case IntegerJSONField:
		for _, e := range v.EnumValues {
			labels = append(labels, e.Label)
			values = append(values, float64(e.Value))
		}
	case UnsignedJSONField:
		for _, e := range v.EnumValues {
			labels = append(labels, e.Label)
			values = append(values, float64(e.Value))
		}
	case StringJSONField:
		for _, e := range v.EnumValues {
			labels = append(labels, e.Label)
		}
	case BooleanJSONField:
		for _, e := range v.EnumValues {
			labels = append(labels, e.Label)
		}
	}
	seen := make(set.Set[string], len(labels))
	for i, label := range labels {
		labelPath := pathOf(path, "enum_values", strconv.Itoa(i), "label")
		if label == "" {
			c.addf(labelPath, "enum label is required")
			continue
		}
		if seen.Contains(label) {
			c.addf(labelPath, "duplicate enum label %q", label)
		}
		seen.Add(label)
	}
	b := f.Base().BaseField
	for i, v := range values {
		if message, out := outOfRange(b, v, v, 0); out {
			c.addf(
				pathOf(path, "enum_values", strconv.Itoa(i), "value"),
				"enum value %s",
				message,
			)
		}
	}
}

func (c *checker) checkMatch(path []string, m JSONMatch) {
	pointer := m.Base().Pointer
	parsed, err := xjson.ParsePointer(pointer)
	if err != nil {
		c.addf(pathOf(path, "pointer"), "invalid JSON Pointer %q", pointer)
	} else if _, ok := m.Variant.(PresentJSONMatch); ok && len(parsed) == 0 {
		// The root holds a value in every document.
		c.addf(pathOf(path, "pointer"), "pointer must not be empty")
	}
	if f, ok := m.Variant.(FloatJSONMatch); ok && !xmath.Finite(f.Value) {
		c.addf(pathOf(path, "value"), "value must be finite")
	}
}

// nested returns a problem format, taking a pointer and a field name, when a and b
// lead to the same value or one leads into the value of the other.
func nested(a, b xjson.Pointer) (string, bool) {
	switch {
	case slices.Equal(a, b):
		return "pointer %q duplicates the pointer of field %q", true
	case a.HasPrefix(b):
		return "pointer %q is inside the value of field %q", true
	case b.HasPrefix(a):
		return "pointer %q holds the value of field %q", true
	}
	return "", false
}

// checkUnscaled finds the scaling set on a field that holds no number.
func (c *checker) checkUnscaled(path []string, b BaseField) {
	if b.Scale != 1 {
		c.addf(pathOf(path, "scale"), "only a number field can scale")
	}
	if b.Offset != 0 {
		c.addf(pathOf(path, "offset"), "only a number field can have an offset")
	}
	if b.Min != nil {
		c.addf(pathOf(path, "min"), "only a number field can have a min")
	}
	if b.Max != nil {
		c.addf(pathOf(path, "max"), "only a number field can have a max")
	}
}

// CheckScale returns an error wrapping validate.ErrValidation when scale is zero. A
// field with a zero scale reads and writes every value as its offset.
func CheckScale(scale float64) error {
	if scale == 0 {
		return validate.Errorf("scale must be non-zero")
	}
	return nil
}

// checkScaling finds the problems in the scale and limits of a number field. The schema
// reports a value that is not finite.
func (c *checker) checkScaling(path []string, b BaseField) {
	if err := CheckScale(b.Scale); err != nil {
		c.add(pathOf(path, "scale"), err)
	}
	if b.Min == nil || b.Max == nil || !xmath.Finite(*b.Min) || !xmath.Finite(*b.Max) {
		return
	}
	if *b.Min > *b.Max {
		c.addf(pathOf(path, "max"), "max must be greater than or equal to min")
	}
}

func (c *checker) checkMultiplexor(
	path []string,
	f BinaryField,
	binary map[FieldKey]BinaryField,
) {
	base := f.Base()
	mux, ok := binary[*base.Multiplexor]
	switch {
	case *base.Multiplexor == base.Key:
		c.addf(pathOf(path, "multiplexor"), "a field cannot multiplex itself")
	case !ok:
		c.addf(pathOf(path, "multiplexor"), "the multiplexor is not in this message")
	case !integral(mux):
		c.addf(pathOf(path, "multiplexor"), "a multiplexor must be an integer field")
	case inCycle(base.Key, binary):
		c.addf(pathOf(path, "multiplexor"), "multiplexors form a cycle")
	}
	if len(base.MultiplexValues) == 0 {
		c.addf(
			pathOf(path, "multiplex_values"),
			"at least one value is required when the field has a multiplexor",
		)
	}
	if !ok || !integral(mux) || !validBitLength(mux.Base().BitLength) {
		return
	}
	for i, v := range base.MultiplexValues {
		if !fits(int64(v), mux) {
			c.addf(
				pathOf(path, "multiplex_values", strconv.Itoa(i)),
				"value %d does not fit the multiplexor %q",
				v,
				mux.Base().Name,
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
		if !ok || f.Base().Multiplexor == nil {
			return false
		}
		if cur = *f.Base().Multiplexor; cur == key {
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
		case !integral(f):
			c.addf(pathOf(idPath, "field"), "the identifier must be an integer field")
		case validBitLength(f.Base().BitLength) && !fits(int64(id.Value), f):
			c.addf(
				pathOf(idPath, "value"),
				"value %d does not fit the field %q",
				id.Value,
				f.Base().Name,
			)
		}
	}
}

func (c *checker) checkBinaryField(path []string, p BinaryPayload, f BinaryField) {
	base := f.Base()
	if !validBitLength(base.BitLength) {
		return
	}
	c.checkBinaryDefault(path, f)
	if p.Length != nil {
		if lastByte(base) >= int(*p.Length) {
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
		lastByte(base) >= int(canLength(id)) {
		c.addf(
			pathOf(path, "start_bit"),
			"field extends past the %d bytes a CAN frame carries",
			canLength(id),
		)
	}
}

// checkBinaryDefault adds a problem when f is a float of the wrong width, or when f
// cannot hold its default.
func (c *checker) checkBinaryDefault(path []string, f BinaryField) {
	if _, ok := f.Variant.(FloatBinaryField); ok {
		if n := f.Base().BitLength; n != 32 && n != 64 {
			c.addf(pathOf(path, "bit_length"), "a float field must be 32 or 64 bits")
		}
	}
	if message, found := binaryDefaultProblem(f); found {
		c.addf(pathOf(path, "default"), "default %s", message)
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
	for j, field := range fields {
		found[j] = noField
		if field.Variant == nil {
			continue
		}
		f := field.Base()
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
			c.addf(
				pathOf(path, strconv.Itoa(j), "start_bit"),
				"field overlaps the bits of field %q",
				fields[i].Base().Name,
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
func coveredBits(f BinaryBaseField) []int {
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

// integral reports whether the binary field f holds an integer.
func integral(f BinaryField) bool {
	switch f.Variant.(type) {
	case IntegerBinaryField, UnsignedBinaryField:
		return true
	default:
		return false
	}
}

// fits reports whether the integer field f, of a valid bit length, can hold v.
func fits(v int64, f BinaryField) bool {
	n := f.Base().BitLength
	if _, ok := f.Variant.(UnsignedBinaryField); ok {
		return v >= 0 && (n == 64 || v < int64(1)<<n)
	}
	return n == 64 || (v >= -(int64(1)<<(n-1)) && v < int64(1)<<(n-1))
}

// lastByte returns the index of the last payload byte that f, of a valid bit length,
// reads.
func lastByte(f BinaryBaseField) int { return slices.Max(coveredBits(f)) / 8 }
