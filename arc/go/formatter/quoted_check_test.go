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

var _ = Describe("Format of quoted channel names, checker cases", func() {
	It("Should keep the space between a keyword and a quoted channel name", func() {
		out := formatter.Format(
			"func f() f64 {\n\tif 'pt-01' > 5 {\n\t\treturn 'pt-01'\n\t}\n\treturn 0.0\n}\n",
		)
		Expect(out).To(ContainSubstring("if 'pt-01' > 5"))
		Expect(out).To(ContainSubstring("return 'pt-01'"))
	})
})
