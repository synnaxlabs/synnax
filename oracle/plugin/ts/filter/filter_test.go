// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter_test

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/filter"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestTSFilter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin TS Filter Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = Describe("TS Filter Plugin", func() {
	var (
		loader *MockFileLoader
		p      *filter.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", "@ts output \"x/ts/src/filter\"\n"+FilterSchema)
		p = filter.New(filter.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("ts/filter"))
		})

		It("Should filter on ts domain", func() {
			Expect(p.Domains()).To(Equal([]string{"ts"}))
		})

		It("Should require ts/types", func() {
			Expect(p.Requires()).To(Equal([]string{"ts/types"}))
		})
	})

	Describe("Generate", func() {
		const thing = `
			import "schemas/x/filter"

			@ts output "client/ts/src/thing"

			Span struct {
				start int64
			}
			Thing struct {
				key        uuid     { @key }
				name       string
				created_at int64
				tags       string[]

				@ts name "Payload"
			}
		`

		It("Should check the shared fields and tag of a union", func(ctx SpecContext) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@ts output "client/ts/src/view"

				Type enum {
					first = "first"
				}
				Base struct {
					key  uuid   { @key }
					name string
				}
				View union on type Type extends Base {
					first { size int32 }
				}

				Where filter View {
					name
					type
				}
			`, "view", loader, p)
			ExpectContent(resp, "where.gen.ts").ToContain(
				"!filter.matchesString(where.name, record.name)",
				"!filter.matchesSet(where.type, record.type)",
			)
		})

		It("Should check each bare line with its operator", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
					name
					created_at
					tags
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.ts").
				ToContain(
					`import { filter } from "@synnaxlabs/x";`,
					`import { Payload, Where } from "@/thing/types.gen";`,
					"export const matches = (record: Payload, where?: Where): boolean => {",
					"if (where == null) return true;",
					" * @returns whether only the Core can check where, because it holds a "+
						"regex. An absent\n * where is not server-computed.",
					"export const serverComputed = (where?: Where): boolean =>",
					"where != null && (",
					"!filter.matchesSet(where.key, record.key)",
					"!filter.matchesString(where.name, record.name)",
					"!filter.matchesOrdered(where.createdAt, record.createdAt)",
					"!filter.matchesContains(where.tags, record.tags)",
					"where.name?.regex != null",
				).
				ToNotContain("where.key?.regex", "lines", "Lines", "@throws")
		})

		It("Should walk the tree", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.ts").
				ToContain(
					"!where.and.every((w) => matches(record, w))",
					"!where.or.some((w) => matches(record, w))",
					"where.not != null && matches(record, where.not)",
					"where.and?.some((w) => serverComputed(w)) === true",
					"where.or?.some((w) => serverComputed(w)) === true",
					"(where.not != null && serverComputed(where.not)));",
				)
		})

		It("Should take typed lines from the Lines table", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
					overlaps Span
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.ts").
				ToContain(
					"export interface Lines {",
					"export const matches = (lines: Lines, record: Payload, where?: "+
						"Where): boolean => {",
					"!where.and.every((w) => matches(lines, record, w))",
					" * @throws {Error} if where is server-computed",
					"export const serverComputed = (lines: Lines, where?: Where): "+
						"boolean =>",
					"where.and?.some((w) => serverComputed(lines, w)) === true",
					`overlaps: ((record: Payload, value: NonNullable<Where["overlaps"]>)`+
						` => boolean) | null;`,
					`if (lines.overlaps == null) throw new Error("only the Core can `+
						`check overlaps");`,
					"if (!lines.overlaps(record, where.overlaps)) return false;",
					"(where.overlaps != null && lines.overlaps == null)",
				)
		})

		It("Should read bare lines from the response type", func(ctx SpecContext) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@ts output "client/ts/src/thing"

				Thing struct {
					key  uuid   { @key }
					name string
				}
				Wire struct {
					key  uuid
					name string
				}
				Where filter Thing {
					name

					@response type Wire
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.ts").
				ToContain(
					`import { Where, Wire } from "@/thing/types.gen";`,
					"record: Wire",
					"!filter.matchesString(where.name, record.name)",
				)
		})

		It("Should reject a bare line the response type does not hold", func(
			ctx SpecContext,
		) {
			req := MustGenerateRequest(ctx, `
				import "schemas/x/filter"

				@ts output "client/ts/src/thing"

				Thing struct {
					key  uuid   { @key }
					name string
				}
				Wire struct {
					key uuid
				}
				Where filter Thing {
					name

					@response type Wire
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"filter thing.Where: the bare line name names no field of Wire, " +
					"which the cache holds",
			)))
		})

		It("Should reject a response type with a type parameter and no default", func(
			ctx SpecContext,
		) {
			req := MustGenerateRequest(ctx, `
				import "schemas/x/filter"

				@ts output "client/ts/src/thing"

				Thing struct<T> {
					key   uuid { @key }
					value T
				}
				Where filter Thing {
					key
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(
				"filter thing.Where: the type parameter T of Thing needs a default, since " +
					"the matcher names Thing with no type arguments",
			))
		})

		It("Should reject an operator member it cannot check", func(ctx SpecContext) {
			loader.Add(
				"schemas/x/filter",
				`@ts output "x/ts/src/filter"`+strings.Replace(
					FilterSchema,
					"in T[]?\n\t@pb omit\n}\nOrdered",
					"in T[]?\n\tgt T?\n\t@pb omit\n}\nOrdered",
					1,
				),
			)
			req := MustGenerateRequest(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(
				"filter thing.Where: line key: ts/filter cannot check the member gt of Set",
			))
		})

		It("Should skip a filter with no TypeScript output", func(ctx SpecContext) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				Thing struct {
					key uuid { @key }
				}
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			Expect(resp.Files).To(BeEmpty())
		})
	})
})
