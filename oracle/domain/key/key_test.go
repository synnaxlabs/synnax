// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package key_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/resolution"
)

var _ = Describe("Collect", func() {
	It("should return empty for nil input", func() {
		Expect(key.Collect(nil, nil)).To(BeEmpty())
	})

	It("should return empty for structs without key domain", func() {
		structs := []resolution.Struct{{
			Fields: []resolution.Field{{
				Name:    "name",
				TypeRef: &resolution.TypeRef{Primitive: "string"},
				Domains: map[string]resolution.Domain{},
			}},
		}}
		Expect(key.Collect(structs, nil)).To(BeEmpty())
	})

	It("should collect field with key domain", func() {
		structs := []resolution.Struct{{
			Fields: []resolution.Field{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}},
		}}
		result := key.Collect(structs, nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("key"))
		Expect(result[0].Primitive).To(Equal("uuid"))
	})

	It("should deduplicate by field name", func() {
		structs := []resolution.Struct{
			{Fields: []resolution.Field{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}},
			{Fields: []resolution.Field{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}},
		}
		Expect(key.Collect(structs, nil)).To(HaveLen(1))
	})

	It("should collect multiple different key fields", func() {
		structs := []resolution.Struct{{
			Fields: []resolution.Field{
				{
					Name:    "key",
					TypeRef: &resolution.TypeRef{Primitive: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				},
				{
					Name:    "rack",
					TypeRef: &resolution.TypeRef{Primitive: "uint32"},
					Domains: map[string]resolution.Domain{"key": {}},
				},
			},
		}}
		result := key.Collect(structs, nil)
		Expect(result).To(HaveLen(2))
	})

	It("should skip structs when skip function returns true", func() {
		structs := []resolution.Struct{{
			Name: "Skipped",
			Fields: []resolution.Field{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}},
		}}
		skip := func(s resolution.Struct) bool { return s.Name == "Skipped" }
		Expect(key.Collect(structs, skip)).To(BeEmpty())
	})

	It("should not skip when skip function returns false", func() {
		structs := []resolution.Struct{{
			Name: "NotSkipped",
			Fields: []resolution.Field{{
				Name:    "key",
				TypeRef: &resolution.TypeRef{Primitive: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}},
		}}
		skip := func(s resolution.Struct) bool { return s.Name == "Skipped" }
		Expect(key.Collect(structs, skip)).To(HaveLen(1))
	})
})
