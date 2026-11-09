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

var _ = Describe("TS adjacent union generation", func() {
	It("Should carry each payload beside the tag", func(ctx SpecContext) {
		loader := NewMockFileLoader()
		resp := MustGenerate(ctx, `
			@ts output "client/ts/src/board"

			Board struct {
				key uuid @key

				action SetNodePosition {
					x float64
				}
			}
		`, "board", loader, types.New(types.DefaultOptions()))
		ExpectContent(resp, "types.gen.ts").
			ToContain(
				"export const setNodePositionPayloadZ = z.object({",
				"export const setNodePositionActionZ = z.object({",
				`type: z.literal("set_node_position"),`,
				"setNodePosition: setNodePositionPayloadZ,",
				"/** Action holds one board mutation. */",
				`export const actionZ: z.ZodType<Action> = z.discriminatedUnion("type", [`,
				"export type Action = SetNodePositionAction;",
			).
			ToNotContain("...setNodePositionPayloadZ.shape")
	})
})

var _ = Describe("TS any fields", func() {
	It("Should read an absent bare any field as undefined", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			@ts output "client/ts/src/board"

			Cell struct {
				value any
				fill  any = 0
			}
		`, "board", NewMockFileLoader(), types.New(types.DefaultOptions()))
		ExpectContent(resp, "types.gen.ts").
			ToContain("value: z.unknown().optional(),").
			ToNotContain("fill: z.unknown().optional()")
	})
})
