// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	. "github.com/synnaxlabs/oracle/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

// analyze resolves source into a table and returns the named type's struct form.
func analyze(
	ctx context.Context,
	source, name string,
) (resolution.StructForm, *resolution.Table) {
	GinkgoHelper()
	table := MustGenerateRequest(ctx, source, "test", NewMockFileLoader()).Resolutions
	typ := MustBeOk(table.Get("test." + name))
	form, ok := typ.Form.(resolution.StructForm)
	Expect(ok).To(BeTrue(), "%s is not a struct", name)
	return form, table
}

// analyzeUnion resolves source into a table and returns the named union's form and
// one of its variants.
func analyzeUnion(
	ctx context.Context,
	source, union, variant string,
) (resolution.UnionForm, resolution.UnionVariant, *resolution.Table) {
	GinkgoHelper()
	table := MustGenerateRequest(ctx, source, "test", NewMockFileLoader()).Resolutions
	typ := MustBeOk(table.Get("test." + union))
	form, ok := typ.Form.(resolution.UnionForm)
	Expect(ok).To(BeTrue(), "%s is not a union", union)
	v, ok := form.Variant(variant)
	Expect(ok).To(BeTrue(), "%s has no variant %s", union, variant)
	return form, v, table
}

// fieldNames lists the names of fields, for asserting on what a variant declares.
func fieldNames(fields []resolution.Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	return names
}

var _ = Describe("Inheritance", func() {
	Describe("HasDomainOmissions", func() {
		It("Should report a field that drops an inherited domain", func() {
			form := resolution.StructForm{Fields: []resolution.Field{
				{Name: "key", OmittedDomains: []string{"validate"}},
			}}
			Expect(resolver.HasDomainOmissions(form)).To(BeTrue())
		})

		It("Should report false when no field omits a domain", func() {
			form := resolution.StructForm{Fields: []resolution.Field{{Name: "key"}}}
			Expect(resolver.HasDomainOmissions(form)).To(BeFalse())
		})
	})

	Describe("DefaultOnlyOverrides", func() {
		It("Should report a field restated with the same type", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					port string = "AIN0"
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(HaveKey("port"))
		})

		It("Should report a typeless override", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					sample_rate float64 = 10
				}

				Child struct extends Base {
					sample_rate = 50
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(HaveKey("sample_rate"))
		})

		It("Should not report a field the parents do not declare", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(BeEmpty())
		})

		It("Should not report an override that changes the type", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					port int32 = 4
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(BeEmpty())
		})

		It("Should not report an override that changes the element type", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Wide struct {
					name string = ""
				}

				Narrow struct extends Wide {
					extra int32 = 0
				}

				Base struct {
					items Wide[]
				}

				Child struct extends Base {
					items Narrow[]
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(BeEmpty())
		})

		It("Should not report an override that changes optionality", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					port string?
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(BeEmpty())
		})

		It("Should report overrides against the leftmost parent", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				First struct {
					port string = "first"
				}

				Second struct {
					rate float64 = 1
				}

				Child struct extends First, Second {
					port string = "AIN0"
					rate float64 = 50
				}
			`, "Child")
			Expect(resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)).
				To(SatisfyAll(HaveKey("port"), HaveKey("rate")))
		})
	})

	Describe("CanUseInheritance", func() {
		It("Should be false without parents", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Solo struct {
					port string = ""
				}
			`, "Solo")
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeFalse())
		})

		It("Should be true for a plain extension", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeTrue())
		})

		It("Should stay true when a field restates only a default", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					port string = "AIN0"
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeTrue())
		})

		It("Should be false when a field restates an inherited type", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					port int32 = 4
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeFalse())
		})

		It("Should be true when the parents are disjoint", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				First struct {
					port string = "first"
				}

				Second struct {
					rate float64 = 1
				}

				Child struct extends First, Second {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeTrue())
		})

		It("Should be false when a parent is omitted in the language", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"
				@ts output "ts/test"

				Base struct {
					port string = ""
					@ts omit
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, "ts", table)).To(BeFalse())
			Expect(resolver.CanUseInheritance(form, "go", table)).To(BeTrue())
		})
	})

	Describe("ParentsExist", func() {
		It("Should be true when every parent has an output", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@ts output "ts/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.ParentsExist(form.Extends, "ts", table)).To(BeTrue())
		})

		It("Should be true for a hand-written parent", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@ts output "ts/test"

				Base struct {
					port string = ""
					@ts hand
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.ParentsExist(form.Extends, "ts", table)).To(BeTrue())
		})

		It("Should be false when a parent omits the language", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@ts output "ts/test"

				Base struct {
					port string = ""
					@ts omit
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.ParentsExist(form.Extends, "ts", table)).To(BeFalse())
		})

		It("Should be false when a parent has no output in the language", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				Base struct {
					port string = ""
					@go output "pkg/test"
				}

				Child struct extends Base {
					range float64 = 10
					@ts output "ts/test"
				}
			`, "Child")
			Expect(resolver.ParentsExist(form.Extends, "ts", table)).To(BeFalse())
		})
	})

	Describe("VariantBases", func() {
		It("Should inherit every base when the variant drops nothing", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				Extra struct {
					scale float64 = 1
				}

				Chan union on type extends Base, Extra {
					builtin {
						units string = "c"
					}
				}
			`, "Chan", "builtin")
			inherited, declared := resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(HaveLen(2))
			Expect(declared).To(BeEmpty())
		})

		It("Should declare the fields of a base absent from the language", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"
				@cpp output "cpp/test"

				Base struct {
					port string = ""
					@cpp omit
				}

				Chan union on type {
					builtin extends Base {
						units string = "c"
					}
				}
			`, "Chan", "builtin")
			inherited, declared := resolver.VariantBases(form, v, "cpp", table)
			Expect(inherited).To(BeEmpty())
			Expect(fieldNames(declared)).To(Equal([]string{"port"}))
			inherited, declared = resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(HaveLen(1))
			Expect(declared).To(BeEmpty())
		})

		It("Should declare the other fields of a base it drops a field from", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				Extra struct {
					scale float64 = 1
				}

				Chan union on type extends Base, Extra {
					builtin {
						-port
						units string = "c"
					}
				}
			`, "Chan", "builtin")
			inherited, declared := resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(HaveLen(1))
			Expect(inherited[0].Name).To(Equal("test.Extra"))
			Expect(fieldNames(declared)).To(Equal([]string{"name"}))
		})

		It(
			"Should declare the other fields of an aliased base it drops a field from",
			func(
				ctx SpecContext,
			) {
				form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				AliasBase = Base

				Chan union on type extends AliasBase {
					builtin {
						-port
						units string = "c"
					}
				}
			`, "Chan", "builtin")
				inherited, declared := resolver.VariantBases(form, v, "go", table)
				Expect(inherited).To(BeEmpty())
				Expect(fieldNames(declared)).To(Equal([]string{"name"}))
			},
		)

		It("Should not declare a field an inherited base already supplies", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				Extra struct {
					name  string = "extra"
					scale float64 = 1
				}

				Chan union on type extends Base {
					builtin extends Extra {
						-port
						units string = "c"
					}
				}
			`, "Chan", "builtin")
			inherited, declared := resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(HaveLen(1))
			Expect(fieldNames(declared)).ToNot(ContainElement("name"))
		})

		It("Should apply a flattened generic base's type arguments", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Record struct<T> {
					value T
					@go omit
				}

				Chan union on type {
					ai extends Record<string> {}
				}
			`, "Chan", "ai")
			inherited, declared := resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(BeEmpty())
			Expect(declared).To(HaveLen(1))
			Expect(declared[0].Type.Name).To(Equal("string"))
		})

		It("Should not declare a field the variant declares itself", func(
			ctx SpecContext,
		) {
			form, v, table := analyzeUnion(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				Chan union on type extends Base {
					builtin {
						-port
						name string = "builtin"
					}
				}
			`, "Chan", "builtin")
			inherited, declared := resolver.VariantBases(form, v, "go", table)
			Expect(inherited).To(BeEmpty())
			Expect(fieldNames(declared)).ToNot(ContainElement("name"))
		})
	})
})
