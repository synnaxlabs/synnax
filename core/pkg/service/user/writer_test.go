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
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
	Describe("Create", func() {
		It("Should create a new user with the given key", func(ctx SpecContext) {
			key := uuid.New()
			name := uuid.New().String()
			u := MustSucceed(w.Create(ctx, user.User{
				Username: name,
				Key:      key,
			}))
			Expect(u.Key).To(Equal(key))
			Expect(u.Username).To(Equal(name))
			Expect(u.FirstName).To(Equal(""))
			Expect(u.LastName).To(Equal(""))
			Expect(u.RootUser).To(BeFalse())
		})
		It(
			"Should assign a new key when none is provided on create",
			func(ctx SpecContext) {
				name := uuid.New().String()
				u := MustSucceed(w.Create(ctx, user.User{Username: name}))
				Expect(u.Key).ToNot(Equal(uuid.Nil()))
				Expect(u.Username).To(Equal(name))
				Expect(u.RootUser).To(BeFalse())
			},
		)
		It("Should create a user with profile metadata", func(ctx SpecContext) {
			name := uuid.New().String()
			u := MustSucceed(w.Create(ctx, user.User{
				Username:  name,
				FirstName: "Patrick",
				LastName:  "Star",
			}))
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
			Expect(u.Username).To(Equal(name))
		})
		It(
			"Should reject RootUser=true so the root-user invariant cannot be bypassed",
			func(ctx SpecContext) {
				Expect(w.Create(ctx, user.User{
					Username: uuid.New().String(),
					RootUser: true,
				})).Error().To(MatchError(ContainSubstring("cannot create a root user; root users are provisioned at startup")))
			},
		)
	})
	Describe("ChangeName", func() {
		It("Should change the names of a user", func(ctx SpecContext) {
			created := MustSucceed(
				w.Create(ctx, user.User{Username: uuid.New().String()}),
			)
			Expect(w.ChangeName(ctx, created.Key, "Patrick", "Star")).To(Succeed())
			var u user.User
			Expect(
				svc.NewRetrieve().
					Where(user.MatchKeys(created.Key)).
					Entry(&u).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
		})
		It("Should only change one name if the other is blank", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{
				Username:  uuid.New().String(),
				FirstName: "Original",
				LastName:  "Surname",
			}))
			Expect(w.ChangeName(ctx, created.Key, "Patrick", "")).To(Succeed())
			var u user.User
			Expect(
				svc.NewRetrieve().
					Where(user.MatchKeys(created.Key)).
					Entry(&u).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Surname"))
		})
	})
	Describe("Delete", func() {
		It("Should delete a single user", func(ctx SpecContext) {
			created := MustSucceed(
				w.Create(ctx, user.User{Username: uuid.New().String()}),
			)
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
			a := MustSucceed(w.Create(ctx, user.User{Username: uuid.New().String()}))
			b := MustSucceed(w.Create(ctx, user.User{Username: uuid.New().String()}))
			Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(
				svc.NewRetrieve().
					Where(user.MatchKeys(a.Key, b.Key)).
					Exists(ctx, tx),
			).To(BeFalse())
		})
		It("Should not delete a root user", func(ctx SpecContext) {
			// Insert a root user directly via the raw gorp writer because
			// Writer.Create rejects RootUser=true.
			rootUser := user.User{
				Key:      uuid.New(),
				Username: uuid.New().String(),
				RootUser: true,
			}
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
					w.Create(ctx, user.User{Username: uuid.New().String()}),
				)
				Expect(w.Delete(ctx, existing.Key, uuid.New())).To(Succeed())
				Expect(
					svc.NewRetrieve().
						Where(user.MatchKeys(existing.Key)).
						Exists(ctx, tx),
				).To(BeFalse())
			},
		)
	})
})
