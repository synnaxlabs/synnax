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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	))
	return root
}

func storedHash(ctx context.Context, username string) []byte {
	GinkgoHelper()
	var p password.Password
	Expect(passwordSvc.NewRetrieve().
		Where(password.MatchUsernames(username)).
		Entry(&p).
		Exec(ctx, nil)).To(Succeed())
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
			u = user.User{Key: uuid.New(), Name: username, RootUser: true}
			if err := gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, u); err != nil {
				return err
			}
			if err := otg.NewWriter(tx).
				DefineResources(ctx, u.OntologyID()); err != nil {
				return err
			}
		} else {
			var err error
			if u, err = svc.NewWriter(tx).Create(ctx, user.User{
				Name: username,
			}); err != nil {
				return err
			}
		}
		return passwordSvc.NewWriter(tx).Set(ctx, u.OntologyID(), auth.Credentials{
			Username: username,
			Password: pwd,
		})
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
		Exec(ctx, nil)).To(Succeed())
	key := MustSucceed(user.KeyFromOntologyID(p.Subject))
	var u user.User
	Expect(svc.NewRetrieve().Where(user.MatchKeys(key)).Entry(&u).
		Exec(ctx, nil)).To(Succeed())
	return u
}

func rootUsers(ctx context.Context) []user.User {
	GinkgoHelper()
	var roots []user.User
	Expect(svc.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, nil)).To(Succeed())
	return roots
}

// purgeUsers deletes raw, because [user.Writer.Delete] rejects root users.
func purgeUsers(ctx context.Context) {
	GinkgoHelper()
	var users []user.User
	Expect(svc.NewRetrieve().Entries(&users).Exec(ctx, nil)).To(Succeed())
	if len(users) == 0 {
		return
	}
	keys := make([]user.Key, len(users))
	for i, u := range users {
		keys[i] = u.Key
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := gorp.WrapWriter[user.Key, user.User](
			tx,
		).Delete(ctx, keys...); err != nil {
			return err
		}
		if err := otg.NewWriter(tx).DeleteResources(
			ctx, user.OntologyIDsFromKeys(keys)...,
		); err != nil {
			return err
		}
		return passwordSvc.NewWriter(tx).
			DeleteFor(ctx, user.OntologyIDsFromKeys(keys)...)
	})).To(Succeed())
}

var _ = Describe("ProvisionRoot", Serial, func() {
	BeforeEach(func(ctx SpecContext) { purgeUsers(ctx) })
	It("Should reject incomplete credentials", func(ctx SpecContext) {
		Expect(user.ProvisionRoot(ctx, svc, passwordSvc, auth.Credentials{})).Error().
			To(MatchError(ContainSubstring("username: required")))
	})
	It("Should create a root user on a fresh Core", func(ctx SpecContext) {
		root := provision(ctx, "alpha", "p1")
		Expect(root.RootUser).To(BeTrue())
		Expect(root.Name).To(Equal("alpha"))
		Expect(findUser(ctx, "alpha")).To(Equal(root))
		Expect(MustSucceed(login(ctx, "alpha", "p1")).Subject).
			To(Equal(root.OntologyID()))
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
		))
		Expect(changed).To(BeTrue())
		before := storedHash(ctx, "alpha")
		_, changed = MustSucceed2(user.ProvisionRoot(
			ctx,
			svc,
			passwordSvc,
			auth.Credentials{Username: "alpha", Password: "p1"},
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
		decoy := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{Name: "delta"}))
		owner := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{Name: "owner"}))
		Expect(passwordSvc.NewWriter(nil).Set(ctx, owner.OntologyID(), auth.Credentials{
			Username: "delta",
			Password: "p",
		})).To(Succeed())
		root := provision(ctx, "delta", "p")
		Expect(root.Key).To(Equal(owner.Key))
		Expect(root.Key).ToNot(Equal(decoy.Key))
	})
	It("Should create a root user when no password holds the username", func(
		ctx SpecContext,
	) {
		other := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{Name: "orphan"}))
		root := provision(ctx, "orphan", "p")
		Expect(root.Key).ToNot(Equal(other.Key))
		Expect(MustSucceed(login(ctx, "orphan", "p")).Subject).
			To(Equal(root.OntologyID()))
	})
	It("Should reject a username whose password is not held by a user", func(
		ctx SpecContext,
	) {
		Expect(passwordSvc.NewWriter(nil).Set(
			ctx,
			ontology.ID{Type: "rack", Key: "1"},
			auth.Credentials{Username: "rack-held", Password: "p"},
		)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(passwordSvc.NewWriter(nil).DeleteFor(
				ctx,
				ontology.ID{Type: "rack", Key: "1"},
			)).To(Succeed())
		})
		Expect(user.ProvisionRoot(ctx, svc, passwordSvc, auth.Credentials{
			Username: "rack-held",
			Password: "p",
		})).Error().To(MatchError(ContainSubstring("not held by a user")))
	})
})
