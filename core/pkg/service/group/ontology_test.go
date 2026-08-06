// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("OntologyIDsFromGroups", func() {
	It("Should map a slice of groups to ontology IDs", func() {
		a := group.Group{Key: uuid.New(), Name: "a"}
		b := group.Group{Key: uuid.New(), Name: "b"}
		Expect(group.OntologyIDsFromGroups([]group.Group{a, b})).
			To(ConsistOf(a.OntologyID(), b.OntologyID()))
	})
})

var _ = Describe("KeyFromOntologyID", func() {
	It("Should return the key of a group ID", func() {
		key := uuid.New()
		Expect(group.KeyFromOntologyID(group.OntologyID(key))).To(Equal(key))
	})
	DescribeTable("Should reject an ID that is not a group",
		func(id ontology.ID, msg string) {
			Expect(group.KeyFromOntologyID(id)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a project",
			ontology.ID{Type: ontology.ResourceTypeProject, Key: uuid.NewString()},
			`must be a group, got "project"`),
		Entry("a zero ID", ontology.ID{}, `must be a group, got ""`),
		Entry("a group whose key is not a UUID",
			ontology.ID{Type: ontology.ResourceTypeGroup, Key: "not-a-uuid"},
			`invalid group key "not-a-uuid"`),
		Entry("a group with an empty key",
			ontology.ID{Type: ontology.ResourceTypeGroup},
			`invalid group key ""`),
	)
})
