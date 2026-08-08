// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Legacy key case conversion", func() {
	// Parity vectors copied verbatim from the TS wire codec's camelToSnake specs in
	// x/ts/src/caseconv/caseconv.spec.ts, so the Go port provably matches the
	// conversion legacy configs went through on the wire.
	DescribeTable("camelToSnake matches the TS wire codec",
		func(input, expected string) {
			Expect(camelToSnake(input)).To(Equal(expected))
		},
		Entry(nil, "fooBar", "foo_bar"),
		Entry(nil, "fooBarBaz", "foo_bar_baz"),
		Entry(nil, "foo", "foo"),
		Entry(nil, "foo_bar", "foo_bar"),
		Entry(nil, "foo_bar_baz", "foo_bar_baz"),
		Entry(nil, "NS=1;ID=5", "NS=1;ID=5"),
		Entry(nil, "foo-bar", "foo-bar"),
		Entry(nil, "foo.bar", "foo.bar"),
		Entry(nil, "fooBarBaz.qux", "foo_bar_baz.qux"),
		Entry(nil, "setXChannel", "set_x_channel"),
		Entry(nil, "fooXBar", "foo_x_bar"),
		Entry(nil, "fooXY", "foo_x_y"),
		Entry(nil, "setXYChannel", "set_x_y_channel"),
		Entry(nil, "Content-Type", "Content-Type"),
		Entry(nil, "customScale", "custom_scale"),
		Entry(nil, "", ""),
	)

	Describe("snakeKeys", func() {
		It("Should convert keys recursively through maps and lists", func() {
			out := snakeKeys(map[string]any{
				"sampleRate": 10,
				"channels": []any{map[string]any{
					"customScale": map[string]any{"preScaledUnits": "V"},
					"minVal":      -1,
				}},
			})
			ch := out["channels"].([]any)[0].(map[string]any)
			Expect(ch).To(HaveKey("min_val"))
			scale, ok := ch["custom_scale"].(map[string]any)
			Expect(ok).To(BeTrue())
			Expect(scale).To(HaveKey("pre_scaled_units"))
			Expect(out).To(HaveKey("sample_rate"))
		})

		It("Should keep the snake_case value when both spellings are present", func() {
			out := snakeKeys(map[string]any{"sampleRate": 1, "sample_rate": 2})
			Expect(out).To(HaveLen(1))
			Expect(out).To(HaveKeyWithValue("sample_rate", 2))
		})

		It("Should not mutate the input", func() {
			in := map[string]any{
				"sampleRate": 10,
				"channels":   []any{map[string]any{"minVal": -1}},
			}
			snakeKeys(in)
			Expect(in).To(HaveKey("sampleRate"))
			Expect(in["channels"].([]any)[0]).To(HaveKey("minVal"))
		})
	})
})
