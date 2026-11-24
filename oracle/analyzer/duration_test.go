// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Duration defaults", func() {
	var loader *MockFileLoader
	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Files["schemas/x/telem"] = `
			TimeSpan int64
			NumericTimeSpan = TimeSpan
		`
	})

	DescribeTable("converts a duration literal to nanoseconds",
		func(lit string, want int64) {
			table, diag := Analyze(`
				import "schemas/x/telem"
				Cfg struct {
					timeout telem.TimeSpan = `+lit+`
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := resolution.MustForm[resolution.StructForm](
				table.MustGet("test.Cfg"),
			)
			Expect(form.Fields[0].Default.Kind).To(Equal(resolution.ValueKindInt))
			Expect(form.Fields[0].Default.IntValue).To(Equal(want))
			Expect(form.Fields[0].Default.Duration).To(Equal(lit))
		},
		Entry("nanoseconds", "250ns", int64(250)),
		Entry("microseconds", "500us", int64(500_000)),
		Entry("milliseconds", "1ms", int64(1_000_000)),
		Entry("seconds", "5s", int64(5_000_000_000)),
		Entry("minutes", "2m", int64(120_000_000_000)),
		Entry("hours", "1h", int64(3_600_000_000_000)),
		Entry("a compound duration", "1m30s", int64(90_000_000_000)),
		Entry("a fractional duration", "1.5ms", int64(1_500_000)),
		Entry("a negative duration", "-2s", int64(-2_000_000_000)),
	)

	It("Should accept a duration on an alias of TimeSpan", func() {
		_, diag := Analyze(`
			import "schemas/x/telem"
			Cfg struct {
				timeout telem.NumericTimeSpan = 5s
			}
		`, "test", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	})

	It("Should accept a duration on a distinct type over TimeSpan", func() {
		_, diag := Analyze(`
			import "schemas/x/telem"
			Timeout telem.TimeSpan
			Cfg struct {
				timeout Timeout = 5s
			}
		`, "test", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	})

	It("Should accept a duration on an override of a TimeSpan field", func() {
		_, diag := Analyze(`
			import "schemas/x/telem"
			Parent struct {
				timeout telem.TimeSpan = 1s
			}
			Child struct extends Parent {
				timeout = 5s
			}
		`, "test", loader)
		Expect(diag.Ok()).To(BeTrue(), diag.String())
	})

	It("Should reject a duration on a field that is not a TimeSpan", func() {
		_, diag := Analyze(`
			Cfg struct {
				count int64 = 5s
			}
		`, "test", loader)
		Expect(diag.Error()).To(ContainSubstring(
			`default 5s on field "count" in "Cfg" has a duration unit, but the ` +
				`field's type is not telem.TimeSpan`,
		))
	})

	It("Should reject a duration that is not a whole number of nanoseconds", func() {
		_, diag := Analyze(`
			import "schemas/x/telem"
			Cfg struct {
				timeout telem.TimeSpan = 1.5ns
			}
		`, "test", loader)
		Expect(diag.Error()).To(ContainSubstring(
			"duration 1.5ns is not a whole number of nanoseconds",
		))
	})

	It("Should reject a duration that does not fit in an int64", func() {
		_, diag := Analyze(`
			import "schemas/x/telem"
			Cfg struct {
				timeout telem.TimeSpan = 3000000h
			}
		`, "test", loader)
		Expect(diag.Error()).To(ContainSubstring(
			"duration 3000000h does not fit in an int64",
		))
	})
})
