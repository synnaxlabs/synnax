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
		It(
			"Should register users and their credentials in a single call",
			func(ctx SpecContext) {
				username := uuid.New().String()
				res := MustSucceed(
					apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
						Users: []apiuser.NewUser{
							{
								Username: username,
								Password: "p",
								Name:     "First Last",
							},
						},
					}),
				)
				Expect(res.Users).To(HaveLen(1))
				Expect(res.Users[0].Name).To(Equal("First Last"))
				Expect(res.Users[0].Key).ToNot(Equal(uuid.Nil()))
				MustSucceed(login(ctx, username, "p"))
			},
		)
		It("Should name a user after its username by default", func(ctx SpecContext) {
			username := uuid.New().String()
			res := MustSucceed(apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
				Users: []apiuser.NewUser{{
					Username: username,
					Password: "p",
				}},
			}))
			Expect(res.Users[0].Name).To(Equal(username))
		})
		It(
			"Should deny access when the subject lacks create permission",
			func(ctx SpecContext) {
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Create(fctx, db, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Username: "should-not-exist-" + uuid.New().String(),
						Password: "p",
					}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should roll back the password when user creation fails inside the tx",
			func(ctx SpecContext) {
				// the second user's repeated username must roll back the first
				username := "rollback-" + uuid.New().String()
				tx := DeferClose(db.OpenTx())
				Expect(apiSvc.Create(rootCtx(ctx), tx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{
						{Username: username, Password: "p"},
						{Username: username, Password: "p"},
					},
				})).Error().To(MatchError(auth.ErrRepeatedUsername))
				Expect(login(ctx, username, "p")).Error().
					To(MatchError(auth.ErrInvalidCredentials))
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
		It("Should retrieve users by the username of their password", func(
			ctx SpecContext,
		) {
			username := "retrieve-by-username-" + uuid.New().String()
			u := createUser(ctx, username)
			res := MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{username},
			}))
			Expect(res.Users).To(ConsistOf(u))
		})
		It("Should match the username of the password, not the user's name", func(
			ctx SpecContext,
		) {
			name := "user-name-" + uuid.New().String()
			password := "password-name-" + uuid.New().String()
			u := MustSucceed(writer.Create(ctx, user.User{Name: name}))
			Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), auth.Credentials{
				Username: password,
				Password: "p",
			})).To(Succeed())
			res := MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{password},
			}))
			Expect(res.Users).To(HaveLen(1))
			Expect(res.Users[0].Key).To(Equal(u.Key))
			res = MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{name},
			}))
			Expect(res.Users).To(BeEmpty())
		})
		It("Should return no users for an unknown username", func(ctx SpecContext) {
			res := MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{"unknown-" + uuid.New().String()},
			}))
			Expect(res.Users).To(BeEmpty())
		})
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

	Describe("ChangeUsername", func() {
		It(
			"Should rename the user's password",
			func(ctx SpecContext) {
				oldName := "change-username-" + uuid.New().String()
				newName := "change-username-new-" + uuid.New().String()
				u := createUser(ctx, oldName)

				Expect(
					apiSvc.ChangeUsername(
						rootCtx(ctx),
						db,
						apiuser.ChangeUsernameRequest{
							Key:      u.Key,
							Username: newName,
						},
					),
				).Error().
					ToNot(HaveOccurred())

				res := MustSucceed(apiSvc.Retrieve(
					rootCtx(ctx),
					apiuser.RetrieveRequest{Usernames: []string{newName}},
				))
				Expect(res.Users).To(ConsistOf(u))
				MustSucceed(login(ctx, newName, "p"))
				Expect(login(ctx, oldName, "p")).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should be a no-op when the target name already matches",
			func(ctx SpecContext) {
				username := "change-username-noop-" + uuid.New().String()
				u := createUser(ctx, username)
				Expect(
					apiSvc.ChangeUsername(
						rootCtx(ctx),
						db,
						apiuser.ChangeUsernameRequest{
							Key:      u.Key,
							Username: username,
						},
					),
				).Error().
					ToNot(HaveOccurred())
			},
		)
		It(
			"Should reject a self-rename through the user service",
			func(ctx SpecContext) {
				fctx, subject := nonRootCtx(ctx)
				Expect(apiSvc.ChangeUsername(fctx, db, apiuser.ChangeUsernameRequest{
					Key:      subject.Key,
					Username: "anything",
				})).Error().To(MatchError(ContainSubstring("change your own username")))
			},
		)
		It(
			"Should return an error when the target user does not exist",
			func(ctx SpecContext) {
				Expect(
					apiSvc.ChangeUsername(
						rootCtx(ctx),
						db,
						apiuser.ChangeUsernameRequest{
							Key:      uuid.New(),
							Username: "does-not-matter-" + uuid.New().String(),
						},
					),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-username-denied-"+uuid.New().String())
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.ChangeUsername(fctx, db, apiuser.ChangeUsernameRequest{
					Key:      u.Key,
					Username: "should-not-apply-" + uuid.New().String(),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should return auth.ErrRepeatedUsername when the target name is already taken",
			func(ctx SpecContext) {
				taken := "change-username-taken-" + uuid.New().String()
				createUser(ctx, taken)
				target := createUser(
					ctx,
					"change-username-collide-"+uuid.New().String(),
				)
				Expect(
					apiSvc.ChangeUsername(
						rootCtx(ctx),
						db,
						apiuser.ChangeUsernameRequest{
							Key:      target.Key,
							Username: taken,
						},
					),
				).Error().
					To(MatchError(auth.ErrRepeatedUsername))
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
