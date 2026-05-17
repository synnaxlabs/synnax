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
	cfg := user.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc, Search: searchIdx, Auth: authSvc,
	}
	if username != "" {
		cfg.RootCredentials = auth.Credentials{Username: username, Password: pwd}
	}
	return MustOpen(user.OpenService(ctx, cfg))
}

// seedUser registers a user record and matching credentials in a single transaction,
// mirroring the API-layer create flow.
func seedUser(
	ctx context.Context,
	svc *user.Service,
	username, password string,
	root bool,
) user.User {
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := authSvc.NewWriter(tx).Register(ctx, auth.Credentials{
			Username: username, Password: password,
		}); err != nil {
			return err
		}
		var err error
		u, err = svc.NewWriter(tx).Create(ctx, user.User{
			Username: username, RootUser: root,
		})
		return err
	})).To(Succeed())
	return u
}

// seedUserRecordOnly creates a user record without registering credentials, simulating
// an orphan record (RootUser=true, no auth row) that the reconciler must heal by
// registering creds.
func seedUserRecordOnly(
	ctx context.Context, svc *user.Service, username string, root bool,
) user.User {
	return MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
		Username: username, RootUser: root,
	}))
}

// seedAuthRowOnly registers credentials without creating a user record, simulating an
// orphan auth row that the reconciler must heal by creating the corresponding user
// record.
func seedAuthRowOnly(ctx context.Context, username, password string) {
	Expect(authSvc.NewWriter(nil).Register(ctx, auth.Credentials{
		Username: username, Password: password,
	})).To(Succeed())
}

// findUser retrieves the user with the given username and fails the spec if no such
// user exists.
func findUser(ctx context.Context, svc *user.Service, username string) user.User {
	var u user.User
	Expect(svc.NewRetrieve().Where(user.MatchUsernames(username)).Entry(&u).
		Exec(ctx, nil)).To(Succeed())
	return u
}

func rootUsers(ctx context.Context, svc *user.Service) []user.User {
	var roots []user.User
	Expect(svc.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entries(&roots).
		Exec(ctx, nil)).To(Succeed())
	return roots
}

// purgeUsersAndAuth deletes every user record and every auth credential from the
// suite-level db, restoring a clean slate between specs. The root-user guard on
// [user.Writer.Delete] is bypassed by first clearing the RootUser flag on every user.
func purgeUsersAndAuth(ctx context.Context) {
	var users []user.User
	Expect(svc.NewRetrieve().Entries(&users).Exec(ctx, nil)).To(Succeed())
	keys := make([]user.Key, len(users))
	usernames := make([]string, len(users))
	for i, u := range users {
		keys[i] = u.Key
		usernames[i] = u.Username
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		w := svc.NewWriter(tx)
		for _, k := range keys {
			if err := w.SetRootUser(ctx, k, false); err != nil {
				return err
			}
		}
		if len(keys) > 0 {
			if err := w.Delete(ctx, keys...); err != nil {
				return err
			}
		}
		if len(usernames) > 0 {
			return authSvc.NewWriter(tx).Deactivate(ctx, usernames...)
		}
		return nil
	})).To(Succeed())
}

var _ = Describe("Root user reconciliation", Serial, func() {
	AfterEach(func(ctx SpecContext) { purgeUsersAndAuth(ctx) })
	Describe("Bootstrap (no existing state)", func() {
		It("Should create a root user on a fresh cluster", func(ctx SpecContext) {
			s := openRootUser(ctx, "alpha", "p1")
			Expect(findUser(ctx, s, "alpha").RootUser).To(BeTrue())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "alpha", Password: "p1",
			})).To(Succeed())
		})
	})
	Describe("Matching root user", func() {
		It("Should be a no-op when the existing root user matches config", func(ctx SpecContext) {
			svc1 := openRootUser(ctx, "alpha", "p1")
			before := findUser(ctx, svc1, "alpha")
			Expect(svc1.Close()).To(Succeed())
			svc2 := openRootUser(ctx, "alpha", "p1")
			after := findUser(ctx, svc2, "alpha")
			Expect(after.Key).To(Equal(before.Key))
			Expect(after.RootUser).To(BeTrue())
			Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
		})
		It("Should rotate the root password when config provides a different password", func(ctx SpecContext) {
			svc1 := openRootUser(ctx, "alpha", "p1")
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "alpha", Password: "p1",
			})).To(Succeed())
			Expect(svc1.Close()).To(Succeed())
			svc2 := openRootUser(ctx, "alpha", "p2")
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "alpha", Password: "p2",
			})).To(Succeed())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "alpha", Password: "p1",
			})).Error().To(MatchError(auth.ErrInvalidCredentials))
			Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
		})
	})
	Describe("Username change", func() {
		It("Should demote the previous root and create a new root when the config username changes", func(ctx SpecContext) {
			svc1 := openRootUser(ctx, "alpha", "p1")
			Expect(svc1.Close()).To(Succeed())
			svc2 := openRootUser(ctx, "beta", "p2")
			Expect(findUser(ctx, svc2, "alpha").RootUser).To(BeFalse())
			Expect(findUser(ctx, svc2, "beta").RootUser).To(BeTrue())
			Expect(rootUsers(ctx, svc2)).To(HaveLen(1))
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "alpha", Password: "p1",
			})).To(Succeed())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "beta", Password: "p2",
			})).To(Succeed())
		})
		It("Should be idempotent after a demotion+recreate", func(ctx SpecContext) {
			Expect(openRootUser(ctx, "alpha", "p1").Close()).To(Succeed())
			Expect(openRootUser(ctx, "beta", "p2").Close()).To(Succeed())
			svc3 := openRootUser(ctx, "beta", "p2")
			Expect(findUser(ctx, svc3, "alpha").RootUser).To(BeFalse())
			Expect(findUser(ctx, svc3, "beta").RootUser).To(BeTrue())
			Expect(rootUsers(ctx, svc3)).To(HaveLen(1))
		})
	})
	Describe("Non-root collision", func() {
		It("Should refuse to start when a non-root user already owns the configured username", func(ctx SpecContext) {
			seedSvc := openRootUser(ctx, "root-bootstrap", "p")
			_ = seedUser(ctx, seedSvc, "gamma", "x", false)
			Expect(seedSvc.Close()).To(Succeed())
			Expect(user.OpenService(ctx, user.ServiceConfig{
				DB: db, Ontology: otg, Group: groupSvc, Search: searchIdx,
				Auth:            authSvc,
				RootCredentials: auth.Credentials{Username: "gamma", Password: "p3"},
			})).Error().To(MatchError(ContainSubstring("non-root user with username")))
			// Ensure the existing creds were not touched.
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "gamma", Password: "x",
			})).To(Succeed())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "gamma", Password: "p3",
			})).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
	})
	Describe("Stale root users", func() {
		It("Should collapse multiple stale roots when new credentials are configured", func(ctx SpecContext) {
			seedSvc := openRootUser(ctx, "root-bootstrap", "p")
			seedUser(ctx, seedSvc, "stale1", "x", true)
			seedUser(ctx, seedSvc, "stale2", "x", true)
			Expect(seedSvc.Close()).To(Succeed())
			s := openRootUser(ctx, "fresh", "p4")
			Expect(findUser(ctx, s, "stale1").RootUser).To(BeFalse())
			Expect(findUser(ctx, s, "stale2").RootUser).To(BeFalse())
			Expect(findUser(ctx, s, "fresh").RootUser).To(BeTrue())
			Expect(rootUsers(ctx, s)).To(HaveLen(1))
		})
		It("Should collapse multiple stale roots without credentials, retaining exactly one", func(ctx SpecContext) {
			seedSvc := openRootUser(ctx, "root-bootstrap", "p")
			seedUser(ctx, seedSvc, "charlie", "x", true)
			seedUser(ctx, seedSvc, "alpha", "x", true)
			seedUser(ctx, seedSvc, "beta", "x", true)
			Expect(rootUsers(ctx, seedSvc)).To(HaveLen(4))
			Expect(seedSvc.Close()).To(Succeed())
			s := openRootUser(ctx, "", "")
			Expect(rootUsers(ctx, s)).To(HaveLen(1))
		})
		It("Should leave a single existing root alone when no credentials are configured", func(ctx SpecContext) {
			seedSvc := openRootUser(ctx, "loner", "p")
			lonerBefore := findUser(ctx, seedSvc, "loner")
			Expect(seedSvc.Close()).To(Succeed())
			s := openRootUser(ctx, "", "")
			loner := findUser(ctx, s, "loner")
			Expect(loner.Key).To(Equal(lonerBefore.Key))
			Expect(loner.RootUser).To(BeTrue())
			Expect(rootUsers(ctx, s)).To(HaveLen(1))
		})
		It("Should open cleanly when no roots exist and no credentials are configured", func(ctx SpecContext) {
			s := MustOpen(user.OpenService(ctx, user.ServiceConfig{
				DB: db, Ontology: otg, Group: groupSvc, Search: searchIdx,
			}))
			Expect(rootUsers(ctx, s)).To(BeEmpty())
		})
	})
	Describe("Orphan state recovery", func() {
		It("Should register credentials when a root user record exists without an auth row", func(ctx SpecContext) {
			seedSvc := openRootUser(ctx, "root-bootstrap", "p")
			orphan := seedUserRecordOnly(ctx, seedSvc, "orphan-record", true)
			Expect(seedSvc.Close()).To(Succeed())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "orphan-record", Password: "newpassword",
			})).Error().To(MatchError(auth.ErrInvalidCredentials))
			s := openRootUser(ctx, "orphan-record", "newpassword")
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "orphan-record", Password: "newpassword",
			})).To(Succeed())
			u := findUser(ctx, s, "orphan-record")
			Expect(u.Key).To(Equal(orphan.Key))
			Expect(u.RootUser).To(BeTrue())
			Expect(rootUsers(ctx, s)).To(HaveLen(1))
		})
		It("Should create the user record when an auth row exists without one", func(ctx SpecContext) {
			seedAuthRowOnly(ctx, "orphan-auth", "p")
			s := openRootUser(ctx, "orphan-auth", "p")
			Expect(findUser(ctx, s, "orphan-auth").RootUser).To(BeTrue())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "orphan-auth", Password: "p",
			})).To(Succeed())
			Expect(rootUsers(ctx, s)).To(HaveLen(1))
		})
		It("Should rotate the auth password when an orphan auth row has a different password", func(ctx SpecContext) {
			seedAuthRowOnly(ctx, "orphan-auth-rot", "old-password")
			s := openRootUser(ctx, "orphan-auth-rot", "new-password")
			Expect(findUser(ctx, s, "orphan-auth-rot").RootUser).To(BeTrue())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "orphan-auth-rot", Password: "new-password",
			})).To(Succeed())
			Expect(authSvc.Authenticate(ctx, auth.Credentials{
				Username: "orphan-auth-rot", Password: "old-password",
			})).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
	})
})
