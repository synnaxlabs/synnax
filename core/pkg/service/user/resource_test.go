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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Resources", func() {
	It("Should name each user by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Patrick Star"}))
		Expect(svc.Resources().Retrieve(ctx, tx, []resource.ID{u.ResourceID()})).
			To(Equal([]resource.Resource{{ID: u.ResourceID(), Name: "Patrick Star"}}))
	})

	It("Should return a validation error for a malformed key", func(ctx SpecContext) {
		Expect(svc.Resources().Retrieve(
			ctx,
			nil,
			[]resource.ID{{Type: resource.TypeUser, Key: "pump"}},
		)).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`invalid user key "pump"`)),
		))
	})

	It("Should leave out an ID that names no user", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Sandy Cheeks"}))
		missing := resource.ID{Type: resource.TypeUser, Key: uuid.New().String()}
		Expect(svc.Resources().Retrieve(
			ctx,
			tx,
			[]resource.ID{missing, u.ResourceID()},
		)).To(Equal([]resource.Resource{{ID: u.ResourceID(), Name: u.Name}}))
	})

	It("Should list each user by its name", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		u := MustSucceed(svc.NewWriter(tx).Create(ctx, user.User{Name: "Sandy Cheeks"}))
		Expect(svc.Resources().List(ctx, tx, 10000, nil)).To(ContainElement(
			resource.Resource{ID: u.ResourceID(), Name: u.Name},
		))
	})
})

var _ = Describe("Resource", func() {
	Describe("ResourceID", func() {
		It(
			"Should build a resource ID with the user resource type and the key",
			func() {
				key := uuid.New()
				id := user.ResourceID(key)
				Expect(id.Type).To(Equal(resource.TypeUser))
				Expect(id.Key).To(Equal(key.String()))
			},
		)
	})
	Describe("ResourceIDsFromKeys", func() {
		It("Should map a slice of keys to a slice of resource IDs", func() {
			a, b := uuid.New(), uuid.New()
			ids := user.ResourceIDsFromKeys([]user.Key{a, b})
			Expect(ids).To(HaveLen(2))
			Expect(ids[0]).To(Equal(user.ResourceID(a)))
			Expect(ids[1]).To(Equal(user.ResourceID(b)))
		})
		It("Should return an empty slice when given no keys", func() {
			Expect(user.ResourceIDsFromKeys(nil)).To(BeEmpty())
		})
	})
	Describe("User.ResourceID", func() {
		It("Should return a resource ID populated from the user's key", func() {
			u := user.User{Key: uuid.New(), Name: "alice"}
			Expect(u.ResourceID()).To(Equal(u.ResourceID()))
		})
	})
	Describe("ResourceIDsFromUsers", func() {
		It("Should map a slice of users to a slice of resource IDs", func() {
			a := user.User{Key: uuid.New(), Name: "a"}
			b := user.User{Key: uuid.New(), Name: "b"}
			ids := user.ResourceIDsFromUsers([]user.User{a, b})
			Expect(ids).To(ConsistOf(a.ResourceID(), b.ResourceID()))
		})
		It("Should return an empty slice when given no users", func() {
			Expect(user.ResourceIDsFromUsers(nil)).To(BeEmpty())
		})
	})
	Describe("KeyFromResourceID", func() {
		It("Should round-trip a key through ResourceID", func() {
			key := uuid.New()
			Expect(user.KeyFromResourceID(user.ResourceID(key))).To(Equal(key))
		})
		It("Should return an error when the ID key is not a valid UUID", func() {
			Expect(user.KeyFromResourceID(resource.ID{
				Type: resource.TypeUser,
				Key:  "not-a-uuid",
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`invalid user key "not-a-uuid"`)),
			))
		})
	})
	Describe("Search", func() {
		DescribeTable("Should search users by each word of their name",
			func(ctx SpecContext, term string) {
				created := MustSucceed(writer.Create(ctx, user.User{
					Name: "Persephone Quintarelli",
				}))
				DeferCleanup(func(ctx SpecContext) {
					Expect(writer.Delete(ctx, created.Key)).To(Succeed())
				})
				q := search.Query[resource.ID]{Term: term, Limit: 10}
				Expect(MustSucceed(svc.Resources().Search(ctx, db, q)).Keys).
					To(ContainElement(created.ResourceID()))
			},
			Entry("first word", "Persephone"),
			Entry("last word", "Quintarelli"),
		)
	})
})
