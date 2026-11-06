// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resource_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ID", func() {
	Describe("ParseUUIDKey", func() {
		It("Should parse the key of an ID as a UUID", func() {
			key := uuid.New()
			Expect(resource.ParseUUIDKey(resource.ID{
				Type: resource.TypeLog,
				Key:  key.String(),
			})).To(Equal(key))
		})

		It("Should return a validation error for a malformed key", func() {
			Expect(resource.ParseUUIDKey(resource.ID{
				Type: resource.TypeLog,
				Key:  "pump",
			})).Error().To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`invalid log key "pump"`)),
			))
		})
	})

	Describe("ParseID", func() {
		It("Should parse an ID from a string", func() {
			Expect(resource.ParseID("channel:bar")).To(Equal(resource.ID{
				Type: resource.TypeChannel,
				Key:  "bar",
			}))
		})
		It("Should return an error if the ID has an invalid structure", func() {
			Expect(resource.ParseID("foo")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring(
							"[resource] - failed to parse id: foo",
						),
					),
				))
		})
		It("Should return an error if the ID is an empty string", func() {
			Expect(resource.ParseID("")).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(
						ContainSubstring("[resource] - failed to parse id: "),
					),
				))
		})
		It(
			"Should return an error if the ID has an empty type (leading colon)",
			func() {
				Expect(resource.ParseID(":bar")).Error().
					To(And(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"[resource] - failed to parse id: :bar (empty type)",
							),
						),
					))
			},
		)
		It(
			"Should return an error if the ID has an empty type with colons in key",
			func() {
				Expect(resource.ParseID(":word1:word2")).Error().
					To(And(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"[resource] - failed to parse id: :word1:word2 (empty type)",
							),
						),
					))
			},
		)
		It(
			"Should return an error if the ID has an empty type and key starts with colon",
			func() {
				Expect(resource.ParseID("::word1")).Error().
					To(And(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"[resource] - failed to parse id: ::word1 (empty type)",
							),
						),
					))
			},
		)
		It("Should parse an ID with empty key (trailing colon)", func() {
			Expect(resource.ParseID("channel:")).
				To(Equal(resource.ID{Type: "channel", Key: ""}))
		})
		It("Should ignore subsequent colons in the key", func() {
			Expect(resource.ParseID("channel:bar:baz")).
				To(Equal(resource.ID{Type: "channel", Key: "bar:baz"}))
		})
	})
	Describe("ParseIDs", func() {
		It("Should parse a list of IDs from a list of strings", func() {
			Expect(resource.ParseIDs([]string{"channel:bar", "channel:baz"})).
				To(ConsistOf(
					resource.ID{Type: "channel", Key: "bar"},
					resource.ID{Type: "channel", Key: "baz"},
				))
		})
		It(
			"Should return an error if any of the IDs have an invalid structure",
			func() {
				Expect(resource.ParseIDs([]string{"channel:bar", "foo"})).Error().
					To(And(
						MatchError(validate.ErrValidation),
						MatchError(
							ContainSubstring(
								"[resource] - failed to parse id: foo",
							),
						),
					))
			},
		)
		It("Should return an empty slice when given an empty slice", func() {
			Expect(resource.ParseIDs([]string{})).To(BeEmpty())
		})
	})
	Describe("IDsToKeys", func() {
		It("Should convert IDs to their keys", func() {
			ids := []resource.ID{
				{Type: "cat", Key: "dog1"},
				{Type: "cat", Key: "dog2"},
			}
			strings := resource.IDsToKeys(ids)
			Expect(strings).To(ConsistOf("cat:dog1", "cat:dog2"))
		})
		It("Should return an empty slice for empty input", func() {
			Expect(resource.IDsToKeys([]resource.ID{})).To(BeEmpty())
		})
	})
})
