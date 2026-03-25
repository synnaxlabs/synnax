// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	"encoding/binary"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Schema Resolution", func() {
	Describe("Resolve", func() {
		It("Should copy unchanged fields", func() {
			// Old: {Name string, Age int32}
			// New: {Name string, Age int32}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "age", Encoding: gorp.EncodingInt32},
			}
			newLayout := oldLayout

			// Encode: name="alice" (5 bytes), age=30
			var data []byte
			data = binary.BigEndian.AppendUint32(data, 5)
			data = append(data, "alice"...)
			data = binary.BigEndian.AppendUint32(data, 30)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(data))
		})

		It("Should add zero value for new field at end", func() {
			// Old: {Name string}
			// New: {Name string, Age int32}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "age", Encoding: gorp.EncodingInt32},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 5)
			data = append(data, "alice"...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			// Should be: name="alice" + age=0
			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 5)
			expected = append(expected, "alice"...)
			expected = binary.BigEndian.AppendUint32(expected, 0)
			Expect(result).To(Equal(expected))
		})

		It("Should drop removed field", func() {
			// Old: {Name string, Age int32, Email string}
			// New: {Name string, Email string}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "age", Encoding: gorp.EncodingInt32},
				{Name: "email", Encoding: gorp.EncodingString},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "email", Encoding: gorp.EncodingString},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 5)
			data = append(data, "alice"...)
			data = binary.BigEndian.AppendUint32(data, 30)
			data = binary.BigEndian.AppendUint32(data, 13)
			data = append(data, "alice@foo.com"...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 5)
			expected = append(expected, "alice"...)
			expected = binary.BigEndian.AppendUint32(expected, 13)
			expected = append(expected, "alice@foo.com"...)
			Expect(result).To(Equal(expected))
		})

		It("Should handle reordered fields", func() {
			// Old: {Name string, Age int32}
			// New: {Age int32, Name string}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "age", Encoding: gorp.EncodingInt32},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "age", Encoding: gorp.EncodingInt32},
				{Name: "name", Encoding: gorp.EncodingString},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 5)
			data = append(data, "alice"...)
			data = binary.BigEndian.AppendUint32(data, 30)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			// Should be: age=30, name="alice"
			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 30)
			expected = binary.BigEndian.AppendUint32(expected, 5)
			expected = append(expected, "alice"...)
			Expect(result).To(Equal(expected))
		})

		It("Should handle optional fields", func() {
			// Old: {Name string, Bio string? (absent)}
			// New: {Name string, Bio string?, Age int32}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "bio", Encoding: gorp.EncodingString, HardOptional: true},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "bio", Encoding: gorp.EncodingString, HardOptional: true},
				{Name: "age", Encoding: gorp.EncodingInt32},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 3)
			data = append(data, "bob"...)
			data = append(data, 0) // bio absent

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 3)
			expected = append(expected, "bob"...)
			expected = append(expected, 0)                          // bio absent
			expected = binary.BigEndian.AppendUint32(expected, 0) // age zero
			Expect(result).To(Equal(expected))
		})

		It("Should recursively resolve changed nested structs", func() {
			// Old inner: {X int32, Y int32}
			// New inner: {X int32, Y int32, Z int32}
			// Old outer: {Name string, Pos struct}
			// New outer: {Name string, Pos struct}
			oldInner := []gorp.FieldLayout{
				{Name: "x", Encoding: gorp.EncodingInt32},
				{Name: "y", Encoding: gorp.EncodingInt32},
			}
			newInner := []gorp.FieldLayout{
				{Name: "x", Encoding: gorp.EncodingInt32},
				{Name: "y", Encoding: gorp.EncodingInt32},
				{Name: "z", Encoding: gorp.EncodingInt32},
			}
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "pos", Encoding: gorp.EncodingStruct, Fields: oldInner},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "pos", Encoding: gorp.EncodingStruct, Fields: newInner},
			}

			// Old inner bytes: x=10, y=20 (8 bytes)
			var innerBytes []byte
			innerBytes = binary.BigEndian.AppendUint32(innerBytes, 10)
			innerBytes = binary.BigEndian.AppendUint32(innerBytes, 20)

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 4)
			data = append(data, "test"...)
			data = binary.BigEndian.AppendUint32(data, uint32(len(innerBytes)))
			data = append(data, innerBytes...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			// New inner: x=10, y=20, z=0 (12 bytes)
			var expectedInner []byte
			expectedInner = binary.BigEndian.AppendUint32(expectedInner, 10)
			expectedInner = binary.BigEndian.AppendUint32(expectedInner, 20)
			expectedInner = binary.BigEndian.AppendUint32(expectedInner, 0) // z=0

			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 4)
			expected = append(expected, "test"...)
			expected = binary.BigEndian.AppendUint32(expected, uint32(len(expectedInner)))
			expected = append(expected, expectedInner...)
			Expect(result).To(Equal(expected))
		})

		It("Should handle arrays of primitives", func() {
			// Old: {Tags []string}
			// New: {Tags []string, Count int32}
			oldLayout := []gorp.FieldLayout{
				{Name: "tags", Encoding: gorp.EncodingArray, Element: &gorp.FieldLayout{
					Encoding: gorp.EncodingString,
				}},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "tags", Encoding: gorp.EncodingArray, Element: &gorp.FieldLayout{
					Encoding: gorp.EncodingString,
				}},
				{Name: "count", Encoding: gorp.EncodingInt32},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 2) // count=2
			data = binary.BigEndian.AppendUint32(data, 3)
			data = append(data, "foo"...)
			data = binary.BigEndian.AppendUint32(data, 3)
			data = append(data, "bar"...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			var expected []byte
			expected = append(expected, data...) // tags unchanged
			expected = binary.BigEndian.AppendUint32(expected, 0) // count=0
			Expect(result).To(Equal(expected))
		})

		It("Should handle nested struct change inside an array", func() {
			// Old: {Items []struct{X int32}}
			// New: {Items []struct{X int32, Y int32}}
			oldElem := gorp.FieldLayout{
				Encoding: gorp.EncodingStruct,
				Fields: []gorp.FieldLayout{
					{Name: "x", Encoding: gorp.EncodingInt32},
				},
			}
			newElem := gorp.FieldLayout{
				Encoding: gorp.EncodingStruct,
				Fields: []gorp.FieldLayout{
					{Name: "x", Encoding: gorp.EncodingInt32},
					{Name: "y", Encoding: gorp.EncodingInt32},
				},
			}
			oldLayout := []gorp.FieldLayout{
				{Name: "items", Encoding: gorp.EncodingArray, Element: &oldElem},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "items", Encoding: gorp.EncodingArray, Element: &newElem},
			}

			// Encode: 2 elements, each struct is length-prefixed [4-byte len][x int32]
			var elem1, elem2 []byte
			elem1 = binary.BigEndian.AppendUint32(elem1, 42) // x=42
			elem2 = binary.BigEndian.AppendUint32(elem2, 99) // x=99

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 2) // count=2
			data = binary.BigEndian.AppendUint32(data, uint32(len(elem1)))
			data = append(data, elem1...)
			data = binary.BigEndian.AppendUint32(data, uint32(len(elem2)))
			data = append(data, elem2...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			// Each element should now have x + y=0
			var newElem1, newElem2 []byte
			newElem1 = binary.BigEndian.AppendUint32(newElem1, 42)
			newElem1 = binary.BigEndian.AppendUint32(newElem1, 0) // y=0
			newElem2 = binary.BigEndian.AppendUint32(newElem2, 99)
			newElem2 = binary.BigEndian.AppendUint32(newElem2, 0) // y=0

			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 2)
			expected = binary.BigEndian.AppendUint32(expected, uint32(len(newElem1)))
			expected = append(expected, newElem1...)
			expected = binary.BigEndian.AppendUint32(expected, uint32(len(newElem2)))
			expected = append(expected, newElem2...)
			Expect(result).To(Equal(expected))
		})

		It("Should handle field removal", func() {
			// Old: {Name string, Age int32, Email string}
			// New: {Name string, Email string}
			// (Age removed)
			oldLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "age", Encoding: gorp.EncodingInt32},
				{Name: "email", Encoding: gorp.EncodingString},
			}
			newLayout := []gorp.FieldLayout{
				{Name: "name", Encoding: gorp.EncodingString},
				{Name: "email", Encoding: gorp.EncodingString},
			}

			var data []byte
			data = binary.BigEndian.AppendUint32(data, 5)
			data = append(data, "alice"...)
			data = binary.BigEndian.AppendUint32(data, 30) // age
			data = binary.BigEndian.AppendUint32(data, 13)
			data = append(data, "alice@foo.com"...)

			result, err := gorp.Resolve(data, oldLayout, newLayout)
			Expect(err).ToNot(HaveOccurred())

			var expected []byte
			expected = binary.BigEndian.AppendUint32(expected, 5)
			expected = append(expected, "alice"...)
			expected = binary.BigEndian.AppendUint32(expected, 13)
			expected = append(expected, "alice@foo.com"...)
			Expect(result).To(Equal(expected))
		})
	})
})
