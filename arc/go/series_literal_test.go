// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/channels"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Typed series literals", func() {
	// first runs body in a func and returns the sample it wrote, as out's type.
	first := func(ctx SpecContext, dt types.Type, dataType telem.DataType, body string) telem.Series {
		GinkgoHelper()
		h := newRuntimeHarness(ctx, `
		func pick() series f32 {
			s series f32 := [1.5, 2.5]
			return s
		}
		func head(s series f32) f32 {
			return s[0]
		}
		func check() {v `+dt.String()+`} `+dt.String()+` {
			`+body+`
		}
		x -> check() -> out`, channelSymbols(map[string]channelDef{
			"x":   {dt, 100},
			"out": {dt, 200},
		}),
			channels.Digest{Key: 100, DataType: dataType},
			channels.Digest{Key: 200, DataType: dataType},
		)
		defer h.Close(ctx)
		h.Ingest(100, telem.NewSeriesFromAny(2, dataType))
		h.Tick(ctx, telem.Millisecond)
		out, _ := h.Flush()
		return out.Get(200).Series[0]
	}

	DescribeTable(
		"Should read an f32 element",
		func(ctx SpecContext, body string, expected float32) {
			got := first(ctx, types.F32(), telem.Float32T, body)
			Expect(got.Unmarshal[float32]()[0]).To(Equal(expected))
		},
		Entry("literal", "y series f32 := [1.5]\nreturn y[0]", float32(1.5)),
		Entry("integer literal", "y series f32 := [3]\nreturn y[0]", float32(3)),
		Entry("arithmetic", "y series f32 := [1.5 + 2]\nreturn y[0]", float32(3.5)),
		Entry("integer arithmetic", "y series f32 := [1 + 2]\nreturn y[0]", float32(3)),
		Entry("negated", "y series f32 := [-1.5]\nreturn y[0]", float32(-1.5)),
		Entry(
			"with a variable",
			"y series f32 := [1.5 + v]\nreturn y[0]",
			float32(3.5),
		),
		Entry("variable first", "y series f32 := [v + 1.5]\nreturn y[0]", float32(3.5)),
		Entry("power", "y series f32 := [2.0 ^ 3.0]\nreturn y[0]", float32(8)),
		Entry("a cast", "y series f32 := [f32(1.5)]\nreturn y[0]", float32(1.5)),
		Entry("parentheses", "y series f32 := [(1.5)]\nreturn y[0]", float32(1.5)),
		Entry(
			"reassigned",
			"y series f32 := [1.5]\ny = [2.5]\nreturn y[0]",
			float32(2.5),
		),
		Entry("stateful", "y series f32 $= [1.5]\nreturn y[0]", float32(1.5)),
		Entry("second element", "y series f32 := [1, 2.5]\nreturn y[1]", float32(2.5)),
		Entry(
			"left operand",
			"y series f32 := [1.5]\nz := [1.5] + y\nreturn z[0]",
			float32(3),
		),
		Entry("from a func", "y := pick()\nreturn y[0]", float32(1.5)),
		Entry("into a func", "return head([1.5])", float32(1.5)),
		Entry("indexed after a call", "return pick()[1]", float32(2.5)),
		Entry("declared from a call index", "y := pick()[1]\nreturn y", float32(2.5)),
		Entry("sliced after a call", "return pick()[1:][0]", float32(2.5)),
		Entry(
			"declared from a call slice",
			"y := pick()[1:]\nreturn y[0]",
			float32(2.5),
		),
		Entry(
			"indexed after a call in arithmetic",
			"return pick()[1] + v",
			float32(4.5),
		),
		Entry(
			"untyped then a series",
			"y := []\ny = pick()\nreturn y[0]",
			float32(1.5),
		),
		Entry("untyped then a literal", "y := []\ny = [v]\nreturn y[0]", float32(2)),
	)

	DescribeTable("Should read an untyped f64 element",
		func(ctx SpecContext, body string, expected float64) {
			got := first(ctx, types.F64(), telem.Float64T, body)
			Expect(got.Unmarshal[float64]()[0]).To(Equal(expected))
		},
		Entry("float, exact float, integer", "s := [1.5, 2.0, 3]\nreturn s[2]", 3.0),
		Entry("integer, float", "s := [1, 2.5]\nreturn s[1]", 2.5),
		Entry("integer, exact float, float", "s := [1, 2.0, 2.5]\nreturn s[2]", 2.5),
		Entry("integer, float", "s := [3, 1.5]\nreturn s[1]", 1.5),
	)

	DescribeTable("Should read an i32 element",
		func(ctx SpecContext, body string, expected int32) {
			got := first(ctx, types.I32(), telem.Int32T, body)
			Expect(got.Unmarshal[int32]()[0]).To(Equal(expected))
		},
		Entry("literal", "y series i32 := [3]\nreturn y[0]", int32(3)),
		Entry("arithmetic", "y series i32 := [1 + 2]\nreturn y[0]", int32(3)),
		Entry("with a variable", "y series i32 := [1 + v]\nreturn y[0]", int32(3)),
		Entry("reassigned", "y series i32 := [3]\ny = [4]\nreturn y[0]", int32(4)),
	)

	It("Should keep an integer-only series as i64", func(ctx SpecContext) {
		got := first(ctx, types.I64(), telem.Int64T, "s := [1, 2]\nreturn s[1]")
		Expect(got.Unmarshal[int64]()[0]).To(Equal(int64(2)))
	})

	DescribeTable("Should accept an empty series of any element type",
		func(ctx SpecContext, body string) {
			got := first(ctx, types.I64(), telem.Int64T, body)
			Expect(got.Unmarshal[int64]()[0]).To(Equal(int64(0)))
		},
		Entry("declared bool", "y series bool := []\nreturn len(y)"),
		Entry("reassigned bool", "y series bool := []\ny = []\nreturn len(y)"),
		Entry("declared f32", "y series f32 := []\nreturn len(y)"),
		Entry("reassigned f32", "y series f32 := []\ny = []\nreturn len(y)"),
		Entry("declared f64", "y series f64 := []\nreturn len(y)"),
		Entry("reassigned f64", "y series f64 := []\ny = []\nreturn len(y)"),
		Entry("declared i8", "y series i8 := []\nreturn len(y)"),
		Entry("reassigned i8", "y series i8 := []\ny = []\nreturn len(y)"),
		Entry("declared i16", "y series i16 := []\nreturn len(y)"),
		Entry("reassigned i16", "y series i16 := []\ny = []\nreturn len(y)"),
		Entry("declared i32", "y series i32 := []\nreturn len(y)"),
		Entry("reassigned i32", "y series i32 := []\ny = []\nreturn len(y)"),
		Entry("declared i64", "y series i64 := []\nreturn len(y)"),
		Entry("reassigned i64", "y series i64 := []\ny = []\nreturn len(y)"),
		Entry("declared u8", "y series u8 := []\nreturn len(y)"),
		Entry("reassigned u8", "y series u8 := []\ny = []\nreturn len(y)"),
		Entry("declared u16", "y series u16 := []\nreturn len(y)"),
		Entry("reassigned u16", "y series u16 := []\ny = []\nreturn len(y)"),
		Entry("declared u32", "y series u32 := []\nreturn len(y)"),
		Entry("reassigned u32", "y series u32 := []\ny = []\nreturn len(y)"),
		Entry("declared u64", "y series u64 := []\nreturn len(y)"),
		Entry("reassigned u64", "y series u64 := []\ny = []\nreturn len(y)"),
		Entry("declared timestamp", "y series i64 ns := []\nreturn len(y)"),
		Entry("reassigned timestamp", "y series i64 ns := []\ny = []\nreturn len(y)"),
		Entry("untyped", "y := []\nreturn len(y)"),
		Entry("untyped reassigned", "y := []\ny = []\nreturn len(y)"),
		Entry("untyped compound", "y := []\ny += 1.5\nreturn len(y)"),
		Entry("untyped negated", "y := []\nz := -y\nreturn len(z)"),
	)

	DescribeTable("Should read a u8 element",
		func(ctx SpecContext, body string, expected uint8) {
			got := first(ctx, types.U8(), telem.Uint8T, body)
			Expect(got.Unmarshal[uint8]()[0]).To(Equal(expected))
		},
		Entry("literal", "y series u8 := [200]\nreturn y[0]", uint8(200)),
		Entry("arithmetic", "y series u8 := [100 + 100]\nreturn y[0]", uint8(200)),
	)
})
