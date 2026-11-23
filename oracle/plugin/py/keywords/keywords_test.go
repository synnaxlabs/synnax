// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package keywords_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/py/keywords"
)

var _ = Describe("Escape", func() {
	DescribeTable("should append underscore to reserved Python keywords",
		func(input, expected string) {
			Expect(keywords.Escape(input)).To(Equal(expected))
		},
		Entry("def", "def", "def_"),
		Entry("class", "class", "class_"),
		Entry("import", "import", "import_"),
		Entry("yield", "yield", "yield_"),
		Entry("lambda", "lambda", "lambda_"),
		Entry("True", "True", "True_"),
	)

	DescribeTable("should return the name unchanged for non-reserved words",
		func(input string) {
			Expect(keywords.Escape(input)).To(Equal(input))
		},
		Entry("regular identifier", "channel"),
		Entry("camelCase identifier", "myVariable"),
		Entry("snake_case identifier", "my_variable"),
		Entry("empty string", ""),
	)

	DescribeTable("EscapeField should escape keywords and BaseModel attributes",
		func(input, expected string) {
			Expect(keywords.EscapeField(input)).To(Equal(expected))
		},
		Entry("a keyword", "from", "from_"),
		Entry("an ABCMeta method", "register", "register_"),
		Entry("a deprecated v1 method", "json", "json_"),
		Entry("a model method", "model_dump", "model_dump_"),
		Entry("a regular field", "channel", "channel"),
	)
})
