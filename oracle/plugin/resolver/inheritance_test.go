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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Inheritance", func() {
	var (
		table    *resolution.Table
		keyed    = resolution.TypeRef{Name: "test.Keyed"}
		named    = resolution.TypeRef{Name: "test.Named"}
		alsoKey  = resolution.TypeRef{Name: "test.AlsoKeyed"}
		missing  = resolution.TypeRef{Name: "test.Missing"}
		addTypes = func(types ...resolution.Type) {
			GinkgoHelper()
			for _, t := range types {
				Expect(table.Add(t)).To(Succeed())
			}
		}
	)

	structType := func(name string, fields ...string) resolution.Type {
		form := resolution.StructForm{}
		for _, f := range fields {
			form.Fields = append(form.Fields, resolution.Field{
				Name: f,
				Type: resolution.TypeRef{Name: "string"},
			})
		}
		return resolution.Type{
			Name:          name,
			Namespace:     "test",
			QualifiedName: "test." + name,
			Form:          form,
		}
	}

	BeforeEach(func() {
		table = resolution.NewTable()
		addTypes(
			structType("Keyed", "key"),
			structType("Named", "name"),
			structType("AlsoKeyed", "key"),
		)
	})

	Describe("HasFieldConflicts", func() {
		DescribeTable("Should detect overlapping fields across parents",
			func(extends []resolution.TypeRef, want bool) {
				Expect(resolver.HasFieldConflicts(extends, table)).To(Equal(want))
			},
			Entry("no parents", nil, false),
			Entry("single parent", []resolution.TypeRef{keyed}, false),
			Entry("disjoint parents", []resolution.TypeRef{keyed, named}, false),
			Entry("conflicting parents", []resolution.TypeRef{keyed, alsoKey}, true),
			Entry("unresolvable parent skipped",
				[]resolution.TypeRef{keyed, missing}, false),
		)
	})

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

	Describe("CanUseInheritance", func() {
		It("Should allow inheritance for a clean single-parent extension", func() {
			form := resolution.StructForm{Extends: []resolution.TypeRef{keyed}}
			Expect(resolver.CanUseInheritance(form, table)).To(BeTrue())
		})

		DescribeTable("Should reject shapes that require flattening",
			func(form resolution.StructForm) {
				Expect(resolver.CanUseInheritance(form, table)).To(BeFalse())
			},
			Entry("no parents", resolution.StructForm{}),
			Entry("omitted fields", resolution.StructForm{
				Extends:       []resolution.TypeRef{keyed},
				OmittedFields: []string{"key"},
			}),
			Entry("domain omissions", resolution.StructForm{
				Extends: []resolution.TypeRef{keyed},
				Fields: []resolution.Field{
					{Name: "key", OmittedDomains: []string{"validate"}},
				},
			}),
			Entry("field conflicts", resolution.StructForm{
				Extends: []resolution.TypeRef{keyed, alsoKey},
			}),
		)
	})
})
