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

func newGoType(
	name string,
	form resolution.TypeForm,
	outputPath string,
) resolution.Type {
	return resolution.Type{
		Name:          name,
		QualifiedName: "test." + name,
		Namespace:     "test",
		Form:          form,
		Domains: map[string]resolution.Domain{
			"go": {
				Expressions: []resolution.Expression{
					{
						Name:   "output",
						Values: []resolution.ExpressionValue{{StringValue: outputPath}},
					},
				},
			},
		},
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
})
