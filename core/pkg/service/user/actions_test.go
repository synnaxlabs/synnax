// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Actions", func() {
	base := user.User{Key: uuid.New(), Name: "alice", Role: uuid.New()}
	tags := []string{"tags"}

	Describe("RemovePayload", func() {
		It("Should reject the removal of the role", func() {
			Expect(user.Reduce(
				base,
				user.NewRemoveAction(user.RemovePayload{Key: "role"}),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[role]: the document requires a value")),
			))
		})
		It("Should leave the user unchanged when the field is missing", func() {
			Expect(user.Reduce(
				base,
				user.NewRemoveAction(user.RemovePayload{Key: "tags"}),
			)).To(Equal(base))
		})
	})

	Describe("InsertPayload", func() {
		It("Should reject an insert into the name", func() {
			Expect(user.Reduce(base, user.NewInsertAction(user.InsertPayload{
				Path:  []string{"name"},
				Value: map[string]any{"key": "a"},
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: the value is not a list")),
			))
		})
		It("Should reject a list the user does not declare", func() {
			Expect(user.Reduce(base, user.NewInsertAction(user.InsertPayload{
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
			Expect(user.Reduce(base, user.NewMoveAction(user.MovePayload{
				Path: []string{"name"},
				Key:  "a",
			}))).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("[name]: the value is not a list")),
			))
		})
		It("Should leave the user unchanged when the list is missing", func() {
			Expect(user.Reduce(base, user.NewMoveAction(user.MovePayload{
				Path: tags,
				Key:  "a",
			}))).To(Equal(base))
		})
	})

	Describe("SetUsernamePayload", func() {
		It("Should leave the user unchanged", func() {
			Expect(user.Reduce(
				base,
				user.NewSetUsernameAction(user.SetUsernamePayload{Username: "bob"}),
			)).To(Equal(base))
		})
	})
})
