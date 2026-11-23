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

var _ = Describe("EscapeField", func() {
	DescribeTable("Should append an underscore to a reserved name",
		func(input, expected string) {
			Expect(keywords.EscapeField(input)).To(Equal(expected))
		},
		Entry("def", "def", "def_"),
		Entry("class", "class", "class_"),
		Entry("import", "import", "import_"),
		Entry("yield", "yield", "yield_"),
		Entry("lambda", "lambda", "lambda_"),
		Entry("True", "True", "True_"),
		Entry("an ABCMeta method", "register", "register_"),
		Entry("a deprecated v1 method", "json", "json_"),
		Entry("a model method", "model_dump", "model_dump_"),
	)

	DescribeTable("Should return any other name unchanged",
		func(input string) {
			Expect(keywords.EscapeField(input)).To(Equal(input))
		},
		Entry("regular identifier", "channel"),
		Entry("camelCase identifier", "myVariable"),
		Entry("snake_case identifier", "my_variable"),
		Entry("empty string", ""),
	)
})

var _ = Describe("Protected", func() {
	DescribeTable("Should report a name in a protected namespace",
		func(input string, expected bool) {
			Expect(keywords.Protected(input)).To(Equal(expected))
		},
		Entry("an escaped model_dump", "model_dump_", true),
		Entry("a model_validate prefix", "model_validate_json", true),
		Entry("another model name", "model_config", false),
		Entry("a regular name", "channel", false),
	)
})
