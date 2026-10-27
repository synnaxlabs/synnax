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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/go/filter"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Go Filter Plugin", func() {
	var (
		loader *MockFileLoader
		p      *filter.Plugin
	)

	BeforeEach(func() {
		loader = NewMockFileLoader()
		loader.Add("schemas/x/filter", "@go output \"x/go/filter\"\n"+FilterSchema)
		p = filter.New(filter.DefaultOptions())
	})

	Describe("Plugin Interface", func() {
		It("Should have correct name", func() {
			Expect(p.Name()).To(Equal("go/filter"))
		})

		It("Should filter on go domain", func() {
			Expect(p.Domains()).To(Equal([]string{"go"}))
		})

		It("Should require go/types and go/query", func() {
			Expect(p.Requires()).To(Equal([]string{"go/types", "go/query"}))
		})
	})

	Describe("Generate", func() {
		const thing = `
			import "schemas/x/filter"

			@go output "core/pkg/service/thing"

			Kind enum {
				a = "a"
			}
			Span struct {
				start int64
			}
			Thing struct {
				key    uuid     { @key }
				name   string
				active bool
				count  int32
				kind   Kind
				tags   string[]
				owner  uuid     { @filter scalar }
				@retrieve
			}
		`

		It("Should match each bare line through its matcher", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
					name
					active
					kind
					owner
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"func MatchWhere(w Where) (Filter, error) {",
					"fs = append(fs, MatchKeys(*l.Eq))",
					"fs = append(fs, MatchKeys(l.In...))",
					"fs = append(fs, MatchNames(l.In...))",
					"fs = append(fs, MatchActive(*l.Eq))",
					"fs = append(fs, MatchKinds(l.In...))",
					"fs = append(fs, MatchOwner(*l.Eq))",
					"return And(fs...), nil",
				).
				ToPreserveOrder(
					"if l := w.Key; l != nil {",
					"if l := w.Name; l != nil {",
					"if l := w.Active; l != nil {",
					"if l := w.Kind; l != nil {",
					"if l := w.Owner; l != nil {",
				)
		})

		It("Should read a union's shared fields and tag through its methods", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/view"

				Type enum {
					first = "first"
				}
				Base struct {
					key  uuid   { @key }
					name string
				}
				View union on type Type extends Base {
					first { size int32 }
					@retrieve
				}

				Where filter View {
					name
					type
				}
			`, "view", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"return re.MatchString(e.Base().Name), nil",
					"fs = append(fs, MatchTypes(l.In...))",
				)
		})

		It("Should OR one call per value for a one-value matcher", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					owner
					tags
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"in[i] = MatchOwner(v)",
					"fs = append(fs, Or(in...))",
					"anyOf[i] = MatchTag(v)",
					"fs = append(fs, Or(anyOf...))",
				).
				ToNotContain("MatchOwner(l.In...)")
		})

		It("Should walk the tree and prefix child errors with their path", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToContain(
					`and, err := matchWheres(w.And, "and")`,
					"fs = append(fs, And(and...))",
					`or, err := matchWheres(w.Or, "or")`,
					"fs = append(fs, Or(or...))",
					"not, err := MatchWhere(*w.Not)",
					`return nil, validate.PathedError(err, "not")`,
					"fs = append(fs, Not(not))",
					"return nil, validate.PathedError(err, path, strconv.Itoa(i))",
				)
		})

		It("Should compile a regex and reject an invalid one at its path", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					name
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					`"regexp"`,
					"re, err := regexp.Compile(*l.Regex)",
					"errors.Wrap(validate.ErrValidation, err.Error())",
					`"name",`,
					`"regex",`,
					"e *Thing,",
					"return re.MatchString(e.Name), nil",
				)
		})

		It("Should reject an invalid enum value at its path", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					kind
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"if !l.Eq.IsValid() {",
					`errors.Wrapf(validate.ErrValidation, "invalid kind %q", *l.Eq)`,
					`"eq",`,
					"if !v.IsValid() {",
					`errors.Wrapf(validate.ErrValidation, "invalid kind %q", v)`,
					`"in",`,
				)
		})

		It("Should compare ordered members in closures", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					count
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"fs = append(fs, MatchCounts(*l.Eq))",
					"v := *l.Gt",
					"return e.Count > v, nil",
					"return e.Count >= v, nil",
					"return e.Count < v, nil",
					"return e.Count <= v, nil",
				).
				ToNotContain(`"regexp"`)
		})

		It("Should pass a typed line's value to its hand-written function", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					overlaps Span
					labels   uuid[]
				}
			`, "thing", loader, p)
			ExpectContent(resp, "where.gen.go").
				ToBeValidGoSource().
				ToContain(
					"fs = append(fs, MatchOverlaps(*l))",
					"fs = append(fs, MatchLabels(l...))",
				).
				ToNotContain(`"regexp"`, `/gorp"`)
		})

		It("Should decode the retrieve request strictly", func(ctx SpecContext) {
			resp := MustGenerate(ctx, thing+`
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "request.gen.go").
				ToContain(
					"func (r *RetrieveRequest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {",
					"type plain RetrieveRequest",
					"json.UnmarshalDecode(dec, (*plain)(r), json.RejectUnknownMembers(true))",
					`"github.com/vmihailenco/msgpack/v5"`,
					"func (r *RetrieveRequest) DecodeMsgpack(dec *msgpack.Decoder) error {",
					"dec.DisallowUnknownFields(true)",
					"defer dec.DisallowUnknownFields(false)",
					"return dec.Decode((*plain)(r))",
				)
		})

		It(
			"Should make the interpreter generic over the target's type parameters",
			func(
				ctx SpecContext,
			) {
				resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/status"

				Variant enum {
					info = "info"
				}
				Status struct<Details?, V extends Variant = Variant> {
					key     string { @key }
					name    string
					variant V
					details Details
					@retrieve indexes
				}
				Where filter Status {
					key
					name
					variant
				}
			`, "status", loader, p)
				ExpectContent(resp, "where.gen.go").
					ToBeValidGoSource().
					ToContain(
						"func MatchWhere[Details any](w Where) (Filter[Details], error) {",
						"var fs []Filter[Details]",
						"fs = append(fs, MatchKeys[Details](l.In...))",
						"fs = append(fs, MatchVariants[Details](l.In...))",
						"fs = append(fs, Match[Details](func(",
						"_ Retrieve[Details],",
						"e *Status[Details],",
						"func matchWheres[Details any](ws []Where, path string) "+
							"([]Filter[Details], error) {",
						"return And[Details](fs...), nil",
					)
			},
		)

		It("Should apply each member of the request to the Retrieve", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/thing"

				Thing struct {
					key  uuid   { @key }
					name string { @search }
					@retrieve
				}
				Where filter Thing {
					key
				}
			`, "thing", loader, p)
			ExpectContent(resp, "request.gen.go").
				ToBeValidGoSource().
				ToContain(
					"func (r RetrieveRequest) Apply(q Retrieve) (Retrieve, error) {",
					"q = q.SkipMissingKeys()",
					"f, err := MatchWhere(*r.Where)",
					`return Retrieve{}, validate.PathedError(err, "where")`,
					`if r.SearchTerm != "" {`,
					"q = q.Where(MatchSearch(r.SearchTerm))",
					"if *r.Limit == 0 {",
					"return q.Where(MatchKeys()), nil",
					"q = q.Limit(int(*r.Limit))",
					"return q.Offset(int(r.Offset)), nil",
				).
				ToPreserveOrder("r.Where", "r.SearchTerm", "r.Limit", "r.Offset")
		})

		It("Should generate the request methods at the request output", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/thing"

				Thing struct {
					key  uuid   { @key }
					name string { @search }
					@retrieve
				}
				Where filter Thing {
					key

					@go request_output "core/pkg/api/thing"
				}
			`, "thing", loader, p)
			Expect(resp.Files).To(HaveLen(2))
			Expect(resp.Files[1].Path).To(Equal("core/pkg/api/thing/request.gen.go"))
			ExpectContent(resp, "request.gen.go").
				ToBeValidGoSource().
				ToContain(
					"package thing",
					`servicething "github.com/synnaxlabs/synnax/core/pkg/service/thing"`,
					"func (r RetrieveRequest) Apply(q servicething.Retrieve) "+
						"(servicething.Retrieve, error) {",
					"f, err := servicething.MatchWhere(*r.Where)",
					"return servicething.Retrieve{}, validate.PathedError(err, \"where\")",
					"return q.Where(servicething.MatchKeys()), nil",
				)
		})

		It("Should apply a request over a generic target to every type argument", func(
			ctx SpecContext,
		) {
			resp := MustGenerate(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/status"

				Status struct<Details?> {
					key     string { @key }
					name    string { @search }
					details Details
					@retrieve indexes
				}
				Where filter Status {
					key
				}
			`, "status", loader, p)
			ExpectContent(resp, "request.gen.go").
				ToBeValidGoSource().
				ToContain(
					"func (r RetrieveRequest) Apply(q Retrieve[any]) (Retrieve[any], error) {",
					"f, err := MatchWhere[any](*r.Where)",
					"q = q.Where(MatchSearch(r.SearchTerm))",
					"return q.Where(MatchKeys[any]()), nil",
				)
		})

		It("Should reject a target with no @retrieve", func(ctx SpecContext) {
			req := MustGenerateRequest(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/thing"

				Thing struct {
					key uuid { @key }
				}
				Where filter Thing {
					key
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(
				"filter thing.Where: the target thing.Thing has no @retrieve, so it has " +
					"no Retrieve or Filter to build on",
			))
		})

		DescribeTable("Should reject an operator member it cannot interpret", func(
			ctx SpecContext,
			from, to, line, message string,
		) {
			loader.Add(
				"schemas/x/filter",
				`@go output "x/go/filter"`+strings.Replace(FilterSchema, from, to, 1),
			)
			req := MustGenerateRequest(ctx, thing+`
				Where filter Thing {
					`+line+`
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(
				"filter thing.Where: line " + line + ": go/filter cannot interpret " +
					message,
			))
		},
			Entry(
				"a member no operator has",
				"regex string?",
				"contains string?",
				"name",
				"the member contains of String",
			),
			Entry(
				"a member of another operator",
				"in T[]?\n\t@pb omit\n}\nOrdered",
				"in T[]?\n\tgt T?\n\t@pb omit\n}\nOrdered",
				"key",
				"the member gt of Set",
			),
		)

		It("Should skip a filter with no Go output", func(ctx SpecContext) {
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

		DescribeTable("Should reject a typed line that calls a generated matcher", func(
			ctx SpecContext,
			line, function string,
		) {
			req := MustGenerateRequest(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/thing"

				Thing struct {
					key    uuid   { @key }
					name   string { @search }
					active bool   { @filter }
					@retrieve
				}
				Where filter Thing {
					`+line+` bool
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"filter thing.Where: typed line " + line + " calls " + function +
					", which the generated code already declares; rename the line",
			)))
		},
			Entry("for a field matcher", "active", "MatchActive"),
			Entry("for the search filter", "search", "MatchSearch"),
		)

		It("Should reject a constrained type parameter", func(ctx SpecContext) {
			req := MustGenerateRequest(ctx, `
				import "schemas/x/filter"

				@go output "core/pkg/service/thing"

				Thing struct<T extends string> {
					key   uuid { @key }
					value T
					@retrieve
				}
				Where filter Thing {
					key
				}
			`, "thing", loader)
			Expect(p.Generate(req)).Error().To(MatchError(ContainSubstring(
				"filter thing.Where: the type parameter T of Thing has a constraint",
			)))
		})
	})
})
