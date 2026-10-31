// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package matcher_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin/go/internal/matcher"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Matcher", func() {
	var loader *MockFileLoader
	BeforeEach(func() {
		loader = NewMockFileLoader()
		AddFilterSchema(loader)
	})

	table := func(ctx SpecContext, source string) *resolution.Table {
		GinkgoHelper()
		req := MustGenerateRequest(ctx, `
			import "schemas/x/filter"

			@go output "core/pkg/service/thing"
		`+source, "thing", loader)
		return req.Resolutions
	}

	type summary struct {
		Func, Access string
		Kind         matcher.Kind
	}
	summarize := func(ms []matcher.Matcher) []summary {
		return lo.Map(ms, func(m matcher.Matcher, _ int) summary {
			return summary{Func: m.Func, Access: m.Access, Kind: m.Kind}
		})
	}

	Describe("List", func() {
		It("Should list the @filter fields and the bare lines in field order", func(
			ctx SpecContext,
		) {
			t := table(ctx, `
				Thing struct {
					key      uuid     { @key }
					internal bool     { @filter }
					name     string
					owner    uuid     { @filter scalar }
					tags     string[]
					color    string
				}
				Where filter Thing {
					key
					tags
					name
				}
			`)
			ms := MustSucceed(matcher.List(t.MustGet("thing.Thing"), t))
			Expect(summarize(ms)).To(Equal([]summary{
				{Func: "MatchInternal", Access: "Internal", Kind: matcher.Single},
				{Func: "MatchNames", Access: "Name", Kind: matcher.Variadic},
				{Func: "MatchOwner", Access: "Owner", Kind: matcher.Single},
				{Func: "MatchTag", Access: "Tags", Kind: matcher.Element},
			}))
		})

		It("Should read a union's base fields and tag through its methods", func(
			ctx SpecContext,
		) {
			t := table(ctx, `
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
			`)
			ms := MustSucceed(matcher.List(t.MustGet("thing.View"), t))
			Expect(summarize(ms)).To(Equal([]summary{
				{Func: "MatchNames", Access: "Base().Name", Kind: matcher.Variadic},
				{Func: "MatchTypes", Access: "Type()", Kind: matcher.Variadic},
			}))
		})

		It("Should reject a line over a field that a variant drops", func(
			ctx SpecContext,
		) {
			t := table(ctx, `
				Type enum {
					first = "first"
				}
				Base struct {
					key  uuid   { @key }
					name string
					size int32
				}
				View union on type Type extends Base {
					first { -size }
				}
				Where filter View {
					name
				}
			`)
			Expect(matcher.List(t.MustGet("thing.View"), t)).Error().To(MatchError(
				"thing.View: a variant drops field name, so no matcher can read it",
			))
		})

		It("Should reject @filter on the @key field", func(ctx SpecContext) {
			t := table(ctx, `
				Thing struct {
					key uuid { @key @filter }
				}
				Where filter Thing {
					key
				}
			`)
			Expect(matcher.List(t.MustGet("thing.Thing"), t)).Error().To(MatchError(
				"thing.Thing: @filter on the @key field key duplicates MatchKeys",
			))
		})
	})

	Describe("Access", func() {
		const source = `
			Type enum {
				first = "first"
			}
			Base struct {
				key  uuid   { @key }
				name string
				size int32
			}
			Thing struct {
				key  uuid { @key }
				name string
			}
			View union on type Type extends Base {
				first { color string }
			}
			Dropped union on type Type extends Base {
				first { -size }
			}
			Where filter Thing {
				key
			}
		`
		field := func(t *resolution.Table, typ, name string) resolution.Field {
			GinkgoHelper()
			fields := MustBeOk(resolution.RecordFields(t.MustGet(typ), t))
			return MustBeOk(lo.Find(fields, func(f resolution.Field) bool {
				return f.Name == name
			}))
		}

		DescribeTable(
			"Should return the selector that reads a field",
			func(ctx SpecContext, typ, name, access string) {
				t := table(ctx, source)
				Expect(MustBeOk(
					matcher.Access(t.MustGet(typ), field(t, typ, name), t),
				)).To(Equal(access))
			},
			Entry("a struct field", "thing.Thing", "name", "Name"),
			Entry("the tag of a union", "thing.View", "type", "Type()"),
			Entry(
				"a field of a union's shared base",
				"thing.View",
				"name",
				"Base().Name",
			),
		)

		It("Should return false for a field that a variant drops", func(
			ctx SpecContext,
		) {
			t := table(ctx, source)
			_, ok := matcher.Access(
				t.MustGet("thing.Dropped"),
				field(t, "thing.Dropped", "name"),
				t,
			)
			Expect(ok).To(BeFalse())
		})
	})

	DescribeTable("Kind.Scalar",
		func(k matcher.Kind, scalar bool) { Expect(k.Scalar()).To(Equal(scalar)) },
		Entry("a variadic matcher", matcher.Variadic, false),
		Entry("a single-value matcher", matcher.Single, true),
		Entry("an element matcher", matcher.Element, true),
	)

	DescribeTable("Plural",
		func(name, plural string) { Expect(matcher.Plural(name)).To(Equal(plural)) },
		Entry("a regular noun", "Name", "Names"),
		Entry("a noun ending in y", "Property", "Properties"),
		Entry("a noun that is already plural", "Properties", "PropertiesList"),
	)

	DescribeTable("Singular",
		func(name, singular string) {
			Expect(matcher.Singular(name)).To(Equal(singular))
		},
		Entry("a regular plural", "Tags", "Tag"),
		Entry("a plural ending in ies", "Properties", "Property"),
		Entry("a plural ending in ches", "Matches", "Match"),
		Entry("a plural ending in xes", "Boxes", "Box"),
		Entry("a name ending in ss", "Class", "Class"),
		Entry("a name with no plural ending", "Data", "Data"),
	)
})
