// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Type Definition Generation", func() {
	var (
		loader *MockFileLoader
		p      *types.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = types.New(types.DefaultOptions())
	})

	Describe("Distinct types with validation", func() {
		It("Should apply min_length validation to a distinct string type", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Slug string {
					@validate { min_length 3 }
				}
			`
			resp := MustGenerate(ctx, source, "slug", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring(".min(3)"))
		})

		It("Should emit ts to_number wrapper on a distinct numeric type", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Count uint32 {
					@ts to_number
				}
			`
			resp := MustGenerate(ctx, source, "count", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("countZ"))
		})

		It("Should emit ts to_string wrapper on a distinct numeric type", func(ctx SpecContext) {
			source := `
				@ts output "out"

				ID uint64 {
					@ts to_string
				}
			`
			resp := MustGenerate(ctx, source, "id", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("ID"))
		})
	})

	Describe("Aliased array types", func() {
		It("Should wrap an array alias with array.nullishToEmpty", func(ctx SpecContext) {
			source := `
				@ts output "out"

				Tags = string[]
			`
			resp := MustGenerate(ctx, source, "tags", loader, p)
			content := MustContentOf(resp, "types.gen.ts")
			Expect(content).To(ContainSubstring("array.nullishToEmpty"))
		})
	})
})
