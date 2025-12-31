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
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Extract", func() {
	It("should return nil for empty key fields", func() {
		structs := []resolution.Struct{{
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		Expect(ontology.Extract(structs, nil, nil)).To(BeNil())
	})

	It("should return nil when no struct has ontology domain", func() {
		structs := []resolution.Struct{{Domains: map[string]resolution.Domain{}}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(structs, keyFields, nil)).To(BeNil())
	})

	It("should return nil when ontology has no type expression", func() {
		structs := []resolution.Struct{{
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(structs, keyFields, nil)).To(BeNil())
	})

	It("should extract ontology data", func() {
		structs := []resolution.Struct{{
			Name: "User",
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		result := ontology.Extract(structs, keyFields, nil)
		Expect(result).NotTo(BeNil())
		Expect(result.TypeName).To(Equal("user"))
		Expect(result.StructName).To(Equal("User"))
		Expect(result.KeyField.Name).To(Equal("key"))
		Expect(result.KeyField.Primitive).To(Equal("uuid"))
	})

	It("should skip structs when skip function returns true", func() {
		structs := []resolution.Struct{{
			Name: "Skipped",
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		skip := func(s resolution.Struct) bool { return s.Name == "Skipped" }
		Expect(ontology.Extract(structs, keyFields, skip)).To(BeNil())
	})

	It("should use first matching struct with ontology domain", func() {
		structs := []resolution.Struct{
			{Name: "First", Domains: map[string]resolution.Domain{}},
			{Name: "Second", Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "task"}},
				}}},
			}},
		}
		keyFields := []key.Field{{Name: "key", Primitive: "uint32"}}
		result := ontology.Extract(structs, keyFields, nil)
		Expect(result.StructName).To(Equal("Second"))
		Expect(result.TypeName).To(Equal("task"))
	})
})
