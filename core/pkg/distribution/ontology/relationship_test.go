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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Relationship", func() {
	Describe("GorpKey", func() {
		It("Should return the correct gorp key", func() {
			Expect(ontology.Relationship{
				From: ontology.ID{Type: "channel", Key: "qux"},
				To:   ontology.ID{Type: "device", Key: "baz"},
				Type: ontology.RelationshipTypeParentOf,
			}.GorpKey()).To(Equal("channel:qux->parent->device:baz"))
		})
		Describe("SetOptions", func() {
			It("Should return nil", func() {
				Expect(ontology.Relationship{}.SetOptions()).To(BeNil())
			})
		})
		Describe("ParseRelationship", func() {
			It("Should parse a relationship from a string", func() {
				Expect((ontology.ParseRelationship("channel:qux->parent->device:baz"))).
					To(Equal(ontology.Relationship{
						From: ontology.ID{Type: "channel", Key: "qux"},
						To:   ontology.ID{Type: "device", Key: "baz"},
						Type: ontology.RelationshipTypeParentOf,
					}))
			})
			DescribeTable("Errors", func(key, message string) {
				Expect(
					ontology.ParseRelationship(key),
				).Error().To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
				Entry("Invalid structure", "foo:qux-parent->bar", "invalid relationship key"),
				Entry("Invalid from", "badfrom->parent->bar", "failed to parse id"),
				Entry("Invalid to", "foo:qux->parent->badto", "failed to parse id"),
			)
		})
	})
})
