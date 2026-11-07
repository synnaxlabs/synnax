// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Resource", func() {
	Describe("Resource", func() {
		var r ontology.Resource
		BeforeEach(func() {
			r = ontology.NewResource(
				zyn.Object(nil),
				resource.ID{Type: resource.TypeChannel, Key: "dog"},
				"cat",
				map[string]any{},
			)
		})
		It("Should correctly construct the resource", func() {
			Expect(r.ID.Type).To(Equal(resource.TypeChannel))
			Expect(r.ID.Key).To(Equal("dog"))
			Expect(r.Name).To(Equal("cat"))
			Expect(r.Data).To(BeEmpty())
		})
		Describe("GorpKey", func() {
			It("Should return the ID as the gorp key of the resource", func() {
				Expect(r.GorpKey()).To(Equal(r.ID.String()))
			})
		})
		Describe("SetOptions", func() {
			It("Should return an empty slice", func() {
				Expect(r.SetOptions()).To(BeEmpty())
			})
		})
	})
	Describe("ResourceIDs", func() {
		It("Should extract IDs from a slice of resources", func() {
			resources := []ontology.Resource{
				ontology.NewResource(
					zyn.Object(nil),
					resource.ID{Type: "cat", Key: "dog1"},
					"cat1",
					map[string]any{},
				),
				ontology.NewResource(
					zyn.Object(nil),
					resource.ID{Type: "cat", Key: "dog2"},
					"cat2",
					map[string]any{},
				),
			}
			ids := ontology.ResourceIDs(resources)
			Expect(ids).To(ConsistOf(
				resource.ID{Type: "cat", Key: "dog1"},
				resource.ID{Type: "cat", Key: "dog2"},
			))
		})
		It("Should return an empty slice for empty input", func() {
			Expect(ontology.ResourceIDs([]ontology.Resource{})).To(BeEmpty())
		})
	})
})
