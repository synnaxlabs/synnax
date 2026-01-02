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
		types := []resolution.Type{{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		Expect(ontology.Extract(types, nil, nil)).To(BeNil())
	})

	It("should return nil when no type has ontology domain", func() {
		types := []resolution.Type{{
			Form:    resolution.StructForm{},
			Domains: map[string]resolution.Domain{},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(types, keyFields, nil)).To(BeNil())
	})

	It("should return nil when ontology has no type expression", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		Expect(ontology.Extract(types, keyFields, nil)).To(BeNil())
	})

	It("should extract ontology data", func() {
		types := []resolution.Type{{
			Name: "User",
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		result := ontology.Extract(types, keyFields, nil)
		Expect(result).NotTo(BeNil())
		Expect(result.TypeName).To(Equal("user"))
		Expect(result.StructName).To(Equal("User"))
		Expect(result.KeyField.Name).To(Equal("key"))
		Expect(result.KeyField.Primitive).To(Equal("uuid"))
	})

	It("should skip types when skip function returns true", func() {
		types := []resolution.Type{{
			Name: "Skipped",
			Form: resolution.StructForm{},
			Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "user"}},
				}}},
			},
		}}
		keyFields := []key.Field{{Name: "key", Primitive: "uuid"}}
		skip := func(t resolution.Type) bool { return t.Name == "Skipped" }
		Expect(ontology.Extract(types, keyFields, skip)).To(BeNil())
	})

	It("should use first matching type with ontology domain", func() {
		types := []resolution.Type{
			{Name: "First", Form: resolution.StructForm{}, Domains: map[string]resolution.Domain{}},
			{Name: "Second", Form: resolution.StructForm{}, Domains: map[string]resolution.Domain{
				"ontology": {Expressions: []resolution.Expression{{
					Name:   "type",
					Values: []resolution.ExpressionValue{{StringValue: "task"}},
				}}},
			}},
		}
		keyFields := []key.Field{{Name: "key", Primitive: "uint32"}}
		result := ontology.Extract(types, keyFields, nil)
		Expect(result.StructName).To(Equal("Second"))
		Expect(result.TypeName).To(Equal("task"))
	})
})
