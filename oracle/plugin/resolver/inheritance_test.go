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

var _ = Describe("Inheritance", func() {
	Describe("InheritedFields", func() {
		It("Should return the fields of a single parent", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
					name string = ""
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.InheritedFields(form.Extends, table)).To(SatisfyAll(
				HaveKey("port"), HaveKey("name"), Not(HaveKey("range")),
			))
		})

		It("Should resolve fields inherited through a grandparent", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Root struct {
					key string = ""
				}

				Middle struct extends Root {
					port string = ""
				}

				Child struct extends Middle {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.InheritedFields(form.Extends, table)).
				To(SatisfyAll(HaveKey("key"), HaveKey("port")))
		})

		It("Should let the leftmost parent win a name collision", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				First struct {
					port string = "first"
				}

				Second struct {
					port string = "second"
				}

				Child struct extends First, Second {
					range float64 = 10
				}
			`, "Child")
			inherited := resolver.InheritedFields(form.Extends, table)
			Expect(inherited["port"].Default.StringValue).To(Equal("first"))
		})

		It("Should substitute a generic parent's type arguments", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Box struct<T> {
					value T
				}

				Child struct extends Box<string> {
					range float64 = 10
				}
			`, "Child")
			inherited := resolver.InheritedFields(form.Extends, table)
			Expect(inherited["value"].Type.Name).To(Equal("string"))
		})

		It("Should return no fields when the struct has no parents", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Solo struct {
					port string = ""
				}
			`, "Solo")
			Expect(resolver.InheritedFields(form.Extends, table)).To(BeEmpty())
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

	Describe("HasStructuralOverride", func() {
		It("Should be false when nothing is redeclared", func(ctx SpecContext) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				Base struct {
					port string = ""
				}

				Child struct extends Base {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.HasStructuralOverride(form, table)).To(BeFalse())
		})

		It("Should be false when a redeclaration changes only the default", func(
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
			Expect(resolver.HasStructuralOverride(form, table)).To(BeFalse())
		})

		It("Should be true when a redeclaration changes the type", func(
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
			Expect(resolver.HasStructuralOverride(form, table)).To(BeTrue())
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
			Expect(resolver.CanUseInheritance(form, table)).To(BeFalse())
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
			Expect(resolver.CanUseInheritance(form, table)).To(BeTrue())
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
			Expect(resolver.CanUseInheritance(form, table)).To(BeTrue())
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
			Expect(resolver.CanUseInheritance(form, table)).To(BeFalse())
		})

		It("Should be false when parents declare the same field", func(
			ctx SpecContext,
		) {
			form, table := analyze(ctx, `
				@go output "pkg/test"

				First struct {
					port string = "first"
				}

				Second struct {
					port string = "second"
				}

				Child struct extends First, Second {
					range float64 = 10
				}
			`, "Child")
			Expect(resolver.CanUseInheritance(form, table)).To(BeFalse())
		})
	})
})
