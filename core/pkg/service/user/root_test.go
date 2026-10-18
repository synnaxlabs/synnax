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
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

// provision runs root provisioning against the suite-level services.
func provision(ctx context.Context, username, pwd string) user.User {
	GinkgoHelper()
	return MustSucceed(user.ProvisionRoot(ctx, user.RootConfig{
		Users:       svc,
		Passwords:   passwordSvc,
		Credentials: auth.Credentials{Username: username, Password: pwd},
	}))
}

// login proves username and pwd against the suite-level password service.
func login(ctx context.Context, username, pwd string) (auth.Identity, error) {
	return passwordSvc.Authenticate(ctx, auth.Proof{
		Variant: auth.PasswordProof{
			Credentials: auth.Credentials{Username: username, Password: pwd},
		},
	})
}

// createUser stores a user record and its password. A root record goes through the raw
// gorp writer, because [user.Writer.Create] rejects RootUser=true.
func createUser(ctx context.Context, username, pwd string, root bool) user.User {
	GinkgoHelper()
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if root {
			u = user.User{Key: uuid.New(), Username: username, RootUser: true}
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
				Username: username,
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

func findUser(ctx context.Context, username string) user.User {
	GinkgoHelper()
	var u user.User
	Expect(svc.NewRetrieve().Where(user.MatchUsernames(username)).Entry(&u).
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

// purgeUsers deletes every user record and password from the suite-level db. It uses
// the raw gorp writer because [user.Writer.Delete] rejects root users by design.
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

var _ = Describe("RootConfig", func() {
	var cfg user.RootConfig
	BeforeEach(func() {
		cfg = user.RootConfig{
			Users:       svc,
			Passwords:   passwordSvc,
			Credentials: auth.Credentials{Username: "u", Password: "p"},
		}
	})
	Describe("Override", func() {
		It("Should take each field from the override when the base is zero", func() {
			result := user.RootConfig{}.Override(cfg)
			Expect(result).To(Equal(cfg))
		})
		It("Should keep each field from the base when the override is zero", func() {
			result := cfg.Override(user.RootConfig{})
			Expect(result).To(Equal(cfg))
		})
	})
	Describe("Validate", func() {
		It("Should succeed when every field is set", func() {
			Expect(cfg.Validate()).To(Succeed())
		})
		DescribeTable(
			"Should return an error when a field is missing",
			func(mutate func(*user.RootConfig), errorMsg string) {
				mutate(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(errorMsg)))
			},
			Entry(
				"users",
				func(c *user.RootConfig) { c.Users = nil },
				"users: must be non-nil",
			),
			Entry(
				"passwords",
				func(c *user.RootConfig) { c.Passwords = nil },
				"passwords: must be non-nil",
			),
			Entry(
				"username",
				func(c *user.RootConfig) { c.Credentials.Username = "" },
				"username: required",
			),
			Entry(
				"password",
				func(c *user.RootConfig) { c.Credentials.Password = "" },
				"password: required",
			),
		)
	})
})

var _ = Describe("ProvisionRoot", Serial, func() {
	BeforeEach(func(ctx SpecContext) { purgeUsers(ctx) })
	It("Should return an error when the config is invalid", func(ctx SpecContext) {
		Expect(user.ProvisionRoot(ctx, user.RootConfig{})).Error().
			To(MatchError(ContainSubstring("users: must be non-nil")))
	})
	It("Should create a root user on a fresh Core", func(ctx SpecContext) {
		root := provision(ctx, "alpha", "p1")
		Expect(root.RootUser).To(BeTrue())
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
		provision(ctx, "alpha", "p2")
		MustSucceed(login(ctx, "alpha", "p2"))
		Expect(login(ctx, "alpha", "p1")).Error().
			To(MatchError(auth.ErrInvalidCredentials))
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
	It("Should give a root record without a password one", func(ctx SpecContext) {
		orphan := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
			Username: "orphan",
		}))
		Expect(login(ctx, "orphan", "p")).Error().
			To(MatchError(auth.ErrInvalidCredentials))
		root := provision(ctx, "orphan", "p")
		Expect(root.Key).To(Equal(orphan.Key))
		MustSucceed(login(ctx, "orphan", "p"))
	})
})
