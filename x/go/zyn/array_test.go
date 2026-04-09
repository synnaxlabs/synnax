// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package zyn_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Array", func() {
	Describe("Basic Parsing", func() {
		Specify("string array", func() {
			schema := zyn.Array(zyn.String())
			data := []any{"a", "b", "c"}
			var dest []string
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal([]string{"a", "b", "c"}))
		})
		Specify("number array", func() {
			schema := zyn.Array(zyn.Number())
			data := []any{1.0, 2.0, 3.0}
			var dest []float64
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal([]float64{1.0, 2.0, 3.0}))
		})
		Specify("int array with coercion", func() {
			schema := zyn.Array(zyn.Uint32().Coerce())
			data := []any{1.0, 2.0, 3.0}
			var dest []uint32
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal([]uint32{1, 2, 3}))
		})
		Specify("bool array", func() {
			schema := zyn.Array(zyn.Bool())
			data := []any{true, false, true}
			var dest []bool
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(Equal([]bool{true, false, true}))
		})
		Specify("empty array", func() {
			schema := zyn.Array(zyn.String())
			data := []any{}
			var dest []string
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(HaveLen(0))
		})
		Specify("nested array of objects", func() {
			type Item struct {
				Name string
			}
			schema := zyn.Array(zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
			}))
			data := []any{
				map[string]any{"Name": "first"},
				map[string]any{"Name": "second"},
			}
			var dest []Item
			Expect(schema.Parse(data, &dest)).To(Succeed())
			Expect(dest).To(HaveLen(2))
			Expect(dest[0].Name).To(Equal("first"))
			Expect(dest[1].Name).To(Equal("second"))
		})
	})
	Describe("Validate", func() {
		It("Should succeed for a valid array", func() {
			Expect(zyn.Array(zyn.String()).Validate([]any{"a", "b"})).To(Succeed())
		})
		It("Should fail for a non-array", func() {
			Expect(zyn.Array(zyn.String()).Validate("not an array")).To(HaveOccurred())
		})
	})
	Describe("Invalid Inputs", func() {
		Specify("non-slice destination", func() {
			var dest string
			Expect(zyn.Array(zyn.String()).Parse([]any{"a"}, &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("nil pointer destination", func() {
			var dest *[]string
			Expect(zyn.Array(zyn.String()).Parse([]any{"a"}, dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("non-pointer destination", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Parse([]any{"a"}, dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("non-slice data", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Parse("not a slice", &dest)).
				To(MatchError(zyn.ErrInvalidDestinationType))
		})
		Specify("element validation error includes index", func() {
			var dest []uint32
			Expect(zyn.Array(zyn.Uint32().Coerce()).Parse([]any{1.0, "bad", 3.0}, &dest)).
				To(MatchError(ContainSubstring("1")))
		})
	})
	Describe("Optional Fields", func() {
		Specify("optional field with nil value", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Optional().Parse(nil, &dest)).To(Succeed())
			Expect(dest).To(BeNil())
		})
		Specify("required field with nil value", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Parse(nil, &dest)).
				To(MatchError(validate.ErrRequired))
		})
	})
	Describe("Length Constraints", func() {
		Specify("min constraint", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Min(2).Parse([]any{"a"}, &dest)).
				To(MatchError(ContainSubstring("less than minimum")))
		})
		Specify("max constraint", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Max(2).Parse([]any{"a", "b", "c"}, &dest)).
				To(MatchError(ContainSubstring("greater than maximum")))
		})
		Specify("within constraints", func() {
			var dest []string
			Expect(zyn.Array(zyn.String()).Min(1).Max(3).Parse([]any{"a", "b"}, &dest)).To(Succeed())
			Expect(dest).To(Equal([]string{"a", "b"}))
		})
	})
	Describe("Dump", func() {
		Specify("basic string array", func() {
			result := MustSucceed(zyn.Array(zyn.String()).Dump([]string{"a", "b", "c"}))
			Expect(result).To(Equal([]any{"a", "b", "c"}))
		})
		Specify("number array", func() {
			result := MustSucceed(zyn.Array(zyn.Number()).Dump([]int{1, 2, 3}))
			Expect(result).To(Equal([]any{int64(1), int64(2), int64(3)}))
		})
		Specify("empty array", func() {
			result := MustSucceed(zyn.Array(zyn.String()).Dump([]string{}))
			Expect(result).To(Equal([]any{}))
		})
		Specify("array of objects", func() {
			type Item struct {
				Name string
			}
			schema := zyn.Array(zyn.Object(map[string]zyn.Schema{
				"Name": zyn.String(),
			}))
			result := MustSucceed(schema.Dump([]Item{{Name: "first"}, {Name: "second"}}))
			Expect(result).To(Equal([]any{
				map[string]any{"name": "first"},
				map[string]any{"name": "second"},
			}))
		})
		Specify("nil required", func() {
			Expect(zyn.Array(zyn.String()).Dump(nil)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("nil pointer", func() {
			var s *[]string
			Expect(zyn.Array(zyn.String()).Dump(s)).Error().
				To(MatchError(validate.ErrRequired))
		})
		Specify("optional nil value", func() {
			result := MustSucceed(zyn.Array(zyn.String()).Optional().Dump(nil))
			Expect(result).To(BeNil())
		})
		Specify("optional nil pointer", func() {
			var s *[]string
			result := MustSucceed(zyn.Array(zyn.String()).Optional().Dump(s))
			Expect(result).To(BeNil())
		})
		Specify("non-slice value", func() {
			Expect(zyn.Array(zyn.String()).Dump("not a slice")).Error().
				To(MatchError(ContainSubstring("expected slice")))
		})
		Specify("min constraint", func() {
			Expect(zyn.Array(zyn.String()).Min(2).Dump([]string{"a"})).Error().
				To(MatchError(ContainSubstring("less than minimum")))
		})
		Specify("max constraint", func() {
			Expect(zyn.Array(zyn.String()).Max(1).Dump([]string{"a", "b"})).Error().
				To(MatchError(ContainSubstring("greater than maximum")))
		})
		Specify("round-trip parse then dump", func() {
			schema := zyn.Array(zyn.String())
			var dest []string
			Expect(schema.Parse([]any{"a", "b", "c"}, &dest)).To(Succeed())
			result := MustSucceed(schema.Dump(dest))
			Expect(result).To(Equal([]any{"a", "b", "c"}))
		})
	})
	Describe("Shape", func() {
		It("Should return an ArrayShape", func() {
			shape := zyn.Array(zyn.String()).Shape()
			Expect(shape.DataType()).To(Equal(zyn.ArrayT))
			Expect(shape.Optional()).To(BeFalse())
			Expect(shape.Fields()).To(BeNil())
			arrayShape, ok := shape.(zyn.ArrayShape)
			Expect(ok).To(BeTrue())
			Expect(arrayShape.Item().DataType()).To(Equal(zyn.StringT))
		})
		It("Should reflect optional", func() {
			Expect(zyn.Array(zyn.Number()).Optional().Shape().Optional()).To(BeTrue())
		})
	})
})
