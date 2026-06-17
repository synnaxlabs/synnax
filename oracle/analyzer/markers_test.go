// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package analyzer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
)

var _ = Describe("Create and Output markers", func() {
	var loader *MockFileLoader
	BeforeEach(func() { loader = NewMockFileLoader() })

	It("Should accept @create on a type and @output on a field", func(ctx SpecContext) {
		source := `
			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
			}
		`
		table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())

		typ := table.MustGet("x.Thing")
		Expect(typ.Domains).To(HaveKey("create"))

		form := typ.Form.(resolution.StructForm)
		author, ok := form.Field("author")
		Expect(ok).To(BeTrue())
		Expect(author.Domains).To(HaveKey("output"))

		name, ok := form.Field("name")
		Expect(ok).To(BeTrue())
		Expect(name.Domains).NotTo(HaveKey("output"))
	})

	It("Should synthesize a derived New type for @create, omitting @output fields", func(ctx SpecContext) {
		source := `
			Thing struct {
				key    uuid @key
				name   string
				author uuid @output
				@create
			}
		`
		table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())

		newType, ok := table.Get("x.New")
		Expect(ok).To(BeTrue())

		form := newType.Form.(resolution.StructForm)
		Expect(form.Extends).To(HaveLen(1))
		Expect(form.Extends[0].Name).To(Equal("x.Thing"))
		resolved, resolvedOK := form.Extends[0].Resolve(table)
		Expect(resolvedOK).To(BeTrue())
		Expect(resolved.Name).To(Equal("Thing"))
		Expect(form.OmittedFields).To(ContainElement("author"))
		Expect(newType.Domains).To(HaveKey("ts"))
		Expect(newType.Domains).To(HaveKey("go"))
	})

	It("Should not synthesize a New when one is hand-written", func(ctx SpecContext) {
		source := `
			Thing struct {
				key  uuid @key
				name string
				@create
			}
			New struct extends Thing {
				key?
				@ts use_input
			}
		`
		// A duplicate synthesized New would add a diagnostic, so a clean analysis
		// proves the hand-written New was respected.
		table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())
		_, ok := table.Get("x.New")
		Expect(ok).To(BeTrue())
	})

	It("Should synthesize a single New when a namespace has two @create structs", func(ctx SpecContext) {
		source := `
			First struct {
				key  uuid @key
				name string
				@create
			}
			Second struct {
				key  uuid @key
				size int32
				@create
			}
		`
		// Both structs would target x.New; only one may be synthesized, otherwise
		// the second table.Add collides and fails analysis.
		table, diag := analyzer.AnalyzeSource(ctx, source, "x", loader)
		Expect(diag.Ok()).To(BeTrue())
		newType, ok := table.Get("x.New")
		Expect(ok).To(BeTrue())
		form := newType.Form.(resolution.StructForm)
		Expect(form.Extends).To(HaveLen(1))
		Expect(form.Extends[0].Name).To(BeElementOf("x.First", "x.Second"))
	})
})
