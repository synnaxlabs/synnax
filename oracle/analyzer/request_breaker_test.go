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

var _ = Describe("Request option breaker", func() {
	analyze := func(options string) (bool, string) {
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
				`+options+`
			}
		`, "test", loader)
		return diag.Ok(), diag.String()
	}

	It("Should refuse an @request option that takes the name of another @request "+
		"option in another casing", func() {
		ok, text := analyze(`@request fooBar "a."
				@request foo_bar "b."`)
		Expect(ok).To(BeFalse())
		Expect(text).To(ContainSubstring(
			"@request foo_bar takes the name of another retrieve option",
		))
	})

	It("Should refuse an @request option that takes the name of a line in another "+
		"casing", func() {
		ok, _ := analyze(`@request Name "a."`)
		Expect(ok).To(BeFalse())
	})
})
