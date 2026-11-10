// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/group"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("TS Group Plugin", func() {
	var (
		loader *MockFileLoader
		p      *group.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		p = group.New(group.DefaultOptions())
		loader.Add("schemas/resource", `
			@ts output "client/ts/src/resource"

			Type enum {
				device           = "device"
				group            = "group"
				rack             = "rack"
				schematic_symbol = "schematic_symbol"
			}
		`)
		loader.Add("schemas/group", `
			import "schemas/resource"

			@ts output "client/ts/src/group"

			Key = uuid

			Group struct {
				key   Key {
					@key
				}
				scope resource.Type

				@resource type "group"
			}
		`)
	})

	It("Should have the name, domains, and requirements of a TS plugin", func() {
		Expect(p.Name()).To(Equal("ts/group"))
		Expect(p.Domains()).To(Equal([]string{"ts"}))
		Expect(p.Requires()).To(Equal([]string{"ts/types"}))
	})

	It("Should emit no files when no field carries @group", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			import "schemas/group"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key?

				@resource type "rack"
			}
		`, "rack", loader, p)
		Expect(resp.Files).To(BeEmpty())
	})

	It("Should emit the rule of each member next to the group", func(ctx SpecContext) {
		resp := MustGenerate(ctx, `
			import "schemas/group"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group scope device
					@group pinned
				}

				@resource type "rack"
			}

			Symbol struct {
				key   uuid {
					@key
				}
				group group.Key? {
					@group scope schematic_symbol
				}

				@resource type "schematic_symbol"
			}
		`, "rack", loader, p)
		Expect(resp.Files).To(HaveLen(1))
		Expect(resp.Files[0].Path).To(Equal("client/ts/src/group/members.gen.ts"))
		ExpectContent(resp, "members.gen.ts").
			ToContain(
				`import { resource } from "@/resource";`,
				"scope: resource.Type;",
				`rack: { scope: "device", pinned: true },`,
				`schematic_symbol: { scope: "schematic_symbol", pinned: false },`,
				"} as const satisfies Partial<Record<resource.Type, MemberRule>>;",
				"export type MemberType = keyof typeof MEMBER_RULES;",
			).
			ToPreserveOrder("rack:", "schematic_symbol:")
	})

	It(
		"Should import the resource types from the types of their own module",
		func(ctx SpecContext) {
			loader.Add("schemas/group", `
				@ts output "client/ts/src/group"

				Key = uuid

				Type enum {
					channel = "channel"
				}

				Group struct {
					key   Key {
						@key
					}
					scope Type
				}

				Channel struct {
					key   uint32 {
						@key
					}
					group Key? {
						@group scope channel
					}

					@resource type "channel"
				}
			`)
			resp := MustGenerateMulti(ctx, loader, p)
			ExpectContent(resp, "members.gen.ts").
				ToContain(
					`import { Type } from "@/group/types.gen";`,
					"scope: Type;",
					`channel: { scope: "channel", pinned: false },`,
				)
		},
	)

	It("Should return the error of a malformed @group", func(ctx SpecContext) {
		req := MustGenerateRequest(ctx, `
			import "schemas/group"

			Rack struct {
				key   uint32 {
					@key
				}
				group group.Key? {
					@group pinned
				}

				@resource type "rack"
			}
		`, "rack", loader)
		Expect(p.Generate(req)).Error().
			To(MatchError("rack.Rack.group: @group needs a scope"))
	})
})
