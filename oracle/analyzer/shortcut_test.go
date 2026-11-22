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
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Filter shortcut", func() {
	It("Should refuse a @ts shortcut other than omit", func() {
		loader := NewMockFileLoader()
		AddFilterSchema(loader)
		_, diag := Analyze(`
			import "schemas/x/filter"

			@go output "out"

			Thing struct {
				key  uuid   { @key }
				name string { @filter }
			}
			Where filter Thing {
				key
				name
				@ts shortcut keep
			}
		`, "test", loader)
		Expect(diag.Ok()).To(BeFalse())
		Expect(diag.String()).To(ContainSubstring(
			"filter Where: @ts shortcut takes only omit",
		))
	})
})
