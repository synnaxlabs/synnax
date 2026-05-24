// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package casing_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
)

var _ = Describe("FieldSnake", func() {
	DescribeTable("should preserve valid snake_case identifiers",
		func(input, expected string) {
			Expect(casing.FieldSnake(input)).To(Equal(expected))
		},
		Entry("axis key x1", "x1", "x1"),
		Entry("axis key y4", "y4", "y4"),
		Entry("single letter", "x", "x"),
		Entry("snake with digits", "value_42", "value_42"),
		Entry("snake without digits", "label_direction", "label_direction"),
		Entry("snake with trailing digit", "axis_1", "axis_1"),
		Entry("snake with leading word and digit", "channel_2_key", "channel_2_key"),
	)

	DescribeTable("should snake-case mixed-case identifiers",
		func(input, expected string) {
			Expect(casing.FieldSnake(input)).To(Equal(expected))
		},
		Entry("camelCase", "clientX", "client_x"),
		Entry("PascalCase", "PascalCaseField", "pascal_case_field"),
		Entry("camel with digits", "Int8Value", "int_8_value"),
		Entry("digit-letter PascalCase", "Utf8Bytes", "utf_8_bytes"),
		Entry("acronym tail", "EntityID", "entity_id"),
		Entry("snake mixed with caps", "Already_Snake_Case", "already_snake_case"),
	)

	It("should return the empty string unchanged", func() {
		Expect(casing.FieldSnake("")).To(Equal(""))
	})
})
