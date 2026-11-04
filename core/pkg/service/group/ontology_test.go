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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ResourceIDsFromGroups", func() {
	It("Should map a slice of groups to resource IDs", func() {
		a := group.Group{Key: uuid.New(), Name: "a"}
		b := group.Group{Key: uuid.New(), Name: "b"}
		Expect(group.ResourceIDsFromGroups([]group.Group{a, b})).
			To(ConsistOf(a.ResourceID(), b.ResourceID()))
	})
})

var _ = Describe("KeyFromResourceID", func() {
	It("Should return the key of a group ID", func() {
		key := uuid.New()
		Expect(group.KeyFromResourceID(group.ResourceID(key))).To(Equal(key))
	})
	DescribeTable("Should reject an ID that is not a group",
		func(id resource.ID, msg string) {
			Expect(group.KeyFromResourceID(id)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a project",
			resource.ID{Type: resource.TypeProject, Key: uuid.New().String()},
			`must be a group, got "project"`),
		Entry("a zero ID", resource.ID{}, `must be a group, got ""`),
		Entry("a group whose key is not a UUID",
			resource.ID{Type: resource.TypeGroup, Key: "not-a-uuid"},
			`invalid group key "not-a-uuid"`),
		Entry("a group with an empty key",
			resource.ID{Type: resource.TypeGroup},
			`invalid group key ""`),
	)
})
