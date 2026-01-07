// Copyright 2026 Synnax Labs, Inc.
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
		Expect(key.Collect(nil, nil, nil)).To(BeEmpty())
	})

	It("should return empty for structs without key domain", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "name",
					Type:    resolution.TypeRef{Name: "string"},
					Domains: map[string]resolution.Domain{},
				}},
			},
		}}
		Expect(key.Collect(types, nil, nil)).To(BeEmpty())
	})

	It("should collect field with key domain", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		result := key.Collect(types, nil, nil)
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("key"))
		Expect(result[0].Primitive).To(Equal("uuid"))
	})

	It("should deduplicate by field name", func() {
		types := []resolution.Type{
			{Form: resolution.StructForm{Fields: []resolution.Field{{
				Name:    "key",
				Type:    resolution.TypeRef{Name: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}}},
			{Form: resolution.StructForm{Fields: []resolution.Field{{
				Name:    "key",
				Type:    resolution.TypeRef{Name: "uuid"},
				Domains: map[string]resolution.Domain{"key": {}},
			}}}},
		}
		Expect(key.Collect(types, nil, nil)).To(HaveLen(1))
	})

	It("should collect multiple different key fields", func() {
		types := []resolution.Type{{
			Form: resolution.StructForm{
				Fields: []resolution.Field{
					{
						Name:    "key",
						Type:    resolution.TypeRef{Name: "uuid"},
						Domains: map[string]resolution.Domain{"key": {}},
					},
					{
						Name:    "rack",
						Type:    resolution.TypeRef{Name: "uint32"},
						Domains: map[string]resolution.Domain{"key": {}},
					},
				},
			},
		}}
		result := key.Collect(types, nil, nil)
		Expect(result).To(HaveLen(2))
	})

	It("should skip types when skip function returns true", func() {
		types := []resolution.Type{{
			Name: "Skipped",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		skip := func(t resolution.Type) bool { return t.Name == "Skipped" }
		Expect(key.Collect(types, nil, skip)).To(BeEmpty())
	})

	It("should not skip when skip function returns false", func() {
		types := []resolution.Type{{
			Name: "NotSkipped",
			Form: resolution.StructForm{
				Fields: []resolution.Field{{
					Name:    "key",
					Type:    resolution.TypeRef{Name: "uuid"},
					Domains: map[string]resolution.Domain{"key": {}},
				}},
			},
		}}
		skip := func(t resolution.Type) bool { return t.Name == "Skipped" }
		Expect(key.Collect(types, nil, skip)).To(HaveLen(1))
	})
})
