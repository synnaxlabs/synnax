// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package builder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/builder"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestTSBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin TS Builder Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = Describe("TS Builder Plugin", func() {
	var (
		loader *MockFileLoader
		p      *builder.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", "@ts output \"x/ts/src/filter\"\n"+FilterSchema)
		p = builder.New(builder.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("ts/builder"))
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
			Label struct {
				key  uuid { @key }
				name string
			}
			Thing struct {
				key        uuid     { @key }
				name       string   { @search }
				active     bool
				created_at int64
				tags       string[]
				labels     Label[]? { @include }
			}
		`

		It("Should tag conditions with the filter", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.ts").ToContain(
				`import { filter } from "@synnaxlabs/x";`,
				`import { Where } from "@/thing/types.gen";`,
				`const builder = filter.builder<"thing.Where", Where>();`,
				`export type Condition = filter.Condition<"thing.Where", Where>;`,
			)
		})

		It("Should build each bare line with its operator", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
					name
					active
					created_at
					tags
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.ts").ToContain(
				"/** key selects by key. */\nexport const key = builder.set(\"key\");",
				`export const name = builder.string("name");`,
				`export const active = builder.bool("active");`,
				`export const createdAt = builder.ordered("createdAt");`,
				`export const tags = builder.contains("tags");`,
			)
		})

		It("Should build each typed line by its type", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					internal bool
					spans    Span[]
					span     Span {
						@doc value "matches things that overlap the span."
					}
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.ts").ToContain(
				`export const internal = builder.flag("internal");`,
				`export const spans = builder.list("spans");`,
				"/** span matches things that overlap the span. */\n"+
					`export const span = builder.value("span");`,
			)
		})

		DescribeTable("Should export a reserved word under an alias",
			func(ctx SpecContext, name string) {
				resp := MustGenerate(ctx, thing+`
					Where filter Thing {
						`+name+` Span
					}
				`, "thing", loader, p)
				ExpectContent(resp, "builder.gen.ts").
					ToContain(
						`const `+name+`_ = builder.value("`+name+`");`,
						"export { "+name+"_ as "+name+" };",
					).
					ToNotContain("export const " + name + " ")
			},
			Entry("for", "for"),
			Entry("eval", "eval"),
			Entry("arguments", "arguments"),
		)

		It("Should export the tree functions", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.ts").ToContain(
				"export const and = builder.and;",
				"export const or = builder.or;",
				"export const not = builder.not;",
				"export const where = builder.where;",
				"export const one = filter.one;",
			)
		})

		It("Should export an option for each request field", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.ts").
				ToContain(
					"/** search keeps the things that match the term, closest match "+
						"first. */\nexport const search = filter.search;",
					"/** limit is the most things to return. */\n"+
						"export const limit = filter.limit;",
					"export const offset = filter.offset;",
					"/** includeLabels includes the labels of each record. */\n"+
						`export const includeLabels = filter.include("includeLabels");`,
				).
				ToNotContain("export const where = filter", "searchTerm =")
		})
	})
})
