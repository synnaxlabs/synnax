// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/formatter"
)

var _ = Describe("Quoted channel names in the formatter (break)", func() {
	DescribeTable(
		"F formats a quoted channel name like a bare one",
		func(input string) {
			Expect(formatter.Format(input)).To(Equal(input))
		},
		Entry("F0 bare return", "func f() f64 {\n    return pt\n}\n"),
		Entry("F1 return", "func f() f64 {\n    return 'pt-01'\n}\n"),
		Entry(
			"F0b bare if",
			"func f() {\n    if pt > 1 {\n        out = 1\n    }\n}\n",
		),
		Entry(
			"F2 if",
			"func f() {\n    if 'pt-01' > 1 {\n        'out' = 1\n    }\n}\n",
		),
		Entry("F3 flow", "'pt-01' -> 'out'\n"),
		Entry("F4 assign", "func f() {\n    'out' = 'pt-01' * 2\n}\n"),
		Entry("F6 bare gap before a paren", "trig -> a (2.0) -> out\n"),
		Entry("F5 quoted gap before a paren", "trig -> 'a' (2.0) -> out\n"),
		Entry("F7 not", "func f() u8 {\n    return not 'flag'\n}\n"),
	)
})
