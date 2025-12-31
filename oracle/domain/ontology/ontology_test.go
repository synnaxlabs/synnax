// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ontology_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/id"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Extract", func() {
	It("should return nil for empty id fields", func() {
		structs := []*resolution.StructEntry{{
			Domains: map[string]*resolution.DomainEntry{
				"ontology": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		Expect(ontology.Extract(structs, nil, nil)).To(BeNil())
	})

	It("should return nil when no struct has ontology domain", func() {
		structs := []*resolution.StructEntry{{Domains: map[string]*resolution.DomainEntry{}}}
		idFields := []id.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(structs, idFields, nil)).To(BeNil())
	})

	It("should return nil when ontology has no type expression", func() {
		structs := []*resolution.StructEntry{{
			Domains: map[string]*resolution.DomainEntry{
				"ontology": {Expressions: []*resolution.ExpressionEntry{}},
			},
		}}
		idFields := []id.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(structs, idFields, nil)).To(BeNil())
	})

	It("should extract ontology data", func() {
		structs := []*resolution.StructEntry{{
			Name: "User",
			Domains: map[string]*resolution.DomainEntry{
				"ontology": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		idFields := []id.Field{{Name: "key", Primitive: "uuid"}}
		result := ontology.Extract(structs, idFields, nil)
		Expect(result).NotTo(BeNil())
		Expect(result.TypeName).To(Equal("user"))
		Expect(result.StructName).To(Equal("User"))
		Expect(result.IDField.Name).To(Equal("key"))
		Expect(result.IDField.Primitive).To(Equal("uuid"))
	})

	It("should skip structs when skip function returns true", func() {
		structs := []*resolution.StructEntry{{
			Name: "Skipped",
			Domains: map[string]*resolution.DomainEntry{
				"ontology": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		idFields := []id.Field{{Name: "key", Primitive: "uuid"}}
		skip := func(s *resolution.StructEntry) bool { return s.Name == "Skipped" }
		Expect(ontology.Extract(structs, idFields, skip)).To(BeNil())
	})

	It("should use first matching struct with ontology domain", func() {
		structs := []*resolution.StructEntry{
			{Name: "First", Domains: map[string]*resolution.DomainEntry{}},
			{Name: "Second", Domains: map[string]*resolution.DomainEntry{
				"ontology": {Expressions: []*resolution.ExpressionEntry{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "task"}},
				}}},
			}},
		}
		idFields := []id.Field{{Name: "key", Primitive: "uint32"}}
		result := ontology.Extract(structs, idFields, nil)
		Expect(result.StructName).To(Equal("Second"))
		Expect(result.TypeName).To(Equal("task"))
	})
})
