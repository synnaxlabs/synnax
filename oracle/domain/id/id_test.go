// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package id_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/id"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Collect", func() {
	It("should return empty for nil input", func() {
		Expect(id.Collect(nil, nil)).To(BeEmpty())
	})

	It("should return empty for structs without id domain", func() {
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{{
				Name:    "name",
				TypeRef: &resolution.TypeRef{Primitive: "string"},
				Domains: map[string]*resolution.DomainEntry{},
			}},
		}}
		Expect(id.Collect(structs, nil)).To(BeEmpty())
	})

	It("should collect field with id domain", func() {
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]*resolution.DomainEntry{"id": {}},
			}},
		}}
		result := id.Collect(structs, nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("key"))
		Expect(result[0].Primitive).To(Equal("uuid"))
	})

	It("should deduplicate by field name", func() {
		structs := []*resolution.StructEntry{
			{Fields: []*resolution.FieldEntry{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]*resolution.DomainEntry{"id": {}},
			}}},
			{Fields: []*resolution.FieldEntry{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]*resolution.DomainEntry{"id": {}},
			}}},
		}
		Expect(id.Collect(structs, nil)).To(HaveLen(1))
	})

	It("should collect multiple different id fields", func() {
		structs := []*resolution.StructEntry{{
			Fields: []*resolution.FieldEntry{
				{
					Name:    "key",
					TypeRef: &resolution.TypeRef{Primitive: "uuid"},
					Domains: map[string]*resolution.DomainEntry{"id": {}},
				},
				{
					Name:    "rack",
					TypeRef: &resolution.TypeRef{Primitive: "uint32"},
					Domains: map[string]*resolution.DomainEntry{"id": {}},
				},
			},
		}}
		result := id.Collect(structs, nil)
		Expect(result).To(HaveLen(2))
	})

	It("should skip structs when skip function returns true", func() {
		structs := []*resolution.StructEntry{{
			Name: "Skipped",
			Fields: []*resolution.FieldEntry{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]*resolution.DomainEntry{"id": {}},
			}},
		}}
		skip := func(s *resolution.StructEntry) bool { return s.Name == "Skipped" }
		Expect(id.Collect(structs, skip)).To(BeEmpty())
	})

	It("should not skip when skip function returns false", func() {
		structs := []*resolution.StructEntry{{
			Name: "NotSkipped",
			Fields: []*resolution.FieldEntry{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]*resolution.DomainEntry{"id": {}},
			}},
		}}
		skip := func(s *resolution.StructEntry) bool { return s.Name == "Skipped" }
		Expect(id.Collect(structs, skip)).To(HaveLen(1))
	})
})
