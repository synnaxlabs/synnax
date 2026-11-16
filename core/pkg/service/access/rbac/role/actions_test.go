// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	base := role.Role{Key: uuid.New(), Name: "operator", Description: "runs tests"}
	tags := []string{"tags"}

	Describe("RemovePayload", func() {
		It("Should reject the removal of the description", func() {
			Expect(role.Reduce(
				base,
				role.NewRemoveAction(role.RemovePayload{Key: "description"}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[description]: the document requires")),
			))
		})
		It("Should leave the role unchanged when the field is missing", func() {
			Expect(role.Reduce(
				base,
				role.NewRemoveAction(role.RemovePayload{Key: "tags"}),
			)).To(Equal(base))
		})
	})

	Describe("InsertPayload", func() {
		It("Should reject an insert into the name", func() {
			Expect(role.Reduce(base, role.NewInsertAction(role.InsertPayload{
				Path:  []string{"name"},
				Value: map[string]any{"key": "a"},
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: the value is not a list")),
			))
		})
		It("Should reject a list the role does not declare", func() {
			Expect(role.Reduce(base, role.NewInsertAction(role.InsertPayload{
				Path:  tags,
				Value: map[string]any{"key": "a"},
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("decoding document")),
			))
		})
	})

	Describe("MovePayload", func() {
		It("Should reject a move in the name", func() {
			Expect(role.Reduce(base, role.NewMoveAction(role.MovePayload{
				Path: []string{"name"},
				Key:  "a",
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: the value is not a list")),
			))
		})
		It("Should leave the role unchanged when the list is missing", func() {
			Expect(role.Reduce(base, role.NewMoveAction(role.MovePayload{
				Path: tags,
				Key:  "a",
			}))).To(Equal(base))
		})
	})
})
