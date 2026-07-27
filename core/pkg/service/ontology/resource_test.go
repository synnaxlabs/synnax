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
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Resource", func() {
	Describe("ParseID", func() {
		It("Should parse an ID from a string", func() {
			Expect(ontology.ParseID("channel:bar")).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeChannel,
				Key:  "bar",
			}))
		})
		It("Should return an error if the ID has an invalid structure", func() {
			Expect(ontology.ParseID("foo")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: foo")),
				))
		})
		It("Should return an error if the ID is an empty string", func() {
			Expect(ontology.ParseID("")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: ")),
				))
		})
		It("Should return an error if the ID has an empty type (leading colon)", func() {
			Expect(ontology.ParseID(":bar")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: :bar (empty type)")),
				))
		})
		It("Should return an error if the ID has an empty type with colons in key", func() {
			Expect(ontology.ParseID(":word1:word2")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: :word1:word2 (empty type)")),
				))
		})
		It("Should return an error if the ID has an empty type and key starts with colon", func() {
			Expect(ontology.ParseID("::word1")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: ::word1 (empty type)")),
				))
		})
		It("Should parse an ID with empty key (trailing colon)", func() {
			Expect(ontology.ParseID("channel:")).
				To(Equal(ontology.ID{Type: "channel", Key: ""}))
		})
		It("Should ignore subsequent colons in the key", func() {
			Expect(ontology.ParseID("channel:bar:baz")).
				To(Equal(ontology.ID{Type: "channel", Key: "bar:baz"}))
		})
	})
	Describe("ParseIDs", func() {
		It("Should parse a list of IDs from a list of strings", func() {
			Expect(ontology.ParseIDs([]string{"channel:bar", "channel:baz"})).
				To(ConsistOf(
					ontology.ID{Type: "channel", Key: "bar"},
					ontology.ID{Type: "channel", Key: "baz"},
				))
		})
		It("Should return an error if any of the IDs have an invalid structure", func() {
			Expect(ontology.ParseIDs([]string{"channel:bar", "foo"})).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("[ontology.resource] - failed to parse id: foo")),
				))
		})
		It("Should return an empty slice when given an empty slice", func() {
			Expect(ontology.ParseIDs([]string{})).To(BeEmpty())
		})
	})
	Describe("IDsToKeys", func() {
		It("Should convert IDs to their keys", func() {
			ids := []ontology.ID{
				{Type: "cat", Key: "dog1"},
				{Type: "cat", Key: "dog2"},
			}
			strings := ontology.IDsToKeys(ids)
			Expect(strings).To(ConsistOf("cat:dog1", "cat:dog2"))
		})
		It("Should return an empty slice for empty input", func() {
			Expect(ontology.IDsToKeys([]ontology.ID{})).To(BeEmpty())
		})
	})
	Describe("Resource", func() {
		var r ontology.Resource
		BeforeEach(func() {
			r = ontology.NewResource(
				zyn.Object(nil),
				ontology.ID{Type: ontology.ResourceTypeChannel, Key: "dog"},
				"cat",
				map[string]any{},
			)
		})
		It("Should correctly construct the resource", func() {
			Expect(r.ID.Type).To(Equal(ontology.ResourceTypeChannel))
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
					ontology.ID{Type: "cat", Key: "dog1"},
					"cat1",
					map[string]any{},
				),
				ontology.NewResource(
					zyn.Object(nil),
					ontology.ID{Type: "cat", Key: "dog2"},
					"cat2",
					map[string]any{},
				),
			}
			ids := ontology.ResourceIDs(resources)
			Expect(ids).To(ConsistOf(
				ontology.ID{Type: "cat", Key: "dog1"},
				ontology.ID{Type: "cat", Key: "dog2"},
			))
		})
		It("Should return an empty slice for empty input", func() {
			Expect(ontology.ResourceIDs([]ontology.Resource{})).To(BeEmpty())
		})
	})
})
