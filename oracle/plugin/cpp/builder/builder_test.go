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
	"github.com/synnaxlabs/oracle/plugin/cpp/builder"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

func TestCppBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin C++ Builder Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = Describe("C++ Builder Plugin", func() {
	var (
		loader *MockFileLoader
		p      *builder.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", "@cpp output \"x/cpp/filter\"\n"+FilterSchema)
		p = builder.New(builder.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("cpp/builder"))
		})

		It("Should filter on cpp domain", func() {
			Expect(p.Domains()).To(Equal([]string{"cpp"}))
		})

		It("Should require cpp/types and cpp/json", func() {
			Expect(p.Requires()).To(Equal([]string{"cpp/types", "cpp/json"}))
		})
	})

	Describe("Generate", func() {
		const thing = `
			import "schemas/x/filter"

			@cpp output "client/cpp/thing"

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

		It("Should alias the condition of the filter", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.h").ToContain(
				`#include "client/cpp/thing/json.gen.h"`,
				`#include "x/cpp/filter/builder.h"`,
				"namespace synnax::thing {",
				"using Condition = x::filter::Condition<Where>;",
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
			ExpectContent(resp, "builder.gen.h").ToContain(
				"/// @brief key selects by key.\n"+
					"inline constexpr x::filter::SetLine<&Where::key> key{};",
				"inline constexpr x::filter::StringLine<&Where::name> name{};",
				"inline constexpr x::filter::BoolLine<&Where::active> active{};",
				"inline constexpr x::filter::OrderedLine<&Where::created_at> created_at{};",
				"inline constexpr x::filter::ContainsLine<&Where::tags> tags{};",
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
			ExpectContent(resp, "builder.gen.h").ToContain(
				"inline constexpr x::filter::FlagLine<&Where::internal> internal{};",
				"inline constexpr x::filter::ListLine<&Where::spans> spans{};",
				"/// @brief span matches things that overlap the span.\n"+
					"inline constexpr x::filter::ValueLine<&Where::span> span{};",
			)
		})

		It("Should escape a C++ keyword", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					for Span
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.h").ToContain(
				"inline constexpr x::filter::ValueLine<&Where::for_> for_{};",
			)
		})

		It("Should export the tree functions", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.h").ToContain(
				"using x::filter::and_;",
				"using x::filter::not_;",
				"using x::filter::or_;",
				"inline Condition where(Where tree) {\n    return {std::move(tree)};\n}",
				"using x::filter::one;",
			)
		})

		It("Should export an option for each request field", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "builder.gen.h").
				ToContain(
					"using x::filter::search;",
					"using x::filter::limit;",
					"using x::filter::offset;",
					"/// @brief include_labels includes the labels of each record.\n"+
						"inline x::filter::Include<&RetrieveRequest::include_labels> "+
						"include_labels() {\n    return {};\n}",
				).
				ToNotContain("using x::filter::where;", "search_term")
		})
	})
})
