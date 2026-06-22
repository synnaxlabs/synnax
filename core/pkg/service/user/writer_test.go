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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
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
			name := uuid.NewString()
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
		It("Should assign a new key when none is provided on create", func(ctx SpecContext) {
			name := uuid.NewString()
			u := MustSucceed(w.Create(ctx, user.User{Username: name}))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal(name))
			Expect(u.RootUser).To(BeFalse())
		})
		It("Should create a user with profile metadata", func(ctx SpecContext) {
			name := uuid.NewString()
			u := MustSucceed(w.Create(ctx, user.User{
				Username:  name,
				FirstName: "Patrick",
				LastName:  "Star",
			}))
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
			Expect(u.Username).To(Equal(name))
		})
		It("Should return ErrRepeatedUsername when the username already exists", func(ctx SpecContext) {
			name := uuid.NewString()
			u := MustSucceed(w.Create(ctx, user.User{Username: name}))
			Expect(u.Key).ToNot(Equal(uuid.Nil))
			Expect(u.Username).To(Equal(name))
			Expect(w.Create(ctx, user.User{Username: name})).
				Error().To(MatchError(auth.ErrRepeatedUsername))
		})
		It("Should return a validation error when the username is empty", func(ctx SpecContext) {
			Expect(w.Create(ctx, user.User{})).Error().
				To(MatchError(ContainSubstring("username: required")))
		})
		It("Should reject RootUser=true so the root-user invariant cannot be bypassed", func(ctx SpecContext) {
			Expect(w.Create(ctx, user.User{
				Username: uuid.NewString(),
				RootUser: true,
			})).Error().To(MatchError(ContainSubstring("cannot create a root user; root users are provisioned at startup")))
		})
	})
	Describe("ChangeUsername", func() {
		It("Should change the username of a user", func(ctx SpecContext) {
			original := uuid.NewString()
			updated := uuid.NewString()
			created := MustSucceed(w.Create(ctx, user.User{Username: original}))
			Expect(w.ChangeUsername(ctx, created.Key, updated)).To(Succeed())
			Expect(svc.NewRetrieve().Where(user.MatchUsernames(updated)).Exists(ctx, tx)).To(BeTrue())
			Expect(svc.NewRetrieve().Where(user.MatchUsernames(original)).Exists(ctx, tx)).To(BeFalse())
		})
		It("Should return ErrRepeatedUsername if the username already exists", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			b := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			Expect(w.ChangeUsername(ctx, a.Key, b.Username)).To(MatchError(auth.ErrRepeatedUsername))
		})
	})
	Describe("ChangeName", func() {
		It("Should change the names of a user", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			Expect(w.ChangeName(ctx, created.Key, "Patrick", "Star")).To(Succeed())
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).Exec(ctx, tx)).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Star"))
		})
		It("Should only change one name if the other is blank", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{
				Username:  uuid.NewString(),
				FirstName: "Original",
				LastName:  "Surname",
			}))
			Expect(w.ChangeName(ctx, created.Key, "Patrick", "")).To(Succeed())
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).Exec(ctx, tx)).To(Succeed())
			Expect(u.FirstName).To(Equal("Patrick"))
			Expect(u.LastName).To(Equal("Surname"))
		})
	})
	Describe("Delete", func() {
		It("Should delete a single user", func(ctx SpecContext) {
			created := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			Expect(w.Delete(ctx, created.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(user.MatchUsernames(created.Username)).Exists(ctx, tx)).To(BeFalse())
			var u user.User
			Expect(svc.NewRetrieve().Where(user.MatchKeys(created.Key)).Entry(&u).Exec(ctx, tx)).
				To(MatchError(query.ErrNotFound))
		})
		It("Should delete multiple users", func(ctx SpecContext) {
			a := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			b := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			Expect(w.Delete(ctx, a.Key, b.Key)).To(Succeed())
			Expect(svc.NewRetrieve().Where(user.MatchUsernames(a.Username)).Exists(ctx, nil)).To(BeFalse())
			Expect(svc.NewRetrieve().Where(user.MatchUsernames(b.Username)).Exists(ctx, nil)).To(BeFalse())
		})
		It("Should not delete a root user", func(ctx SpecContext) {
			// Insert a root user directly via the raw gorp writer because
			// Writer.Create rejects RootUser=true.
			rootUser := user.User{
				Key:      uuid.New(),
				Username: uuid.NewString(),
				RootUser: true,
			}
			Expect(gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, rootUser)).To(Succeed())
			Expect(w.Delete(ctx, rootUser.Key)).To(MatchError(ContainSubstring("cannot delete root user")))
		})
		It("Should be a no-op when the user does not exist", func(ctx SpecContext) {
			Expect(w.Delete(ctx, uuid.New())).To(Succeed())
		})
		It("Should delete existing users and ignore unknown keys in the same call", func(ctx SpecContext) {
			existing := MustSucceed(w.Create(ctx, user.User{Username: uuid.NewString()}))
			Expect(w.Delete(ctx, existing.Key, uuid.New())).To(Succeed())
			Expect(svc.NewRetrieve().Where(user.MatchKeys(existing.Key)).Exists(ctx, tx)).To(BeFalse())
		})
	})
})
