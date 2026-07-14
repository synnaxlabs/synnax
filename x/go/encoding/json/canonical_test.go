// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Canonical", func() {
	DescribeTable("numbers match JSON.stringify",
		func(v any, expected string) {
			Expect(string(MustSucceed(json.Canonical(v)))).To(Equal(expected))
		},
		Entry("integral float", 1.0, "1"),
		Entry("negative zero", math.Copysign(0, -1), "0"),
		Entry("simple decimal", 0.1, "0.1"),
		Entry("half", 2.5, "2.5"),
		Entry("scale factor", 0.001, "0.001"),
		Entry("smallest decimal form", 0.000001, "0.000001"),
		Entry("exponent below decimal range", 1e-7, "1e-7"),
		Entry("large decimal form", 1e20, "100000000000000000000"),
		Entry("exponent above decimal range", 1e21, "1e+21"),
		Entry("large with mantissa", 1.5e300, "1.5e+300"),
		Entry("denormal", 5e-324, "5e-324"),
		Entry("full precision", 123456789012345680000.0, "123456789012345680000"),
		Entry("negative", -42.75, "-42.75"),
		Entry("int64", int64(50000), "50000"),
		Entry("uint64", uint64(math.MaxUint64), "18446744073709551615"),
		Entry("integral float rate", 50.0, "50"),
	)

	DescribeTable("values",
		func(v any, expected string) {
			Expect(string(MustSucceed(json.Canonical(v)))).To(Equal(expected))
		},
		Entry("nil", nil, "null"),
		Entry("true", true, "true"),
		Entry("false", false, "false"),
		Entry("empty object", map[string]any{}, "{}"),
		Entry("empty array", []any{}, "[]"),
		Entry(
			"sorted keys",
			map[string]any{"b": 2.5, "a": 1, "g": map[string]any{"z": 1, "a": "x"}},
			`{"a":1,"b":2.5,"g":{"a":"x","z":1}}`,
		),
		Entry(
			"escapes and unicode",
			map[string]any{"notes": "héllo⚡ <&> \n\ttab", "name": `ch"1"`},
			`{"name":"ch\"1\"","notes":"héllo⚡ <&> \n\ttab"}`,
		),
		Entry(
			"control characters",
			string([]byte{0x01, 0x1f}),
			`"\u0001\u001f"`,
		),
		Entry(
			"nested config",
			map[string]any{
				"rate":    50.0,
				"port":    8080,
				"host":    "localhost",
				"enabled": false,
				"channels": []any{
					map[string]any{"key": 12, "name": `ch"1"`, "scale": 0.001},
				},
			},
			`{"channels":[{"key":12,"name":"ch\"1\"","scale":0.001}],`+
				`"enabled":false,"host":"localhost","port":8080,"rate":50}`,
		),
		Entry(
			"float extremes",
			map[string]any{
				"huge": 1e21, "list": []any{}, "neg": -42.75,
				"sample_rate": 1e-7, "tiny": 5e-324, "zero": 0,
			},
			`{"huge":1e+21,"list":[],"neg":-42.75,"sample_rate":1e-7,`+
				`"tiny":5e-324,"zero":0}`,
		),
	)

	DescribeTable("rejects non-JSON values",
		func(v any) {
			Expect(json.Canonical(v)).Error().
				To(MatchError(ContainSubstring("canonical JSON")))
		},
		Entry("NaN", math.NaN()),
		Entry("positive infinity", math.Inf(1)),
		Entry("channel", map[string]any{"ch": make(chan int)}),
		Entry("non-string key", map[any]any{1: "a"}),
	)
})
