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

var _ = Describe("Request option check", func() {
	It(
		"Should refuse an @request option that takes the name of a derived option",
		func() {
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
				@request where "takes the name of the derived where option."
			}
		`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"@request where takes the name of another retrieve option",
			))
		},
	)

	It(
		"Should refuse an @request option that takes the name of the one option",
		func() {
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
				@request one "takes the name of the one retrieve option."
			}
		`, "test", loader)
			Expect(diag.Ok()).To(BeFalse())
			Expect(diag.String()).To(ContainSubstring(
				"@request one takes the name of another retrieve option",
			))
		},
	)
})
