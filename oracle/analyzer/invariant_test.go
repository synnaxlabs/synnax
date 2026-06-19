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
	"github.com/synnaxlabs/oracle/analyzer"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Default Invariant", func() {
	var loader *MockFileLoader
	BeforeEach(func() { loader = NewMockFileLoader() })

	// A required field carrying a static default must default to its type's zero
	// value, or its zero value must be invalid. The analyzer can settle the bool and
	// integer-enum cases without bounds, and abstains (accepts) everywhere else so it
	// never raises a false positive.
	DescribeTable("a required field with a static default",
		func(ctx SpecContext, source, wantErr string) {
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			if wantErr == "" {
				Expect(diag.Ok()).To(BeTrue())
			} else {
				Expect(diag.Error()).To(ContainSubstring(wantErr))
			}
		},
		Entry("rejects a bool defaulting to true (false is a valid non-zero-distinct value)",
			`
				Cfg struct {
					visible bool = true
				}
			`,
			"bool default"),
		Entry("accepts a bool defaulting to false (the zero value)",
			`
				Cfg struct {
					visible bool = false
				}
			`,
			""),
		Entry("rejects an integer-enum default that is not the zeroth member",
			`
				Priority enum {
					low = 0
					high = 1
				}

				Cfg struct {
					priority Priority = PriorityHigh
				}
			`,
			"integer-enum default that is not the zeroth member"),
		Entry("accepts an integer-enum default on the zeroth member (the zero value)",
			`
				Priority enum {
					low = 0
					high = 1
				}

				Cfg struct {
					priority Priority = PriorityLow
				}
			`,
			""),
		Entry("accepts a string-enum default (a string enum's zero value is the empty string)",
			`
				Level enum {
					h1 = "h1"
					h2 = "h2"
				}

				Cfg struct {
					level Level = LevelH2
				}
			`,
			""),
		Entry("abstains on a non-zero numeric default (numeric bounds are a follow-up)",
			`
				Cfg struct {
					precision int32 = 2
				}
			`,
			""),
	)
})

var _ = Describe("Optional Default Invariant", func() {
	var loader *MockFileLoader
	BeforeEach(func() { loader = NewMockFileLoader() })

	// A field may be nullable (`?`) or carry a default, but never both. The check is
	// structural and fires for every field type, so the table exercises bools,
	// strings, numbers, enums, and struct references, not just the bool case.
	DescribeTable("a field that is both nullable and defaulted",
		func(ctx SpecContext, source, wantErr string) {
			_, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
			if wantErr == "" {
				Expect(diag.Ok()).To(BeTrue())
			} else {
				Expect(diag.Error()).To(ContainSubstring(wantErr))
			}
		},
		Entry("rejects a nullable bool with a default",
			`
				Cfg struct {
					visible bool? = true
				}
			`,
			`field "visible" in "Cfg" is both nullable`),
		Entry("rejects a nullable string with a default",
			`
				Cfg struct {
					name string? = "untitled"
				}
			`,
			`field "name" in "Cfg" is both nullable`),
		Entry("rejects a nullable number with a default",
			`
				Cfg struct {
					precision int32? = 2
				}
			`,
			`field "precision" in "Cfg" is both nullable`),
		Entry("rejects a nullable enum with a default",
			`
				Priority enum {
					low = 0
					high = 1
				}

				Cfg struct {
					priority Priority? = PriorityHigh
				}
			`,
			`field "priority" in "Cfg" is both nullable`),
		Entry("rejects a nullable struct reference with a default",
			`
				Inner struct {
					x int32 = 0
				}

				Cfg struct {
					inner Inner? = nil
				}
			`,
			`field "inner" in "Cfg" is both nullable`),
		Entry("accepts a plain nullable field with no default",
			`
				Cfg struct {
					name string?
				}
			`,
			""),
		Entry("accepts a required defaulted field",
			`
				Cfg struct {
					name string = "untitled"
				}
			`,
			""),
	)
})
