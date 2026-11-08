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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func provision(ctx context.Context, username, pwd string) user.User {
	GinkgoHelper()
	root, _ := MustSucceed2(user.ProvisionRoot(
		ctx,
		svc,
		passwordSvc,
		auth.Credentials{Username: username, Password: pwd},
		owner,
	))
	return root
}

func storedHash(ctx context.Context, username string) []byte {
	GinkgoHelper()
	var p password.Password
	Expect(passwordSvc.NewRetrieve().
		Where(password.MatchUsernames(username)).
		Entry(&p).
		Exec(ctx, db)).To(Succeed())
	return p.Hash
}

func login(ctx context.Context, username, pwd string) (auth.Identity, error) {
	return passwordSvc.Authenticate(ctx, auth.Proof{
		Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: username, Password: pwd},
		},
	})
}

// createUser writes root records raw, because [user.Writer.Create] rejects them.
func createUser(ctx context.Context, username, pwd string, root bool) user.User {
	GinkgoHelper()
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if root {
			u = user.User{Key: uuid.New(), Name: username, RootUser: true, Role: owner}
			if err := gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, u); err != nil {
				return err
			}
		} else {
			var err error
			if u, err = svc.NewWriter(tx).Create(ctx, user.User{
				Name: username,
				Role: member,
			}); err != nil {
				return err
			}
		}
		return passwordSvc.NewWriter(tx).
			Set(ctx, u.ResourceID(), username, hash(passwordSvc, pwd))
	})).To(Succeed())
	return u
}

// findUser returns the user whose password holds username.
func findUser(ctx context.Context, username string) user.User {
	GinkgoHelper()
	var p password.Password
	Expect(passwordSvc.NewRetrieve().
		Where(password.MatchUsernames(username)).
		Entry(&p).
		Exec(ctx, db)).To(Succeed())
	key := MustSucceed(user.KeyFromResourceID(p.Subject))
	var u user.User
	Expect(svc.NewRetrieve().Where(user.MatchKeys(key)).Entry(&u).
		Exec(ctx, db)).To(Succeed())
	return u
}

func rootUsers(ctx context.Context) []user.User {
	GinkgoHelper()
	var roots []user.User
	Expect(svc.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, db)).To(Succeed())
	return roots
}

// purgeUsers deletes raw, because [user.Writer.Delete] rejects root users. A raw delete
// follows no cascade, so it deletes the passwords too.
func purgeUsers(ctx context.Context) {
	GinkgoHelper()
	var users []user.User
	Expect(svc.NewRetrieve().Entries(&users).Exec(ctx, db)).To(Succeed())
	keys := make([]user.Key, len(users))
	for i, u := range users {
		keys[i] = u.Key
	}
	Expect(gorp.WrapWriter[user.Key, user.User](db).Delete(ctx, keys...)).
		To(Succeed())
	var passwords []password.Password
	Expect(passwordSvc.NewRetrieve().Entries(&passwords).Exec(ctx, db)).To(Succeed())
	pwKeys := make([]password.Key, len(passwords))
	for i, p := range passwords {
		pwKeys[i] = p.Key
	}
	Expect(gorp.WrapWriter[password.Key, password.Password](db).Delete(ctx, pwKeys...)).
		To(Succeed())
}

var _ = Describe("ProvisionRoot", Serial, func() {
	BeforeEach(func(ctx SpecContext) { purgeUsers(ctx) })
	It("Should reject incomplete credentials", func(ctx SpecContext) {
		Expect(user.ProvisionRoot(ctx, svc, passwordSvc, auth.Credentials{}, owner)).
			Error().
			To(MatchError(ContainSubstring("username: required")))
	})
	It("Should create a root user on a fresh Core", func(ctx SpecContext) {
		root := provision(ctx, "alpha", "p1")
		Expect(root.RootUser).To(BeTrue())
		Expect(root.Name).To(Equal("alpha"))
		Expect(root.Role).To(Equal(owner))
		Expect(findUser(ctx, "alpha")).To(Equal(root))
		Expect(MustSucceed(login(ctx, "alpha", "p1")).Subject).
			To(Equal(root.ResourceID()))
	})
	It("Should keep a root user whose username matches", func(ctx SpecContext) {
		before := provision(ctx, "alpha", "p1")
		after := provision(ctx, "alpha", "p1")
		Expect(after.Key).To(Equal(before.Key))
		Expect(rootUsers(ctx)).To(HaveLen(1))
	})
	It("Should set the root password from the settings", func(ctx SpecContext) {
		provision(ctx, "alpha", "p1")
		_, changed := MustSucceed2(user.ProvisionRoot(
			ctx,
			svc,
			passwordSvc,
			auth.Credentials{Username: "alpha", Password: "p2"},
			owner,
		))
		Expect(changed).To(BeTrue())
		MustSucceed(login(ctx, "alpha", "p2"))
		Expect(login(ctx, "alpha", "p1")).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should keep the stored hash when the password matches", func(ctx SpecContext) {
		_, changed := MustSucceed2(user.ProvisionRoot(
			ctx,
			svc,
			passwordSvc,
			auth.Credentials{Username: "alpha", Password: "p1"},
			owner,
		))
		Expect(changed).To(BeTrue())
		before := storedHash(ctx, "alpha")
		_, changed = MustSucceed2(user.ProvisionRoot(
			ctx,
			svc,
			passwordSvc,
			auth.Credentials{Username: "alpha", Password: "p1"},
			owner,
		))
		Expect(changed).To(BeFalse())
		Expect(storedHash(ctx, "alpha")).To(Equal(before))
	})
	It(
		"Should demote the previous root when the username changes",
		func(ctx SpecContext) {
			provision(ctx, "alpha", "p1")
			provision(ctx, "beta", "p2")
			Expect(findUser(ctx, "alpha").RootUser).To(BeFalse())
			Expect(findUser(ctx, "alpha").Role).To(Equal(owner))
			Expect(findUser(ctx, "beta").RootUser).To(BeTrue())
			Expect(rootUsers(ctx)).To(HaveLen(1))
			MustSucceed(login(ctx, "alpha", "p1"))
			MustSucceed(login(ctx, "beta", "p2"))
		},
	)
	It("Should promote a user that holds the username", func(ctx SpecContext) {
		provision(ctx, "old-root", "p")
		before := createUser(ctx, "gamma", "x", false)
		after := provision(ctx, "gamma", "p3")
		Expect(after.Key).To(Equal(before.Key))
		Expect(after.RootUser).To(BeTrue())
		Expect(after.Role).To(Equal(owner))
		Expect(findUser(ctx, "gamma").Role).To(Equal(owner))
		Expect(findUser(ctx, "old-root").RootUser).To(BeFalse())
		Expect(rootUsers(ctx)).To(HaveLen(1))
		MustSucceed(login(ctx, "gamma", "p3"))
		Expect(login(ctx, "gamma", "x")).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should demote every stale root", func(ctx SpecContext) {
		provision(ctx, "root-bootstrap", "p")
		createUser(ctx, "stale1", "x", true)
		createUser(ctx, "stale2", "x", true)
		Expect(rootUsers(ctx)).To(HaveLen(3))
		provision(ctx, "fresh", "p4")
		Expect(findUser(ctx, "stale1").RootUser).To(BeFalse())
		Expect(findUser(ctx, "stale2").RootUser).To(BeFalse())
		Expect(findUser(ctx, "fresh").RootUser).To(BeTrue())
		Expect(rootUsers(ctx)).To(HaveLen(1))
	})
	It("Should promote the owner of the password, not a user with the name", func(
		ctx SpecContext,
	) {
		provision(ctx, "old-root", "p")
		var decoy, owner user.User
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			decoy, err = svc.NewWriter(tx).
				Create(ctx, user.User{Name: "delta", Role: member})
			return err
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			owner, err = svc.NewWriter(tx).
				Create(ctx, user.User{Name: "owner", Role: member})
			return err
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return passwordSvc.NewWriter(tx).
				Set(ctx, owner.ResourceID(), "delta", hash(passwordSvc, "p"))
		})).To(Succeed())
		root := provision(ctx, "delta", "p")
		Expect(root.Key).To(Equal(owner.Key))
		Expect(root.Key).ToNot(Equal(decoy.Key))
	})
	It("Should create a root user when no password holds the username", func(
		ctx SpecContext,
	) {
		var other user.User
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			other, err = svc.NewWriter(tx).
				Create(ctx, user.User{Name: "orphan", Role: member})
			return err
		})).To(Succeed())
		root := provision(ctx, "orphan", "p")
		Expect(root.Key).ToNot(Equal(other.Key))
		Expect(MustSucceed(login(ctx, "orphan", "p")).Subject).
			To(Equal(root.ResourceID()))
	})
})
