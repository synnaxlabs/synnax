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
			"export const thingZ",
			`export interface New extends Omit<z.input<typeof thingZ>, "author"> {}`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})

	It("Should derive a generic factory New that partials defaulted fields and keeps non-defaulted keys required", func(ctx SpecContext) {
		source := `
			@ts output "client/ts/src/thing"

			Thing struct<Properties extends record = record> {
				key        string         @key
				name       string
				configured bool = false
				properties Properties
				@create
				@ts concrete_types
			}
		`
		resp := MustGenerate(ctx, source, "thing", loader, typesPlugin)
		ExpectContent(resp, "types.gen.ts").ToContain(
			`optional.Optional<Thing<Properties>, "configured">`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
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
		// The derived New must emit under its own name (New), referencing the base's
		// renamed payloadZ schema, not re-emit a renamed payloadZ/Payload (which would
		// be a duplicate symbol with an empty body).
		ExpectContent(resp, "types.gen.ts").ToContain(
			"export const payloadZ = z.object(",
			`export interface New extends Omit<z.input<typeof payloadZ>, "author"> {}`,
		)
		ExpectContent(resp, "types.gen.ts").ToNotContain("export const newZ")
	})
})
