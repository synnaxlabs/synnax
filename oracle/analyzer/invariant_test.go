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
		Entry("exempts a nullable field, whose absence is itself meaningful",
			`
				Cfg struct {
					visible bool? = true
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
