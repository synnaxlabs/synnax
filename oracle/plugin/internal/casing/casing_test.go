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

var _ = Describe("TypeSnake", func() {
	// Regression: lo.SnakeCase collapses leading acronyms with no preceding
	// lowercase because the splitter has already consumed the boundary
	// character once it lands on the first cap-cap pair. SetXChannel becomes
	// set_xchannel instead of set_x_channel; URLValue becomes urlvalue
	// instead of url_value. TypeSnake nudges the splitter by inserting a
	// separator before any cap-cap-lower run.
	DescribeTable("should split PascalCase acronym-followed-by-word boundaries",
		func(input, expected string) {
			Expect(casing.TypeSnake(input)).To(Equal(expected))
		},
		Entry("leading-acronym word", "SetXChannel", "set_x_channel"),
		Entry("longer acronym", "SetXAxisChannel", "set_x_axis_channel"),
		Entry("acronym at start", "URLValue", "url_value"),
		Entry("trailing acronym preserved", "EntityID", "entity_id"),
		Entry("two-letter acronym in middle", "ParseHTTPRequest", "parse_http_request"),
		Entry("plain camel still splits", "clientX", "client_x"),
		Entry("digit-letter boundary still splits", "Int8Value", "int_8_value"),
		Entry("already snake passes through unchanged", "x1", "x1"),
	)
})

var _ = Describe("TypePascal", func() {
	// Regression: lo.PascalCase shares the lo.SnakeCase blind spot for leading
	// acronyms, so "SetXChannel" round-trips to "SetXchannel" instead of
	// staying "SetXChannel". TypePascal routes through TypeSnake first.
	DescribeTable("should preserve acronym-followed-by-word boundaries",
		func(input, expected string) {
			Expect(casing.TypePascal(input)).To(Equal(expected))
		},
		Entry("PascalCase with leading acronym", "SetXChannel", "SetXChannel"),
		Entry("snake input", "set_x_channel", "SetXChannel"),
		Entry("camel input", "clientX", "ClientX"),
		Entry("snake with acronym word", "url_value", "UrlValue"),
	)
})

var _ = Describe("TypeCamel", func() {
	DescribeTable("should preserve acronym-followed-by-word boundaries",
		func(input, expected string) {
			Expect(casing.TypeCamel(input)).To(Equal(expected))
		},
		Entry("PascalCase with leading acronym", "SetXChannel", "setXChannel"),
		Entry("snake input", "set_x_channel", "setXChannel"),
		Entry("camel input", "clientX", "clientX"),
		Entry("snake with acronym word", "url_value", "urlValue"),
	)
})

var _ = Describe("PascalAcronym", func() {
	DescribeTable("should fully upper-case whole acronym segments only",
		func(input, expected string) {
			Expect(casing.PascalAcronym(input)).To(Equal(expected))
		},
		Entry("leading acronym", "ai_voltage", "AIVoltage"),
		Entry("embedded acronym", "accel_4_wire_dc_voltage", "Accel4WireDCVoltage"),
		Entry("rtd acronym", "ai_rtd", "AIRTD"),
		Entry("no acronym", "force_bridge_table", "ForceBridgeTable"),
		Entry("word containing acronym is not mangled", "email_address", "EmailAddress"),
	)
})

var _ = Describe("VariantTypeName", func() {
	DescribeTable("should factor a repeated union acronym, else prefix the union name",
		func(union, variant, expected string) {
			Expect(casing.VariantTypeName(union, variant)).To(Equal(expected))
		},
		Entry("acronym union factors the shared prefix", "AIChannel", "ai_voltage", "AIVoltageChannel"),
		Entry("multi-word variant", "AIChannel", "ai_force_bridge_table", "AIForceBridgeTableChannel"),
		Entry("ci union", "CIChannel", "ci_two_edge_sep", "CITwoEdgeSepChannel"),
		Entry("ao union", "AOChannel", "ao_voltage", "AOVoltageChannel"),
		Entry("plain union keeps union prefix", "Scale", "linear", "ScaleLinear"),
		Entry("variant not repeating the acronym keeps union prefix", "AIChannel", "voltage", "AIChannelVoltage"),
	)
})
