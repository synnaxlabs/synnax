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

	Describe("multiple_of", func() {
		It("Should read a duration step in nanoseconds", func() {
			table, diag := Analyze(`
				import "schemas/x/telem"
				Cfg struct {
					keep_alive telem.TimeSpan = 60s { @validate multiple_of 1s }
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
			form := resolution.MustForm[resolution.StructForm](
				table.MustGet("test.Cfg"),
			)
			step := form.Fields[0].Domains["validate"].Expressions[0].Values[0]
			Expect(step.IntValue).To(Equal(int64(1_000_000_000)))
			Expect(step.Duration).To(Equal("1s"))
		})

		It("Should accept a step on an integer", func() {
			_, diag := Analyze(`
				Cfg struct {
					count int32 { @validate multiple_of 4 }
				}
			`, "test", loader)
			Expect(diag.Ok()).To(BeTrue(), diag.String())
		})

		DescribeTable("Should reject a rule that cannot apply",
			func(source, want string) {
				_, diag := Analyze(source, "test", loader)
				Expect(diag.Ok()).To(BeFalse())
				Expect(diag.String()).To(ContainSubstring(want))
			},
			Entry("on a string", `
				Cfg struct { name string { @validate multiple_of 2 } }
			`, "test.Cfg.name has a multiple_of rule, but its type is not a number"),
			Entry("with a zero step", `
				Cfg struct { count int32 { @validate multiple_of 0 } }
			`, "test.Cfg.count has a multiple_of rule, but its step is not positive"),
			Entry("with a negative step", `
				Cfg struct { count int32 { @validate multiple_of -5 } }
			`, "test.Cfg.count has a multiple_of rule, but its step is not positive"),
			Entry("on a float", `
				Cfg struct { rate float64 { @validate multiple_of 1 } }
			`, "test.Cfg.rate has a multiple_of rule, but its type is a float"),
			Entry("with a fractional step on an integer", `
				Cfg struct { count int32 { @validate multiple_of 0.5 } }
			`, "test.Cfg.count has a multiple_of rule, but its step is not a whole number"),
			Entry("with a step that does not fit the type", `
				Cfg struct { count uint8 { @validate multiple_of 300 } }
			`, "test.Cfg.count has a multiple_of rule, but its step does not fit in a uint8"),
			Entry("with a step where its type sets one", `
				Span int64 { @validate multiple_of 1000 }
				Cfg struct { span Span { @validate multiple_of 7 } }
			`, "test.Cfg.span has a multiple_of rule, but its type already sets one"),
			Entry("with a duration on an integer", `
				Cfg struct { count int64 { @validate multiple_of 1s } }
			`, "test.Cfg.count has a multiple_of rule of 1s, but its type is not telem.TimeSpan"),
			Entry("with a duration on another rule", `
				Cfg struct { count int64 { @validate min 1s } }
			`, "test.Cfg.count has a min rule of 1s, but its type is not telem.TimeSpan"),
		)
	})
})
