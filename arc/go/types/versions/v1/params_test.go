// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Params", func() {
	var params types.Params
	BeforeEach(func() {
		params = types.Params{
			{Name: "x", Type: types.I32(), Value: 42},
			{Name: "y", Type: types.F64(), Value: 3.14},
			{Name: "flag", Type: types.U8(), Value: uint8(1)},
		}
	})
	Describe("MarshalJSON", func() {
		It("Should marshal nil params as an empty array", func() {
			var nilParams types.Params
			data := MustSucceed(json.Marshal(nilParams))
			Expect(string(data)).To(Equal("[]"))
		})
		It("Should marshal params as a JSON array", func() {
			data := MustSucceed(json.Marshal(params))
			Expect(string(data)).To(HavePrefix("["))
			Expect(string(data)).To(ContainSubstring(`"name":"x"`))
			Expect(string(data)).To(ContainSubstring(`"name":"flag"`))
		})
	})
	Describe("Get", func() {
		It("Should return parameter when found", func() {
			param, ok := params.Get("x")
			Expect(ok).To(BeTrue())
			Expect(param.Name).To(Equal("x"))
			Expect(param.Type).To(Equal(types.I32()))
			Expect(param.Value).To(Equal(42))
		})
		It("Should return false when parameter not found", func() {
			param, ok := params.Get("nonexistent")
			Expect(ok).To(BeFalse())
			Expect(param).To(Equal(types.Param{}))
		})
		It("Should find last parameter", func() {
			param, ok := params.Get("flag")
			Expect(ok).To(BeTrue())
			Expect(param.Name).To(Equal("flag"))
		})
		It("Should work with empty params", func() {
			empty := types.Params{}
			param, ok := empty.Get("x")
			Expect(ok).To(BeFalse())
			Expect(param).To(Equal(types.Param{}))
		})
	})
	Describe("GetIndex", func() {
		It("Should return correct index when found", func() {
			Expect(params.GetIndex("x")).To(Equal(0))
			Expect(params.GetIndex("y")).To(Equal(1))
			Expect(params.GetIndex("flag")).To(Equal(2))
		})
		It("Should return -1 when not found", func() {
			Expect(params.GetIndex("nonexistent")).To(Equal(-1))
		})
		It("Should return -1 for empty params", func() {
			empty := types.Params{}
			Expect(empty.GetIndex("x")).To(Equal(-1))
		})
	})
	Describe("Has", func() {
		It("Should return true for existing parameters", func() {
			Expect(params.Has("x")).To(BeTrue())
			Expect(params.Has("y")).To(BeTrue())
			Expect(params.Has("flag")).To(BeTrue())
		})
		It("Should return false for non-existing parameters", func() {
			Expect(params.Has("nonexistent")).To(BeFalse())
		})
		It("Should return false for empty params", func() {
			empty := types.Params{}
			Expect(empty.Has("x")).To(BeFalse())
		})
	})
	Describe("Positional", func() {
		It("Should return all params when the trigger is empty", func() {
			Expect(params.Positional("")).To(Equal(params))
		})
		It("Should exclude the trigger param", func() {
			positional := params.Positional("y")
			Expect(positional).To(HaveLen(2))
			Expect(positional.Has("y")).To(BeFalse())
			Expect(positional.Has("x")).To(BeTrue())
			Expect(positional.Has("flag")).To(BeTrue())
		})
		It("Should return all params when the trigger names no param", func() {
			Expect(params.Positional("nonexistent")).To(HaveLen(3))
		})
		It("Should return empty for empty params", func() {
			empty := types.Params{}
			Expect(empty.Positional("y")).To(BeEmpty())
		})
	})
	Describe("ValueMap", func() {
		It("Should return map of parameter names to values", func() {
			valueMap := params.ValueMap()
			Expect(valueMap).To(HaveLen(3))
			Expect(valueMap["x"]).To(Equal(42))
			Expect(valueMap["y"]).To(Equal(3.14))
			Expect(valueMap["flag"]).To(Equal(uint8(1)))
		})
		It("Should return empty map for empty params", func() {
			empty := types.Params{}
			valueMap := empty.ValueMap()
			Expect(valueMap).To(BeEmpty())
		})
		It("Should handle nil values", func() {
			paramsWithNil := types.Params{
				{Name: "a", Type: types.I32(), Value: nil},
				{Name: "b", Type: types.F64(), Value: 1.5},
			}
			valueMap := paramsWithNil.ValueMap()
			Expect(valueMap).To(HaveLen(2))
			Expect(valueMap["a"]).To(BeNil())
			Expect(valueMap["b"]).To(Equal(1.5))
		})
	})
	Describe("RequiredCount", func() {
		It("Should return total count when no parameters have defaults", func() {
			requiredOnly := types.Params{
				{Name: "a", Type: types.I32(), Value: nil},
				{Name: "b", Type: types.F64(), Value: nil},
				{Name: "c", Type: types.U8(), Value: nil},
			}
			Expect(requiredOnly.RequiredCount()).To(Equal(3))
		})
		It("Should return zero when all parameters have defaults", func() {
			allOptional := types.Params{
				{Name: "a", Type: types.I32(), Value: int32(10)},
				{Name: "b", Type: types.F64(), Value: 3.14},
			}
			Expect(allOptional.RequiredCount()).To(Equal(0))
		})
		It("Should return count of parameters without defaults (mixed)", func() {
			mixed := types.Params{
				{Name: "required1", Type: types.I64(), Value: nil},
				{Name: "required2", Type: types.I64(), Value: nil},
				{Name: "optional1", Type: types.I64(), Value: int64(100)},
				{Name: "optional2", Type: types.I64(), Value: int64(200)},
			}
			Expect(mixed.RequiredCount()).To(Equal(2))
		})
		It("Should return zero for empty params", func() {
			empty := types.Params{}
			Expect(empty.RequiredCount()).To(Equal(0))
		})
		It("Should count correctly with single required parameter", func() {
			single := types.Params{
				{Name: "x", Type: types.I32(), Value: nil},
			}
			Expect(single.RequiredCount()).To(Equal(1))
		})
		It("Should count correctly with single optional parameter", func() {
			single := types.Params{
				{Name: "x", Type: types.I32(), Value: int32(42)},
			}
			Expect(single.RequiredCount()).To(Equal(0))
		})
	})
	Describe("String", func() {
		DescribeTable(
			"Rendering",
			func(params types.Params, expected string) {
				Expect(params.String()).To(Equal(expected))
			},
			Entry("empty params", types.Params{}, "(none)"),
			Entry("params without values",
				types.Params{
					{Name: "x", Type: types.I64()},
					{Name: "y", Type: types.F64()},
				},
				"x (i64), y (f64)"),
			Entry("a param with a value",
				types.Params{{Name: "x", Type: types.I32(), Value: 42}},
				"x (i32) = 42"),
			Entry("mixed valued and unvalued params",
				types.Params{
					{Name: "x", Type: types.I32(), Value: 42},
					{Name: "y", Type: types.F64()},
				},
				"x (i32) = 42, y (f64)"),
		)
	})
})
