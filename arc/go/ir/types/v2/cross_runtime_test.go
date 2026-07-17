// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"os"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir/types/v2"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// cppIRHeaderPath is the C++ runtime's IR header, relative to this package
// directory. It holds param-name constants that must mirror the Go ones in
// edge.go. The two are hand-maintained in separate languages; commit d7a8a9b666
// fixed a regression where they drifted ("lhs_input" vs "a") and binary math ops
// failed to load on the driver's C++ runtime.
const cppIRHeaderPath = "../../../../cpp/ir/ir.h"

var cppStringConstRe = regexp.MustCompile(`inline const std::string (\w+)\s*=\s*"([^"]*)";`)

// parseCppStringConstants maps each `inline const std::string NAME = "VALUE";`
// declaration in src to NAME->VALUE.
func parseCppStringConstants(src []byte) map[string]string {
	out := map[string]string{}
	for _, m := range cppStringConstRe.FindAllStringSubmatch(string(src), -1) {
		out[m[1]] = m[2]
	}
	return out
}

var _ = Describe("Cross-runtime IR contract", Ordered, func() {
	var cppConstants map[string]string
	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		cppConstants = parseCppStringConstants(MustSucceed(os.ReadFile(cppIRHeaderPath)))
		Expect(cppConstants).ToNot(BeEmpty(), "no string constants parsed from %s", cppIRHeaderPath)
	})

	// When adding a shared IR param-name constant, define it on both sides and
	// add an Entry below so any future drift stays caught.
	DescribeTable("Go and C++ IR param-name constants must agree",
		func(cppName, goValue string) {
			Expect(cppConstants).To(HaveKeyWithValue(cppName, goValue))
		},
		Entry("default output param", "default_output_param", v2.DefaultOutputParam),
		Entry("default input param", "default_input_param", v2.DefaultInputParam),
		Entry("lhs input param", "lhs_input_param", v2.LHSInputParam),
		Entry("rhs input param", "rhs_input_param", v2.RHSInputParam),
	)

	// Guard against drift-by-addition: a new constant added to the C++ header
	// without a matching Go agreement Entry above must fail the suite.
	It("Should cover every C++ ir.h param-name constant", func() {
		covered := set.New(
			"default_output_param",
			"default_input_param",
			"lhs_input_param",
			"rhs_input_param",
		)
		var uncovered []string
		for name := range cppConstants {
			if !covered.Contains(name) {
				uncovered = append(uncovered, name)
			}
		}
		Expect(uncovered).To(BeEmpty(),
			"C++ ir.h declares param-name constants with no Go agreement Entry: %v", uncovered)
	})
})
