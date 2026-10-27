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
	"github.com/synnaxlabs/oracle/plugin/py/builder"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPyBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Py Builder Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = Describe("Py Builder Plugin", func() {
	var (
		loader *MockFileLoader
		p      *builder.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", "@py output \"x/py/x/filter\"\n"+FilterSchema)
		loader.Add("schemas/rack", `
			@py output "client/py/synnax/rack"

			Key = uint32
		`)
		p = builder.New(builder.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("py/builder"))
		})

		It("Should filter on py domain", func() {
			Expect(p.Domains()).To(Equal([]string{"py"}))
		})

		It("Should require py/types", func() {
			Expect(p.Requires()).To(Equal([]string{"py/types"}))
		})
	})

	Describe("Generate", func() {
		const thing = `
			import "schemas/x/filter"
			import "schemas/rack"

			@py output "client/py/synnax/thing"

			Span struct {
				start int64
			}
			Thing struct {
				key        uuid     { @key }
				name       string
				active     bool
				created_at int64
				tags       string[]
				rack       rack.Key
			}
		`

		It("Should declare the condition class of the filter", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder_gen.py").ToContain(
				"from x import filter\n",
				"from synnax.thing.types_gen import Where\n",
				"class Condition(filter.Condition):",
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
			ExpectContent(resp, "builder_gen.py").ToContain(
				"from uuid import UUID\n",
				`key: filter.SetLine[Condition, UUID] = filter.SetLine(Condition, "key")`+
					"\n\"\"\"Selects by key.\"\"\"",
				`name: filter.StringLine[Condition] = filter.StringLine(Condition, "name")`,
				`active: filter.BoolLine[Condition] = filter.BoolLine(Condition, "active")`,
				`created_at: filter.OrderedLine[Condition, int] = filter.OrderedLine(`+
					`Condition, "created_at")`,
				`tags: filter.ContainsLine[Condition, str] = filter.ContainsLine(`+
					`Condition, "tags")`,
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
			ExpectContent(resp, "builder_gen.py").ToContain(
				"from synnax.thing.types_gen import Where, Span\n",
				`internal: filter.FlagLine[Condition] = filter.FlagLine(`+
					`Condition, "internal")`,
				`spans: filter.ListLine[Condition, Span] = filter.ListLine(`+
					`Condition, "spans")`,
				`span: filter.ValueLine[Condition, Span] = filter.ValueLine(`+
					`Condition, "span")`+"\n\"\"\"Matches things that overlap the span.\"\"\"",
			)
		})

		It("Should escape a Python keyword", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					for Span
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder_gen.py").ToContain(
				`for_: filter.ValueLine[Condition, Span] = filter.ValueLine(` +
					`Condition, "for_")`,
			)
		})

		It("Should alias a module whose name a line takes", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					rack
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder_gen.py").ToContain(
				"from synnax import rack as rack_\n",
				`rack: filter.OrderedLine[Condition, rack_.Key] = filter.OrderedLine(`+
					`Condition, "rack")`,
			)
		})

		It("Should bind the tree functions", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder_gen.py").ToContain(
				"and_ = Condition.and_\n",
				"or_ = Condition.or_\n",
				"not_ = Condition.not_\n",
				"def where(tree: Where) -> Condition:\n    \"\"\"Turns a stored tree "+
					"into a condition.\"\"\"\n    return Condition(tree)",
			)
		})
	})
})
