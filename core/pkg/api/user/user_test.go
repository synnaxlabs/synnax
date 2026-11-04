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
	"github.com/synnaxlabs/freighter"
	apiuser "github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// nonRootCtx returns a freighter.Context whose Subject is a freshly-created user
// holding no roles. Every RBAC enforcement against this subject must fail with
// access.ErrDenied. Returns both the context and the underlying user so callers can
// assert on identity-bearing behavior (e.g., the self-rename guard).
func nonRootCtx(ctx SpecContext) (freighter.Context, user.User) {
	GinkgoHelper()
	u := MustSucceed(writer.Create(ctx, user.User{Name: "non-root"}))
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx, u
}

// createUser creates a user and a password holding username.
func createUser(ctx SpecContext, username string) user.User {
	GinkgoHelper()
	u := MustSucceed(writer.Create(ctx, user.User{Name: username}))
	Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), auth.Credentials{
		Username: username,
		Password: "p",
	})).To(Succeed())
	return u
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It("Should create users and give each a new key", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
				Users: []user.User{{Name: "Ada"}, {Name: "Grace"}},
			}))
			Expect(res.Users).To(HaveLen(2))
			Expect(res.Users[0].Name).To(Equal("Ada"))
			Expect(res.Users[1].Name).To(Equal("Grace"))
			Expect(res.Users[0].Key).ToNot(Equal(uuid.Nil()))
			Expect(res.Users[0].Key).ToNot(Equal(res.Users[1].Key))
		})
		It("Should keep a key the caller gives", func(ctx SpecContext) {
			key := uuid.New()
			res := MustSucceed(apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
				Users: []user.User{{Key: key, Name: "Ada"}},
			}))
			Expect(res.Users[0].Key).To(Equal(key))
		})
		It("Should refuse a root user", func(ctx SpecContext) {
			Expect(apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
				Users: []user.User{{Name: "root", RootUser: true}},
			})).Error().To(MatchError(validate.ErrValidation))
		})
		It(
			"Should deny access when the subject lacks create permission",
			func(ctx SpecContext) {
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Create(fctx, db, apiuser.CreateRequest{
					Users: []user.User{{Name: "denied"}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})
	Describe("Retrieve", func() {
		It(
			"Should retrieve users by key when the subject has retrieve access",
			func(ctx SpecContext) {
				u := createUser(ctx, "retrieve-by-key-"+uuid.New().String())
				res := MustSucceed(
					apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
						Keys: []user.Key{u.Key},
					}),
				)
				Expect(res.Users).To(ConsistOf(u))
			},
		)
		It(
			"Should deny access when the subject lacks retrieve permission on any user",
			func(ctx SpecContext) {
				u := createUser(ctx, "retrieve-denied-"+uuid.New().String())
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Retrieve(fctx, apiuser.RetrieveRequest{
					Keys: []user.Key{u.Key},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should return query.ErrNotFound when any requested key does not exist",
			func(ctx SpecContext) {
				Expect(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
					Keys: []user.Key{uuid.New()},
				})).Error().To(MatchError(query.ErrNotFound))
			},
		)
	})

	Describe("Rename", func() {
		It(
			"Should update the name of the target user",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{Name: "rename"}))
				Expect(apiSvc.Rename(rootCtx(ctx), db, apiuser.RenameRequest{
					Key:  u.Key,
					Name: "Renamed User",
				})).Error().ToNot(HaveOccurred())
				var updated user.User
				Expect(userSvc.NewRetrieve().Where(user.MatchKeys(u.Key)).
					Entry(&updated).Exec(ctx, nil)).To(Succeed())
				Expect(updated.Name).To(Equal("Renamed User"))
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{Name: "rename-denied"}))
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Rename(fctx, db, apiuser.RenameRequest{
					Key:  u.Key,
					Name: "Should Not Apply",
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("Delete", func() {
		It(
			"Should be a no-op when none of the supplied keys exist",
			func(ctx SpecContext) {
				Expect(
					apiSvc.Delete(
						rootCtx(ctx),
						db,
						apiuser.DeleteRequest{Keys: []user.Key{uuid.New()}},
					),
				).
					Error().
					ToNot(HaveOccurred())
			},
		)
		It(
			"Should delete existing users and ignore unknown keys in the same call",
			func(ctx SpecContext) {
				username := uuid.New().String()
				created := MustSucceed(
					writer.Create(ctx, user.User{Name: username}),
				)
				Expect(passwordSvc.NewWriter(nil).Set(
					ctx,
					created.OntologyID(),
					auth.Credentials{Username: username, Password: "password"},
				)).To(Succeed())
				Expect(apiSvc.Delete(
					rootCtx(ctx),
					db,
					apiuser.DeleteRequest{Keys: []user.Key{created.Key, uuid.New()}},
				)).Error().ToNot(HaveOccurred())
				Expect(
					userSvc.NewRetrieve().
						Where(user.MatchKeys(created.Key)).
						Exists(ctx, nil),
				).
					To(BeFalse())
				Expect(login(ctx, username, "password")).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should deny access when the subject lacks delete permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{Name: "delete-denied"}))
				fctx, _ := nonRootCtx(ctx)
				Expect(
					apiSvc.Delete(
						fctx,
						db,
						apiuser.DeleteRequest{Keys: []user.Key{u.Key}},
					),
				).
					Error().
					To(MatchError(access.ErrDenied))
				Expect(
					userSvc.NewRetrieve().Where(user.MatchKeys(u.Key)).Exists(ctx, nil),
				).
					To(BeTrue())
			},
		)
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiuser.NewService(apiuser.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("rbac: must be non-nil")))
	})
})
