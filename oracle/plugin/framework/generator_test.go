// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framework_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

type captureFileGenerator struct {
	contexts []*framework.GenerateContext
}

func (c *captureFileGenerator) GenerateFile(
	ctx *framework.GenerateContext,
) (string, error) {
	c.contexts = append(c.contexts, ctx)
	return "// generated", nil
}

type errFileGenerator struct{}

func (errFileGenerator) GenerateFile(*framework.GenerateContext) (string, error) {
	return "", errors.New("boom")
}

type emptyFileGenerator struct{}

func (emptyFileGenerator) GenerateFile(*framework.GenerateContext) (string, error) {
	return "", nil
}

func newGoType(
	name string,
	form resolution.TypeForm,
	outputPath string,
	extraExprs ...resolution.Expression,
) resolution.Type {
	exprs := []resolution.Expression{
		{
			Name:   "output",
			Values: []resolution.ExpressionValue{{StringValue: outputPath}},
		},
	}
	exprs = append(exprs, extraExprs...)
	return resolution.Type{
		Name:          name,
		QualifiedName: "test." + name,
		Namespace:     "test",
		Form:          form,
		Domains:       map[string]resolution.Domain{"go": {Expressions: exprs}},
	}
}

var _ = Describe("Generator", func() {
	var (
		table *resolution.Table
		fg    *captureFileGenerator
		gen   *framework.Generator
	)

	BeforeEach(func() {
		table = resolution.NewTable()
		fg = &captureFileGenerator{}
		gen = &framework.Generator{
			FileGenerator: fg,
			Domain:        "go",
			FilePattern:   "types.gen.go",
			CollectUnions: true,
		}
		Expect(table.Add(newGoType("Payload", resolution.StructForm{}, "pkg/types"))).
			To(Succeed())
	})

	Describe("Union Collection", func() {
		It(
			"Should pass unions sharing the structs' output path to the file generator",
			func() {
				Expect(table.Add(newGoType(
					"Scale",
					resolution.UnionForm{Discriminator: "type"},
					"pkg/types",
				))).To(Succeed())
				resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
				Expect(resp.Files).To(HaveLen(1))
				Expect(fg.contexts).To(HaveLen(1))
				Expect(fg.contexts[0].Unions).To(HaveLen(1))
				Expect(fg.contexts[0].Unions[0].Name).To(Equal("Scale"))
			},
		)

		It("Should not collect unions when CollectUnions is disabled", func() {
			gen.CollectUnions = false
			Expect(table.Add(newGoType(
				"Scale",
				resolution.UnionForm{Discriminator: "type"},
				"pkg/types",
			))).To(Succeed())
			MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(fg.contexts).To(HaveLen(1))
			Expect(fg.contexts[0].Unions).To(BeEmpty())
		})

		It(
			"Should generate a union-only file for unions at an output with no structs",
			func() {
				Expect(table.Add(newGoType(
					"Scale",
					resolution.UnionForm{Discriminator: "type"},
					"other/types",
				))).To(Succeed())
				resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
				Expect(resp.Files).To(HaveLen(2))
				Expect(fg.contexts).To(HaveLen(2))

				structCtx := fg.contexts[0]
				Expect(structCtx.OutputPath).To(Equal("pkg/types"))
				Expect(structCtx.Unions).To(BeEmpty())

				unionCtx := fg.contexts[1]
				Expect(unionCtx.OutputPath).To(Equal("other/types"))
				Expect(unionCtx.Structs).To(BeEmpty())
				Expect(unionCtx.Unions).To(HaveLen(1))
				Expect(unionCtx.Unions[0].Name).To(Equal("Scale"))
			},
		)
	})

	enumForm := func() resolution.EnumForm {
		return resolution.EnumForm{
			Values: []resolution.EnumValue{{Name: "fast", Value: "Fast"}},
		}
	}

	Describe("Union Enum Merging", func() {
		BeforeEach(func() {
			gen.CollectEnums = true
			Expect(table.Add(newGoType(
				"Scale",
				resolution.UnionForm{Discriminator: "type"},
				"other/types",
			))).To(Succeed())
			Expect(table.Add(newGoType("Mode", enumForm(), "other/types"))).
				To(Succeed())
		})

		It("Should merge enums and typedefs into a union-only file", func() {
			gen.CollectTypeDefs = true
			Expect(table.Add(newGoType(
				"Key", resolution.DistinctForm{}, "other/types",
			))).To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(2))
			unionCtx := fg.contexts[1]
			Expect(unionCtx.OutputPath).To(Equal("other/types"))
			Expect(unionCtx.Unions).To(HaveLen(1))
			Expect(unionCtx.Enums).To(HaveLen(1))
			Expect(unionCtx.TypeDefs).To(HaveLen(1))
		})

		It("Should dedupe union-file enums by name when MergeByName is set", func() {
			gen.MergeByName = true
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(2))
			Expect(fg.contexts[1].Enums).To(HaveLen(1))
		})
	})

	Describe("Enum Collection", func() {
		BeforeEach(func() {
			gen.CollectEnums = true
		})

		It("Should merge same-path standalone enums into the struct file", func() {
			Expect(table.Add(newGoType("Mode", enumForm(), "pkg/types"))).
				To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(1))
			Expect(fg.contexts[0].Enums).To(HaveLen(1))
			Expect(fg.contexts[0].Enums[0].Name).To(Equal("Mode"))
		})

		It("Should dedupe merged enums by name when MergeByName is set", func() {
			gen.MergeByName = true
			Expect(table.Add(newGoType("Mode", enumForm(), "pkg/types"))).
				To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(1))
			Expect(fg.contexts[0].Enums).To(HaveLen(1))
		})

		It("Should generate an enum-only file for enums at their own output", func() {
			Expect(table.Add(newGoType("Mode", enumForm(), "enums/only"))).
				To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(2))
			enumCtx := fg.contexts[1]
			Expect(enumCtx.OutputPath).To(Equal("enums/only"))
			Expect(enumCtx.Structs).To(BeEmpty())
			Expect(enumCtx.Enums).To(HaveLen(1))
		})

		It("Should merge extra enums from ExtraEnumsFunc", func() {
			extra := newGoType("Extra", enumForm(), "elsewhere/enums")
			gen.ExtraEnumsFunc = func(
				[]resolution.Type, *resolution.Table, string,
			) []resolution.Type {
				return []resolution.Type{extra}
			}
			MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(fg.contexts[0].Enums).To(HaveLen(1))
			Expect(fg.contexts[0].Enums[0].Name).To(Equal("Extra"))
		})

		It("Should apply FilterEnums to the merged enum set", func() {
			Expect(table.Add(newGoType("Mode", enumForm(), "pkg/types"))).
				To(Succeed())
			gen.FilterEnums = func(
				[]resolution.Type, string,
			) []resolution.Type {
				return nil
			}
			MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(fg.contexts[0].Enums).To(BeEmpty())
		})
	})

	Describe("TypeDef Collection", func() {
		BeforeEach(func() {
			gen.CollectTypeDefs = true
		})

		It("Should merge same-path distinct and alias types into the file", func() {
			Expect(table.Add(newGoType(
				"Key", resolution.DistinctForm{}, "pkg/types",
			))).To(Succeed())
			Expect(table.Add(newGoType(
				"Name", resolution.AliasForm{}, "pkg/types",
			))).To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(1))
			Expect(fg.contexts[0].TypeDefs).To(HaveLen(2))
		})

		It("Should generate a typedef-only file at an output with no structs", func() {
			Expect(table.Add(newGoType(
				"Key", resolution.DistinctForm{}, "ids/only",
			))).To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(2))
			defCtx := fg.contexts[1]
			Expect(defCtx.OutputPath).To(Equal("ids/only"))
			Expect(defCtx.Structs).To(BeEmpty())
			Expect(defCtx.TypeDefs).To(HaveLen(1))
		})
	})

	Describe("PathFilter", func() {
		It("Should skip filtered-out output paths in every collection pass", func() {
			gen.CollectEnums = true
			gen.CollectTypeDefs = true
			gen.PathFilter = func(outputPath string) bool {
				return outputPath == "pkg/types"
			}
			Expect(table.Add(newGoType(
				"Scale",
				resolution.UnionForm{Discriminator: "type"},
				"other/unions",
			))).To(Succeed())
			Expect(table.Add(newGoType("Mode", enumForm(), "other/enums"))).
				To(Succeed())
			Expect(table.Add(newGoType(
				"Key", resolution.DistinctForm{}, "other/ids",
			))).To(Succeed())
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(HaveLen(1))
			Expect(resp.Files[0].Path).To(Equal("pkg/types/types.gen.go"))
		})
	})

	Describe("Hand and Omitted Types", func() {
		It("Should skip hand types by default and collect them with IncludeHand",
			func() {
				hand := newGoType(
					"Hand", resolution.StructForm{}, "pkg/types",
					resolution.Expression{Name: "hand"},
				)
				Expect(table.Add(hand)).To(Succeed())
				MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
				Expect(fg.contexts[0].Structs).To(HaveLen(1))

				fg.contexts = nil
				gen.IncludeHand = true
				MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
				Expect(fg.contexts[0].Structs).To(HaveLen(2))
			})

		It("Should skip omitted types even with IncludeHand", func() {
			gen.IncludeHand = true
			omitted := newGoType(
				"Gone", resolution.StructForm{}, "pkg/types",
				resolution.Expression{Name: "omit"},
			)
			Expect(table.Add(omitted)).To(Succeed())
			MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(fg.contexts[0].Structs).To(HaveLen(1))
		})
	})

	Describe("Failure paths", func() {
		It("Should wrap file generator errors with the output path", func() {
			gen.FileGenerator = errFileGenerator{}
			Expect(gen.Generate(&plugin.Request{Resolutions: table})).Error().
				To(MatchError(ContainSubstring("failed to generate pkg/types")))
		})

		It("Should emit no file when the generator returns empty content", func() {
			gen.FileGenerator = emptyFileGenerator{}
			resp := MustSucceed(gen.Generate(&plugin.Request{Resolutions: table}))
			Expect(resp.Files).To(BeEmpty())
		})

		It("Should propagate invalid struct output paths", func() {
			Expect(table.Add(newGoType(
				"Bad", resolution.StructForm{}, "../escape",
			))).To(Succeed())
			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    GinkgoT().TempDir(),
			}
			Expect(gen.Generate(req)).Error().
				To(MatchError(ContainSubstring("invalid output path for Bad")))
		})

		It("Should propagate invalid typedef output paths", func() {
			gen.CollectTypeDefs = true
			Expect(table.Add(newGoType(
				"BadKey", resolution.DistinctForm{}, "../escape",
			))).To(Succeed())
			req := &plugin.Request{
				Resolutions: table,
				RepoRoot:    GinkgoT().TempDir(),
			}
			Expect(gen.Generate(req)).Error().
				To(MatchError(ContainSubstring("invalid output path for BadKey")))
		})
	})
})
