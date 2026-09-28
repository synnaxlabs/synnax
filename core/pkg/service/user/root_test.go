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

// openRootUser opens a [user.Service] against the suite-level db using the given root
// credentials. Pass an empty username to open without root credentials.
func openRootUser(ctx context.Context, username, pwd string) *user.Service {
	GinkgoHelper()
	cfg := user.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc, Search: searchIdx, Auth: authSvc,
	}
	if username != "" {
		cfg.RootCredentials = auth.Credentials{Username: username, Password: pwd}
	}
	return MustOpen(user.OpenService(ctx, cfg))
}

// createUser registers a user record and matching credentials in a single transaction.
// Non-root users go through the public Writer.Create; root users are inserted with
// the raw gorp writer because [user.Writer.Create] now rejects RootUser=true. The
// raw insert mirrors what the reconciler does internally.
func createUser(
	ctx context.Context,
	svc *user.Service,
	username, password string,
	root bool,
) user.User {
	GinkgoHelper()
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if !root {
			var err error
			if u, err = svc.NewWriter(tx).
				Create(ctx, user.User{Username: username}); err != nil {
				return err
			}
			return authSvc.NewWriter(tx).Register(ctx, u.Key, auth.Credentials{
				Username: username,
				Password: password,
			})
		}
		u = user.User{Key: uuid.New(), Username: username, RootUser: true}
		if err := gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, u); err != nil {
			return err
		}
		if err := otg.NewWriter(tx).DefineResources(ctx, u.OntologyID()); err != nil {
			return err
		}
		return authSvc.NewWriter(tx).Register(ctx, u.Key, auth.Credentials{
			Username: username,
			Password: password,
		})
	})).To(Succeed())
	return u
}

// createRootRecordOnly creates a root user record without credentials. The raw gorp
// writer is used because [user.Writer.Create] rejects RootUser=true.
func createRootRecordOnly(ctx context.Context) user.User {
	GinkgoHelper()
	u := user.User{Key: uuid.New(), RootUser: true}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := gorp.WrapWriter[user.Key, user.User](tx).Set(ctx, u); err != nil {
			return err
		}
		return otg.NewWriter(tx).DefineResources(ctx, u.OntologyID())
	})).To(Succeed())
	return u
}

// authenticate checks the given credentials against the auth service.
func authenticate(ctx context.Context, username, password string) error {
	GinkgoHelper()
	_, err := authSvc.Authenticate(ctx, nil, auth.Credentials{
		Username: username,
		Password: password,
	})
	return err
}

// findUser retrieves the user whose credentials hold username and fails the spec if no
// such user exists.
func findUser(ctx context.Context, svc *user.Service, username string) user.User {
	GinkgoHelper()
	keys := MustSucceed(authSvc.KeysByUsername(ctx, nil, username))
	Expect(keys).To(HaveLen(1))
	u := []user.User{{}}
	Expect(svc.NewRetrieve().Where(user.MatchKeys(keys[0])).Entry(&u[0]).
		Exec(ctx, nil)).To(Succeed())
	Expect(svc.ResolveUsernames(ctx, nil, u)).To(Succeed())
	return u[0]
}

func rootUsers(ctx context.Context, svc *user.Service) []user.User {
	GinkgoHelper()
	var roots []user.User
	Expect(svc.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, nil)).To(Succeed())
	return roots
}

// purgeUsersAndAuth deletes every user record and its credentials from the
// suite-level db, restoring a clean slate between specs. Uses the raw gorp writer and
// the ontology writer directly so it can delete root users — the public
// [user.Writer.Delete] rejects them by design.
func purgeUsersAndAuth(ctx context.Context) {
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
		return authSvc.NewWriter(tx).Deactivate(ctx, keys...)
	})).To(Succeed())
}

var _ = Describe("Root user reconciliation", Serial, func() {
	BeforeEach(func(ctx SpecContext) { purgeUsersAndAuth(ctx) })
	Describe("Bootstrap (no existing state)", func() {
		It("Should create a root user on a fresh cluster", func(ctx SpecContext) {
			s := openRootUser(ctx, "alpha", "p1")
			Expect(findUser(ctx, s, "alpha").RootUser).To(BeTrue())
			Expect(authenticate(ctx, "alpha", "p1")).To(Succeed())
		})
	})
	Describe("Matching root user", func() {
		It(
			"Should be a no-op when the existing root user matches config",
			func(ctx SpecContext) {
				svc1 := openRootUser(ctx, "alpha", "p1")
				before := findUser(ctx, svc1, "alpha")
				Expect(svc1.Close()).To(Succeed())
				svc2 := openRootUser(ctx, "alpha", "p1")
				after := findUser(ctx, svc2, "alpha")
				Expect(after.Key).To(Equal(before.Key))
				Expect(after.RootUser).To(BeTrue())
				Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
			},
		)
		It(
			"Should rotate the root password when config provides a different password",
			func(ctx SpecContext) {
				svc1 := openRootUser(ctx, "alpha", "p1")
				Expect(authenticate(ctx, "alpha", "p1")).To(Succeed())
				Expect(svc1.Close()).To(Succeed())
				svc2 := openRootUser(ctx, "alpha", "p2")
				Expect(authenticate(ctx, "alpha", "p2")).To(Succeed())
				Expect(
					authenticate(ctx, "alpha", "p1"),
				).Error().
					To(MatchError(auth.ErrInvalidCredentials))
				Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
			},
		)
	})
	Describe("Username change", func() {
		It(
			"Should demote the previous root and create a new root when the config username changes",
			func(ctx SpecContext) {
				svc1 := openRootUser(ctx, "alpha", "p1")
				Expect(svc1.Close()).To(Succeed())
				svc2 := openRootUser(ctx, "beta", "p2")
				Expect(findUser(ctx, svc2, "alpha").RootUser).To(BeFalse())
				Expect(findUser(ctx, svc2, "beta").RootUser).To(BeTrue())
				Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
				Expect(authenticate(ctx, "alpha", "p1")).To(Succeed())
				Expect(authenticate(ctx, "beta", "p2")).To(Succeed())
			},
		)
		It("Should be idempotent after a demotion+recreate", func(ctx SpecContext) {
			Expect(openRootUser(ctx, "alpha", "p1").Close()).To(Succeed())
			Expect(openRootUser(ctx, "beta", "p2").Close()).To(Succeed())
			svc3 := openRootUser(ctx, "beta", "p2")
			Expect(findUser(ctx, svc3, "alpha").RootUser).To(BeFalse())
			Expect(findUser(ctx, svc3, "beta").RootUser).To(BeTrue())
			Expect(rootUsers(ctx, svc3)).To(HaveLen(1))
		})
	})
	Describe("Promotion of an existing non-root user", func() {
		It(
			"Should promote the existing user, rotate their password, and demote previous roots",
			func(ctx SpecContext) {
				seedSvc := openRootUser(ctx, "old-root", "p")
				createUser(ctx, seedSvc, "gamma", "x", false)
				gammaBefore := findUser(ctx, seedSvc, "gamma")
				Expect(gammaBefore.RootUser).To(BeFalse())
				Expect(seedSvc.Close()).To(Succeed())

				s := openRootUser(ctx, "gamma", "p3")
				gammaAfter := findUser(ctx, s, "gamma")
				Expect(gammaAfter.Key).To(Equal(gammaBefore.Key))
				Expect(gammaAfter.RootUser).To(BeTrue())
				// Previous root demoted.
				Expect(findUser(ctx, s, "old-root").RootUser).To(BeFalse())
				Expect(rootUsers(ctx, s)).To(HaveLen(1))
				// New password is active; the old one no longer authenticates.
				Expect(authenticate(ctx, "gamma", "p3")).To(Succeed())
				Expect(
					authenticate(ctx, "gamma", "x"),
				).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should promote and keep the same password when config matches the existing password",
			func(ctx SpecContext) {
				seedSvc := DeferClose(openRootUser(ctx, "old-root", "p"))
				createUser(ctx, seedSvc, "gamma", "same-pwd", false)
				gammaBefore := findUser(ctx, seedSvc, "gamma")

				s := openRootUser(ctx, "gamma", "same-pwd")
				gammaAfter := findUser(ctx, s, "gamma")
				Expect(gammaAfter.Key).To(Equal(gammaBefore.Key))
				Expect(gammaAfter.RootUser).To(BeTrue())
				Expect(rootUsers(ctx, s)).To(HaveLen(1))
				Expect(authenticate(ctx, "gamma", "same-pwd")).To(Succeed())
			},
		)
	})
	Describe("Stale root users", func() {
		It(
			"Should collapse multiple stale roots when new credentials are configured",
			func(ctx SpecContext) {
				seedSvc := openRootUser(ctx, "root-bootstrap", "p")
				createUser(ctx, seedSvc, "stale1", "x", true)
				createUser(ctx, seedSvc, "stale2", "x", true)
				Expect(seedSvc.Close()).To(Succeed())
				s := openRootUser(ctx, "fresh", "p4")
				Expect(findUser(ctx, s, "stale1").RootUser).To(BeFalse())
				Expect(findUser(ctx, s, "stale2").RootUser).To(BeFalse())
				Expect(findUser(ctx, s, "fresh").RootUser).To(BeTrue())
				Expect(rootUsers(ctx, s)).To(HaveLen(1))
			},
		)
		It(
			"Should collapse multiple stale roots without credentials, retaining exactly one",
			func(ctx SpecContext) {
				seedSvc := openRootUser(ctx, "root-bootstrap", "p")
				createUser(ctx, seedSvc, "charlie", "x", true)
				createUser(ctx, seedSvc, "alpha", "x", true)
				createUser(ctx, seedSvc, "beta", "x", true)
				Expect(rootUsers(ctx, seedSvc)).To(HaveLen(4))
				Expect(seedSvc.Close()).To(Succeed())
				s := openRootUser(ctx, "", "")
				Expect(rootUsers(ctx, s)).To(HaveLen(1))
			},
		)
		It(
			"Should leave a single existing root alone when no credentials are configured",
			func(ctx SpecContext) {
				seedSvc := openRootUser(ctx, "loner", "p")
				lonerBefore := findUser(ctx, seedSvc, "loner")
				Expect(seedSvc.Close()).To(Succeed())
				s := openRootUser(ctx, "", "")
				loner := findUser(ctx, s, "loner")
				Expect(loner.Key).To(Equal(lonerBefore.Key))
				Expect(loner.RootUser).To(BeTrue())
				Expect(rootUsers(ctx, s)).To(HaveLen(1))
			},
		)
		It(
			"Should open cleanly when no roots exist and no credentials are configured",
			func(ctx SpecContext) {
				s := MustOpen(user.OpenService(ctx, user.ServiceConfig{
					DB:       db,
					Ontology: otg,
					Group:    groupSvc,
					Search:   searchIdx,
					Auth:     authSvc,
				}))
				Expect(rootUsers(ctx, s)).To(BeEmpty())
			},
		)
	})
	Describe("Orphan state recovery", func() {
		It(
			"Should demote a root user record without credentials and create a new root",
			func(ctx SpecContext) {
				seedSvc := openRootUser(ctx, "root-bootstrap", "p")
				orphan := createRootRecordOnly(ctx)
				Expect(seedSvc.Close()).To(Succeed())
				s := openRootUser(ctx, "fresh-root", "p")
				Expect(authenticate(ctx, "fresh-root", "p")).To(Succeed())
				fresh := findUser(ctx, s, "fresh-root")
				Expect(fresh.Key).ToNot(Equal(orphan.Key))
				Expect(fresh.RootUser).To(BeTrue())
				Expect(rootUsers(ctx, s)).To(ConsistOf(HaveField("Key", fresh.Key)))
			},
		)
	})
})
