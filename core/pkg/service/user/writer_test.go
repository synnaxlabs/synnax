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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  user.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx)
	})
	exists := func(ctx SpecContext, key user.Key) bool {
		GinkgoHelper()
		return MustSucceed(svc.NewRetrieve().Where(user.MatchKeys(key)).Exists(ctx, tx))
	}
	roleOf := func(ctx SpecContext, key user.Key) role.Key {
		GinkgoHelper()
		var u user.User
		Expect(svc.NewRetrieve().
			Where(user.MatchKeys(key)).
			Entry(&u).
			Exec(ctx, tx)).To(Succeed())
		return u.Role
	}
	Describe("Create", func() {
		It("Should refuse an empty name", func(ctx SpecContext) {
			Expect(w.Create(ctx, user.User{Role: member})).Error().
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should create a new user with the given key", func(ctx SpecContext) {
			key := uuid.New()
			u := MustSucceed(
				w.Create(ctx, user.User{Name: "Patrick Star", Key: key, Role: member}),
			)
			Expect(u.Key).To(Equal(key))
			Expect(u.Name).To(Equal("Patrick Star"))
			Expect(u.RootUser).To(BeFalse())
		})
		It(
			"Should assign a new key when none is provided on create",
			func(ctx SpecContext) {
				u := MustSucceed(
					w.Create(ctx, user.User{Name: "Patrick Star", Role: member}),
				)
				Expect(u.Key).ToNot(Equal(uuid.Nil()))
				Expect(exists(ctx, u.Key)).To(BeTrue())
			},
		)
		It("Should reject a key that another user holds", func(ctx SpecContext) {
			u := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: member}))
			Expect(w.Create(ctx, user.User{Key: u.Key, Name: "b", Role: member})).
				Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("user %s already exists", u.Key)),
				))
		})
		It("Should allow two users with the same name", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{Name: "Twin", Role: member}))
			b := MustSucceed(w.Create(ctx, user.User{Name: "Twin", Role: member}))
			Expect(a.Key).ToNot(Equal(b.Key))
		})
		It(
			"Should reject RootUser=true so the root-user invariant cannot be bypassed",
			func(ctx SpecContext) {
				Expect(w.Create(ctx, user.User{
					Name:     "root",
					RootUser: true,
					Role:     owner,
				})).Error().To(MatchError(ContainSubstring("cannot create a root user; root users are provisioned at startup")))
			},
		)
	})
	Describe("Role", func() {
		It("Should create a user with a role", func(ctx SpecContext) {
			r := createRole(ctx, tx, "role")
			u := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: r}))
			Expect(roleOf(ctx, u.Key)).To(Equal(r))
		})
		It("Should refuse to create a user with no role", func(ctx SpecContext) {
			Expect(w.Create(ctx, user.User{Name: "a"})).Error().
				To(MatchError(ContainSubstring("role: must be non-zero")))
		})
		It("Should refuse to create a user with a role that does not exist", func(
			ctx SpecContext,
		) {
			Expect(w.Create(ctx, user.User{
				Name: "a",
				Role: uuid.New(),
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("role: role")),
				MatchError(ContainSubstring("does not exist")),
			))
		})
		It("Should give the role to each user", func(ctx SpecContext) {
			next := createRole(ctx, tx, "next")
			a := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: member}))
			b := MustSucceed(w.Create(ctx, user.User{Name: "b", Role: member}))
			Expect(w.SetRole(ctx, next, a.Key, b.Key)).To(Succeed())
			Expect(roleOf(ctx, a.Key)).To(Equal(next))
			Expect(roleOf(ctx, b.Key)).To(Equal(next))
		})
		It("Should refuse to give a role that does not exist", func(ctx SpecContext) {
			u := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: member}))
			Expect(w.SetRole(ctx, uuid.New(), u.Key)).To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("does not exist")),
			))
			Expect(roleOf(ctx, u.Key)).To(Equal(member))
		})
		It("Should return not found for a user that does not exist", func(
			ctx SpecContext,
		) {
			Expect(w.SetRole(ctx, member, uuid.New())).
				To(MatchError(query.ErrNotFound))
		})
		It("Should retrieve the users that hold a role", func(ctx SpecContext) {
			r := createRole(ctx, tx, "role")
			u := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: r}))
			MustSucceed(w.Create(ctx, user.User{Name: "b", Role: member}))
			var users []user.User
			Expect(svc.NewRetrieve().
				Where(user.MatchRoles(r)).
				Entries(&users).
				Exec(ctx, tx)).To(Succeed())
			Expect(users).To(HaveExactElements(HaveField("Key", u.Key)))
		})
		It("Should refuse to delete a role that a user holds", func(ctx SpecContext) {
			r := createRole(ctx, tx, "held")
			u := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: r}))
			Expect(roleSvc.NewWriter(tx, false).Delete(ctx, r)).To(MatchError(
				gorp.RestrictError{
					Target:    gorp.NewTarget("role", r),
					Holder:    gorp.NewTarget("user", u.Key),
					Reference: "role",
				},
			))
		})
	})
	Describe("Delete", func() {
		It("Should delete a single user", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: member}))
			Expect(w.Delete(ctx, created.Key)).To(Succeed())
			var u user.User
			Expect(
				svc.NewRetrieve().
					Where(user.MatchKeys(created.Key)).
					Entry(&u).
					Exec(ctx, tx),
			).
				To(MatchError(query.ErrNotFound))
		})
		It("Should delete multiple users", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{Name: "a", Role: member}))
			b := MustSucceed(w.Create(ctx, user.User{Name: "b", Role: member}))
			Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(exists(ctx, a.Key)).To(BeFalse())
			Expect(exists(ctx, b.Key)).To(BeFalse())
		})
		It("Should not delete a root user", func(ctx SpecContext) {
			// Insert a root user directly via the raw gorp writer because
			// Writer.Create rejects RootUser=true.
			rootUser := user.User{Key: uuid.New(), Name: "root", RootUser: true}
			Expect(
				gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, rootUser),
			).To(Succeed())
			Expect(
				w.Delete(ctx, rootUser.Key),
			).To(MatchError(ContainSubstring("cannot delete root user")))
		})
		It("Should be a no-op when the user does not exist", func(ctx SpecContext) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
		It(
			"Should delete existing users and ignore unknown keys in the same call",
			func(ctx SpecContext) {
				existing := MustSucceed(
					w.Create(ctx, user.User{Name: "a", Role: member}),
				)
				Expect(w.Delete(ctx, existing.Key, uuid.New())).To(Succeed())
				Expect(exists(ctx, existing.Key)).To(BeFalse())
			},
		)
	})
})
