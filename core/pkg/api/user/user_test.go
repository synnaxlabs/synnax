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
	u := MustSucceed(writer.Create(ctx, user.User{
		Username: "non-root-" + uuid.NewString(),
	}))
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx, u
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It(
			"Should register users and their credentials in a single call",
			func(ctx SpecContext) {
				username := uuid.NewString()
				res := MustSucceed(
					apiSvc.Create(rootCtx(ctx), db, apiuser.CreateRequest{
						Users: []apiuser.NewUser{
							{
								Credentials: auth.Credentials{
									Username: username,
									Password: "p",
								},
								FirstName: "First",
								LastName:  "Last",
							},
						},
					}),
				)
				Expect(res.Users).To(HaveLen(1))
				Expect(res.Users[0].Username).To(Equal(username))
				Expect(res.Users[0].FirstName).To(Equal("First"))
				Expect(res.Users[0].LastName).To(Equal("Last"))
				Expect(res.Users[0].Key).ToNot(Equal(uuid.Nil))
				Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
					Username: username, Password: "p",
				})).To(Succeed())
			},
		)
		It(
			"Should deny access when the subject lacks create permission",
			func(ctx SpecContext) {
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Create(fctx, db, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Credentials: auth.Credentials{
							Username: "should-not-exist-" + uuid.NewString(),
							Password: "p",
						},
					}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should roll back the auth row when user creation fails inside the tx",
			func(ctx SpecContext) {
				// Trigger a per-tx rollback by requesting two users whose usernames
				// collide: the first auth row registers, then the second user-record
				// create returns auth.ErrRepeatedUsername, which must roll back the
				// first auth row.
				username := "rollback-" + uuid.NewString()
				tx := DeferClose(db.OpenTx())
				Expect(apiSvc.Create(rootCtx(ctx), tx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{
						{
							Credentials: auth.Credentials{
								Username: username,
								Password: "p",
							},
						},
						{
							Credentials: auth.Credentials{
								Username: username,
								Password: "p",
							},
						},
					},
				})).Error().To(MatchError(auth.ErrRepeatedUsername))
				Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
					Username: username, Password: "p",
				})).To(MatchError(auth.ErrInvalidCredentials))
			},
		)
	})

	Describe("Retrieve", func() {
		It(
			"Should retrieve users by key when the subject has retrieve access",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "retrieve-by-key-" + uuid.NewString(),
				}))
				res := MustSucceed(
					apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
						Keys: []user.Key{u.Key},
					}),
				)
				Expect(res.Users).To(ConsistOf(u))
			},
		)
		It("Should retrieve users by username", func(ctx SpecContext) {
			username := "retrieve-by-username-" + uuid.NewString()
			u := MustSucceed(writer.Create(ctx, user.User{
				Username: username,
			}))
			res := MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{username},
			}))
			Expect(res.Users).To(ConsistOf(u))
		})
		It(
			"Should deny access when the subject lacks retrieve permission on any user",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "retrieve-denied-" + uuid.NewString(),
				}))
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
			"Should update the first and last name of the target user",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "rename-" + uuid.NewString(),
				}))
				Expect(apiSvc.Rename(rootCtx(ctx), db, apiuser.RenameRequest{
					Key:       u.Key,
					FirstName: "Renamed",
					LastName:  "User",
				})).Error().ToNot(HaveOccurred())
				var updated user.User
				Expect(userSvc.NewRetrieve().Where(user.MatchKeys(u.Key)).
					Entry(&updated).Exec(ctx, nil)).To(Succeed())
				Expect(updated.FirstName).To(Equal("Renamed"))
				Expect(updated.LastName).To(Equal("User"))
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "rename-denied-" + uuid.NewString(),
				}))
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.Rename(fctx, db, apiuser.RenameRequest{
					Key:       u.Key,
					FirstName: "Should",
					LastName:  "NotApply",
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
	})

	Describe("ChangeUsername", func() {
		It(
			"Should rename the user and rotate the matching auth row",
			func(ctx SpecContext) {
				oldName := "change-username-" + uuid.NewString()
				newName := "change-username-new-" + uuid.NewString()
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: oldName,
				}))
				Expect(authSvc.NewWriter(nil).Register(ctx, auth.Credentials{
					Username: oldName, Password: "p",
				})).To(Succeed())

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

				var updated user.User
				Expect(userSvc.NewRetrieve().Where(user.MatchKeys(u.Key)).
					Entry(&updated).Exec(ctx, nil)).To(Succeed())
				Expect(updated.Username).To(Equal(newName))
				Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
					Username: newName, Password: "p",
				})).To(Succeed())
				Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
					Username: oldName, Password: "p",
				})).To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should be a no-op when the target name already matches",
			func(ctx SpecContext) {
				username := "change-username-noop-" + uuid.NewString()
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: username,
				}))
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
							Username: "does-not-matter-" + uuid.NewString(),
						},
					),
				).Error().
					To(HaveOccurred())
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "change-username-denied-" + uuid.NewString(),
				}))
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.ChangeUsername(fctx, db, apiuser.ChangeUsernameRequest{
					Key:      u.Key,
					Username: "should-not-apply-" + uuid.NewString(),
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should return auth.ErrRepeatedUsername when the target name is already taken",
			func(ctx SpecContext) {
				taken := "change-username-taken-" + uuid.NewString()
				MustSucceed(writer.Create(ctx, user.User{
					Username: taken,
				}))
				target := MustSucceed(writer.Create(ctx, user.User{
					Username: "change-username-collide-" + uuid.NewString(),
				}))
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
				username := uuid.NewString()
				created := MustSucceed(
					writer.Create(ctx, user.User{Username: username}),
				)
				Expect(authSvc.NewWriter(nil).Register(ctx, auth.Credentials{
					Username: username,
					Password: "password",
				})).To(Succeed())
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
				Expect(authSvc.Authenticate(ctx, nil, auth.Credentials{
					Username: username,
					Password: "password",
				})).To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should deny access when the subject lacks delete permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "delete-denied-" + uuid.NewString(),
				}))
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
