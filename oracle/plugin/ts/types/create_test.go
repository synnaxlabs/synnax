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
	"github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Derived New from @create", func() {
	var (
		loader      *MockFileLoader
		typesPlugin *types.Plugin
	)
	BeforeEach(func() {
		loader = NewMockFileLoader()
		typesPlugin = types.New(types.DefaultOptions())
	})

	It("Should generate a z.input New that omits @output fields", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			"export const newZ = thingZ",
			".omit({ author: true })",
			"export interface New extends z.input<typeof newZ> {}",
		)
	})

	It("Should derive New under its own name when the base is renamed", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
				@ts name "Payload"
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		// The derived New must emit as newZ/New, extending the base's payloadZ, not
		// re-emit the base's renamed payloadZ/Payload (which would be a duplicate
		// symbol with an empty body).
		ExpectContent(resp, "types.gen.ts").ToContain(
			"export const payloadZ = z.object(",
			"export const newZ = payloadZ",
			".omit({ author: true })",
			"export interface New extends z.input<typeof newZ> {}",
		)
	})
})
