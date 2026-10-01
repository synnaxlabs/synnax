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
	"fmt"
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiuser "github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// nonRootCtx returns a freighter.Context whose Subject is a freshly-created user
// holding no roles. Every RBAC enforcement against this subject must fail with
// access.ErrDenied. Returns both the context and the underlying user so callers can
// assert on identity-bearing behavior (e.g., the self-rename guard).
func nonRootCtx(ctx SpecContext) (freighter.Context, user.User) {
	GinkgoHelper()
	u := MustSucceed(writer.Create(ctx, user.User{
		Username: "non-root-" + uuid.New().String(),
	}))
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.OntologyID())
	return fctx, u
}

// createUser commits a new user holding password through the API and returns it.
func createUser(ctx SpecContext, prefix, password string) user.User {
	GinkgoHelper()
	return upsert(ctx, apiuser.NewUser{
		Username: prefix + "-" + uuid.New().String(),
		Password: password,
	})
}

// upsert commits a create request for nu and returns the resulting user.
func upsert(ctx SpecContext, nu apiuser.NewUser) user.User {
	GinkgoHelper()
	var u user.User
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		res, err := apiSvc.Create(rootCtx(ctx), tx, apiuser.CreateRequest{
			Users: []apiuser.NewUser{nu},
		})
		if err == nil {
			u = res.Users[0]
		}
		return err
	})).To(Succeed())
	return u
}

// usernameExists returns true if a user holds username.
func usernameExists(ctx SpecContext, username string) bool {
	GinkgoHelper()
	return len(MustSucceed(authSvc.KeysByUsername(ctx, nil, username))) > 0
}

// authenticate checks password against the credentials of the user with the given key.
func authenticate(ctx SpecContext, key user.Key, password string) error {
	GinkgoHelper()
	username := MustSucceed(authSvc.UsernamesByKey(ctx, nil, key))[key]
	if username == "" {
		return auth.ErrInvalidCredentials
	}
	authenticated, err := authSvc.Authenticate(ctx, nil, auth.Credentials{
		Username: username,
		Password: password,
	})
	if err != nil {
		return err
	}
	Expect(authenticated).To(Equal(key))
	return nil
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
								Username:  username,
								Password:  "p",
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
				Expect(res.Users[0].Key).ToNot(Equal(uuid.Nil()))
				Expect(authenticate(ctx, res.Users[0].Key, "p")).
					To(Succeed())
			},
		)
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
		Describe("Existing key", func() {
			It(
				"Should rename the user, rotate the password, and free the old username",
				func(ctx SpecContext) {
					u := createUser(ctx, "existing", "one")
					renamed := "renamed-" + uuid.New().String()
					res := upsert(ctx, apiuser.NewUser{
						Username:  renamed,
						Password:  "two",
						FirstName: "First",
						Key:       u.Key,
					})
					Expect(res.Key).To(Equal(u.Key))
					Expect(res.Username).To(Equal(renamed))
					Expect(res.FirstName).To(Equal("First"))
					Expect(authenticate(ctx, u.Key, "two")).To(Succeed())
					Expect(authenticate(ctx, u.Key, "one")).
						To(MatchError(auth.ErrInvalidCredentials))
					Expect(usernameExists(ctx, u.Username)).To(BeFalse())
					other := upsert(ctx, apiuser.NewUser{
						Username: u.Username,
						Password: "three",
					})
					Expect(other.Key).ToNot(Equal(u.Key))
					Expect(authenticate(ctx, other.Key, "three")).
						To(Succeed())
				},
			)
			It(
				"Should rotate the password when the username is unchanged",
				func(ctx SpecContext) {
					u := createUser(ctx, "existing", "one")
					res := upsert(ctx, apiuser.NewUser{
						Username: u.Username,
						Password: "two",
						Key:      u.Key,
					})
					Expect(res.Username).To(Equal(u.Username))
					Expect(authenticate(ctx, u.Key, "two")).To(Succeed())
					Expect(authenticate(ctx, u.Key, "one")).
						To(MatchError(auth.ErrInvalidCredentials))
				},
			)
			It(
				"Should reject a username held by another user and roll back",
				func(ctx SpecContext) {
					u := createUser(ctx, "existing", "one")
					taken := createUser(ctx, "existing", "two")
					tx := DeferClose(db.OpenTx())
					Expect(apiSvc.Create(rootCtx(ctx), tx, apiuser.CreateRequest{
						Users: []apiuser.NewUser{{
							Username: taken.Username,
							Password: "three",
							Key:      u.Key,
						}},
					})).Error().To(MatchError(auth.ErrRepeatedUsername))
					Expect(authenticate(ctx, u.Key, "one")).To(Succeed())
					Expect(authenticate(ctx, taken.Key, "two")).
						To(Succeed())
				},
			)
			It(
				"Should deny access when the subject lacks update permission",
				func(ctx SpecContext) {
					u := createUser(ctx, "existing", "one")
					fctx, _ := nonRootCtx(ctx)
					Expect(apiSvc.Create(fctx, db, apiuser.CreateRequest{
						Users: []apiuser.NewUser{{
							Username: u.Username,
							Password: "two",
							Key:      u.Key,
						}},
					})).Error().To(MatchError(access.ErrDenied))
				},
			)
		})
		It(
			"Should roll back the auth row when user creation fails inside the tx",
			func(ctx SpecContext) {
				// Two users whose usernames collide: the first user and its password
				// write, then the second create returns auth.ErrRepeatedUsername,
				// which must roll back the first.
				username := "rollback-" + uuid.New().String()
				tx := DeferClose(db.OpenTx())
				Expect(apiSvc.Create(rootCtx(ctx), tx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{
						{
							Username: username,
							Password: "p",
						},
						{
							Username: username,
							Password: "p",
						},
					},
				})).Error().To(MatchError(auth.ErrRepeatedUsername))
				Expect(usernameExists(ctx, username)).To(BeFalse())
			},
		)
	})

	Describe("Retrieve", func() {
		It(
			"Should retrieve users by key when the subject has retrieve access",
			func(ctx SpecContext) {
				u := createUser(ctx, "retrieve-by-key", "p")
				res := MustSucceed(
					apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
						Keys: []user.Key{u.Key},
					}),
				)
				Expect(res.Users).To(ConsistOf(u))
			},
		)
		It("Should retrieve users by username", func(ctx SpecContext) {
			u := createUser(ctx, "retrieve-by-username", "p")
			res := MustSucceed(apiSvc.Retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Usernames: []string{u.Username},
			}))
			Expect(res.Users).To(ConsistOf(u))
		})
		It(
			"Should deny access when the subject lacks retrieve permission on any user",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "retrieve-denied-" + uuid.New().String(),
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
					Username: "rename-" + uuid.New().String(),
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
					Username: "rename-denied-" + uuid.New().String(),
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
			"Should rename the user and keep its password",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-username", "p")
				newName := "change-username-new-" + uuid.New().String()
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
				Expect(authSvc.UsernamesByKey(ctx, nil, u.Key)).
					To(HaveKeyWithValue(u.Key, newName))
				Expect(authenticate(ctx, u.Key, "p")).To(Succeed())
			},
		)
		It(
			"Should free the old username for a new user",
			func(ctx SpecContext) {
				u := createUser(ctx, "stranded", "p")
				Expect(apiSvc.ChangeUsername(
					rootCtx(ctx),
					db,
					apiuser.ChangeUsernameRequest{
						Key:      u.Key,
						Username: "stranded-new-" + uuid.New().String(),
					},
				)).Error().ToNot(HaveOccurred())
				reused := upsert(ctx, apiuser.NewUser{
					Username: u.Username,
					Password: "q",
				})
				Expect(authenticate(ctx, reused.Key, "q")).To(Succeed())
				Expect(authenticate(ctx, u.Key, "p")).To(Succeed())
			},
		)
		It(
			"Should be a no-op when the target name already matches",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-username-noop", "p")
				Expect(
					apiSvc.ChangeUsername(
						rootCtx(ctx),
						db,
						apiuser.ChangeUsernameRequest{
							Key:      u.Key,
							Username: u.Username,
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
					To(HaveOccurred())
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "change-username-denied-" + uuid.New().String(),
				}))
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
				taken := createUser(ctx, "change-username-taken", "p").Username
				target := createUser(ctx, "change-username-collide", "p")
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

	Describe("ChangePassword", func() {
		It(
			"Should replace the password without the current one",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-password", "old")
				Expect(apiSvc.ChangePassword(
					rootCtx(ctx),
					db,
					apiuser.ChangePasswordRequest{Key: u.Key, Password: "new"},
				)).To(Equal(struct{}{}))
				Expect(authenticate(ctx, u.Key, "new")).To(Succeed())
				Expect(authenticate(ctx, u.Key, "old")).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should refuse to change the root user's password",
			func(ctx SpecContext) {
				Expect(apiSvc.ChangePassword(
					rootCtx(ctx),
					db,
					apiuser.ChangePasswordRequest{Key: root.Key, Password: "root-new"},
				)).Error().To(MatchError(user.ErrRootCredentialsManaged))
				Expect(authenticate(ctx, root.Key, "p")).To(Succeed())
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-password-denied", "old")
				fctx, _ := nonRootCtx(ctx)
				Expect(apiSvc.ChangePassword(fctx, db, apiuser.ChangePasswordRequest{
					Key:      u.Key,
					Password: "new",
				})).Error().To(MatchError(access.ErrDenied))
				Expect(authenticate(ctx, u.Key, "old")).To(Succeed())
			},
		)
		It(
			"Should return query.ErrNotFound when the target user does not exist",
			func(ctx SpecContext) {
				Expect(apiSvc.ChangePassword(
					rootCtx(ctx),
					db,
					apiuser.ChangePasswordRequest{Key: uuid.New(), Password: "new"},
				)).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It("Should reject an empty password", func(ctx SpecContext) {
			u := createUser(ctx, "change-password-empty", "old")
			Expect(apiSvc.ChangePassword(
				rootCtx(ctx),
				db,
				apiuser.ChangePasswordRequest{Key: u.Key, Password: ""},
			)).Error().To(MatchError(ContainSubstring("password: required")))
		})
		It(
			"Should keep the new password when a rename runs concurrently",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-password-race", "p0")
				for i := range 5 {
					var (
						newName     = fmt.Sprintf("%s-%d", u.Username, i)
						newPassword = fmt.Sprintf("p%d", i+1)
						wg          sync.WaitGroup
					)
					wg.Go(func() {
						defer GinkgoRecover()
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							_, err := apiSvc.ChangePassword(
								rootCtx(ctx),
								tx,
								apiuser.ChangePasswordRequest{
									Key:      u.Key,
									Password: newPassword,
								},
							)
							return err
						})).To(Succeed())
					})
					wg.Go(func() {
						defer GinkgoRecover()
						Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
							_, err := apiSvc.ChangeUsername(
								rootCtx(ctx),
								tx,
								apiuser.ChangeUsernameRequest{
									Key:      u.Key,
									Username: newName,
								},
							)
							return err
						})).To(Succeed())
					})
					wg.Wait()
					Expect(usernameExists(ctx, newName)).To(BeTrue())
					Expect(authenticate(ctx, u.Key, newPassword)).
						To(Succeed())
				}
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
				created := createUser(ctx, "delete", "password")
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
				Expect(authenticate(ctx, created.Key, "password")).
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should deny access when the subject lacks delete permission",
			func(ctx SpecContext) {
				u := MustSucceed(writer.Create(ctx, user.User{
					Username: "delete-denied-" + uuid.New().String(),
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
