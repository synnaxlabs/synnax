// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package gorp provides resolve.go: a schema-driven binary resolver that transforms
// entries from one field layout to another without requiring frozen Go types. Uses
// Avro-style schema resolution: fields are matched by name between old and new layouts,
// reads use old positions, writes use new positions.

package gorp

import (
	"encoding/binary"
	"math"

	"github.com/synnaxlabs/x/errors"
)

// Encoding describes how a field is encoded in the binary format.
type Encoding int

const (
	EncodingBool    Encoding = iota // 1 byte
	EncodingInt8                    // 1 byte
	EncodingInt16                   // 2 bytes big-endian
	EncodingInt32                   // 4 bytes big-endian
	EncodingInt64                   // 8 bytes big-endian
	EncodingUint8                   // 1 byte
	EncodingUint16                  // 2 bytes big-endian
	EncodingUint32                  // 4 bytes big-endian
	EncodingUint64                  // 8 bytes big-endian
	EncodingFloat32                 // 4 bytes big-endian (IEEE 754)
	EncodingFloat64                 // 8 bytes big-endian (IEEE 754)
	EncodingUUID                    // 16 bytes direct
	EncodingString                  // [uint32 len][bytes]
	EncodingBytes                   // [uint32 len][bytes]
	EncodingJSON                    // [uint32 len][json bytes]
	EncodingStruct                  // [uint32 len][struct bytes]
	EncodingArray                   // [uint32 count][elements]
	EncodingMap                     // [uint32 count][kv pairs]
)

// FieldLayout describes the binary encoding of a single field.
type FieldLayout struct {
	Name         string
	Encoding     Encoding
	Optional     bool // soft optional: 1-byte presence flag
	HardOptional bool // hard optional: 1-byte presence flag (pointer in Go)
	Fields       []FieldLayout // nested struct fields
	Element      *FieldLayout  // array element layout
	Key          *FieldLayout  // map key layout
	Value        *FieldLayout  // map value layout
}

// fixedSize returns the byte size for fixed-size encodings, or 0 for variable-size.
func fixedSize(e Encoding) int {
	switch e {
	case EncodingBool, EncodingInt8, EncodingUint8:
		return 1
	case EncodingInt16, EncodingUint16:
		return 2
	case EncodingInt32, EncodingUint32, EncodingFloat32:
		return 4
	case EncodingInt64, EncodingUint64, EncodingFloat64:
		return 8
	case EncodingUUID:
		return 16
	default:
		return 0
	}
}

// Resolve transforms binary data from oldLayout to newLayout by matching fields
// by name. Fields present in old but not new are dropped. Fields present in new
// but not old get zero-value bytes. Nested structs are resolved recursively.
func Resolve(oldData []byte, oldLayout, newLayout []FieldLayout) ([]byte, error) {
	extracted, err := extractFields(oldData, oldLayout)
	if err != nil {
		return nil, err
	}
	return writeFields(extracted, oldLayout, newLayout)
}

// extractedField holds raw bytes for a field, keyed by name.
type extractedField struct {
	bytes  []byte
	layout FieldLayout
}

// extractFields reads all fields from data using the given layout, returning
// raw bytes keyed by field name.
func extractFields(data []byte, layout []FieldLayout) (map[string]extractedField, error) {
	result := make(map[string]extractedField, len(layout))
	for _, f := range layout {
		n, err := fieldSize(data, f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compute size for field %s", f.Name)
		}
		if n > len(data) {
			break // bounds-safe: remaining fields get zero values
		}
		result[f.Name] = extractedField{bytes: data[:n], layout: f}
		data = data[n:]
	}
	return result, nil
}

// writeFields writes fields in newLayout order, using extracted data from old fields.
func writeFields(
	extracted map[string]extractedField,
	oldLayout, newLayout []FieldLayout,
) ([]byte, error) {
	var buf []byte
	for _, nf := range newLayout {
		old, exists := extracted[nf.Name]
		if !exists {
			buf = append(buf, zeroValue(nf)...)
			continue
		}
		if LayoutsEqual(old.layout, nf) {
			buf = append(buf, old.bytes...)
			continue
		}
		resolved, err := resolveField(old.bytes, old.layout, nf)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve field %s", nf.Name)
		}
		buf = append(buf, resolved...)
	}
	return buf, nil
}

// resolveField resolves a single field whose layout changed between old and new.
func resolveField(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	switch newField.Encoding {
	case EncodingStruct:
		return resolveStructField(data, oldField, newField)
	case EncodingArray:
		return resolveArrayField(data, oldField, newField)
	case EncodingMap:
		return resolveMapField(data, oldField, newField)
	default:
		// Primitive type changed or optional flag changed. Copy raw bytes.
		// The bounds-safe decoder handles any size differences.
		return data, nil
	}
}

// resolveArrayField resolves an array where the element layout changed.
func resolveArrayField(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if oldField.Optional || oldField.HardOptional {
		if len(data) < 1 {
			return zeroValue(newField), nil
		}
		if data[0] == 0 {
			return []byte{0}, nil
		}
		inner, err := resolveArrayInner(data[1:], oldField, newField)
		if err != nil {
			return nil, err
		}
		return append([]byte{1}, inner...), nil
	}
	return resolveArrayInner(data, oldField, newField)
}

func resolveArrayInner(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if len(data) < 4 || oldField.Element == nil || newField.Element == nil {
		return data, nil
	}
	count := int(binary.BigEndian.Uint32(data[:4]))
	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(count))
	offset := 4
	for range count {
		elemSize, err := fieldSize(data[offset:], *oldField.Element)
		if err != nil {
			return nil, err
		}
		elemData := data[offset : offset+elemSize]
		if LayoutsEqual(*oldField.Element, *newField.Element) {
			buf = append(buf, elemData...)
		} else {
			resolved, err := resolveField(elemData, *oldField.Element, *newField.Element)
			if err != nil {
				return nil, err
			}
			buf = append(buf, resolved...)
		}
		offset += elemSize
	}
	return buf, nil
}

// resolveMapField resolves a map where key or value layouts changed.
func resolveMapField(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if oldField.Optional || oldField.HardOptional {
		if len(data) < 1 {
			return zeroValue(newField), nil
		}
		if data[0] == 0 {
			return []byte{0}, nil
		}
		inner, err := resolveMapInner(data[1:], oldField, newField)
		if err != nil {
			return nil, err
		}
		return append([]byte{1}, inner...), nil
	}
	return resolveMapInner(data, oldField, newField)
}

func resolveMapInner(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if len(data) < 4 || oldField.Key == nil || oldField.Value == nil ||
		newField.Key == nil || newField.Value == nil {
		return data, nil
	}
	count := int(binary.BigEndian.Uint32(data[:4]))
	var buf []byte
	buf = binary.BigEndian.AppendUint32(buf, uint32(count))
	offset := 4
	for range count {
		// Resolve key
		kSize, err := fieldValueSize(data[offset:], *oldField.Key)
		if err != nil {
			return nil, err
		}
		buf = append(buf, data[offset:offset+kSize]...) // keys are always copied raw
		offset += kSize
		// Resolve value
		vSize, err := fieldValueSize(data[offset:], *oldField.Value)
		if err != nil {
			return nil, err
		}
		vData := data[offset : offset+vSize]
		if LayoutsEqual(*oldField.Value, *newField.Value) {
			buf = append(buf, vData...)
		} else {
			resolved, err := resolveField(vData, *oldField.Value, *newField.Value)
			if err != nil {
				return nil, err
			}
			buf = append(buf, resolved...)
		}
		offset += vSize
	}
	return buf, nil
}

// resolveStructField resolves a length-prefixed struct field from old to new layout.
func resolveStructField(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if oldField.Optional || oldField.HardOptional {
		if len(data) < 1 {
			return zeroValue(newField), nil
		}
		flag := data[0]
		if flag == 0 {
			return zeroValue(newField), nil
		}
		data = data[1:] // skip presence flag
		inner, err := resolveStructInner(data, oldField, newField)
		if err != nil {
			return nil, err
		}
		return append([]byte{1}, inner...), nil
	}
	return resolveStructInner(data, oldField, newField)
}

// resolveStructInner resolves the inner bytes of a length-prefixed struct.
func resolveStructInner(data []byte, oldField, newField FieldLayout) ([]byte, error) {
	if len(data) < 4 {
		return zeroValue(newField), nil
	}
	innerLen := int(binary.BigEndian.Uint32(data[:4]))
	innerData := data[4:]
	if len(innerData) > innerLen {
		innerData = innerData[:innerLen]
	}
	// If both layouts have empty Fields, this is a recursive type cycle.
	// Copy raw bytes as-is. The bounds-safe decoder handles additive changes
	// at read time. Breaking changes to recursive types are not supported.
	if len(oldField.Fields) == 0 && len(newField.Fields) == 0 {
		return data[:4+innerLen], nil
	}
	resolved, err := Resolve(innerData, oldField.Fields, newField.Fields)
	if err != nil {
		return nil, err
	}
	result := make([]byte, 4+len(resolved))
	binary.BigEndian.PutUint32(result[:4], uint32(len(resolved)))
	copy(result[4:], resolved)
	return result, nil
}

// fieldSize computes the total byte size of a field in the data, including any
// presence flags and length prefixes.
func fieldSize(data []byte, f FieldLayout) (int, error) {
	if f.Optional || f.HardOptional {
		if len(data) < 1 {
			return 0, nil
		}
		if data[0] == 0 {
			return 1, nil // absent
		}
		inner, err := fieldValueSize(data[1:], f)
		if err != nil {
			return 0, err
		}
		return 1 + inner, nil
	}
	return fieldValueSize(data, f)
}

// fieldValueSize computes the byte size of a field's value (without presence flags).
func fieldValueSize(data []byte, f FieldLayout) (int, error) {
	if fs := fixedSize(f.Encoding); fs > 0 {
		return fs, nil
	}
	switch f.Encoding {
	case EncodingString, EncodingBytes, EncodingJSON:
		if len(data) < 4 {
			return 0, nil
		}
		n := int(binary.BigEndian.Uint32(data[:4]))
		return 4 + n, nil

	case EncodingStruct:
		if len(data) < 4 {
			return 0, nil
		}
		n := int(binary.BigEndian.Uint32(data[:4]))
		return 4 + n, nil

	case EncodingArray:
		if f.Element == nil {
			return 0, errors.New("array field missing element layout")
		}
		if len(data) < 4 {
			return 0, nil
		}
		count := int(binary.BigEndian.Uint32(data[:4]))
		offset := 4
		for range count {
			elemSize, err := fieldSize(data[offset:], *f.Element)
			if err != nil {
				return 0, err
			}
			offset += elemSize
		}
		return offset, nil

	case EncodingMap:
		if f.Key == nil || f.Value == nil {
			return 0, errors.New("map field missing key/value layout")
		}
		if len(data) < 4 {
			return 0, nil
		}
		count := int(binary.BigEndian.Uint32(data[:4]))
		offset := 4
		for range count {
			kSize, err := fieldValueSize(data[offset:], *f.Key)
			if err != nil {
				return 0, err
			}
			offset += kSize
			vSize, err := fieldValueSize(data[offset:], *f.Value)
			if err != nil {
				return 0, err
			}
			offset += vSize
		}
		return offset, nil

	default:
		return 0, errors.Newf("unknown encoding %d", f.Encoding)
	}
}

// zeroValue returns the zero-value byte encoding for a field.
func zeroValue(f FieldLayout) []byte {
	if f.Optional || f.HardOptional {
		return []byte{0} // absent
	}
	switch f.Encoding {
	case EncodingBool, EncodingInt8, EncodingUint8:
		return []byte{0}
	case EncodingInt16, EncodingUint16:
		return []byte{0, 0}
	case EncodingInt32, EncodingUint32:
		return []byte{0, 0, 0, 0}
	case EncodingFloat32:
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, math.Float32bits(0))
		return b
	case EncodingInt64, EncodingUint64:
		return []byte{0, 0, 0, 0, 0, 0, 0, 0}
	case EncodingFloat64:
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, math.Float64bits(0))
		return b
	case EncodingUUID:
		return make([]byte, 16)
	case EncodingString, EncodingBytes, EncodingJSON:
		return []byte{0, 0, 0, 0} // length 0
	case EncodingStruct:
		// Empty struct: length 0
		return []byte{0, 0, 0, 0}
	case EncodingArray, EncodingMap:
		return []byte{0, 0, 0, 0} // count 0
	default:
		return nil
	}
}

// layoutsEqual returns true if two field layouts describe the same encoding.
func LayoutsEqual(a, b FieldLayout) bool {
	if a.Encoding != b.Encoding || a.Optional != b.Optional || a.HardOptional != b.HardOptional {
		return false
	}
	if len(a.Fields) != len(b.Fields) {
		return false
	}
	for i := range a.Fields {
		if a.Fields[i].Name != b.Fields[i].Name || !LayoutsEqual(a.Fields[i], b.Fields[i]) {
			return false
		}
	}
	if (a.Element == nil) != (b.Element == nil) {
		return false
	}
	if a.Element != nil && !LayoutsEqual(*a.Element, *b.Element) {
		return false
	}
	if (a.Key == nil) != (b.Key == nil) {
		return false
	}
	if a.Key != nil && !LayoutsEqual(*a.Key, *b.Key) {
		return false
	}
	if (a.Value == nil) != (b.Value == nil) {
		return false
	}
	if a.Value != nil && !LayoutsEqual(*a.Value, *b.Value) {
		return false
	}
	return true
}
