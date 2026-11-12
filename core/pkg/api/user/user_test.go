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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	apiuser "github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// nonRootCtx returns a freighter.Context whose Subject is a new user that can do
// nothing, so every RBAC enforcement against it fails with access.ErrDenied. It also
// returns the user, so callers can assert on identity, such as the self-rename guard.
func nonRootCtx(ctx SpecContext) (freighter.Context, user.User) {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(
		ctx,
		db,
		rbacSvc,
		userSvc,
		user.User{Name: "non-root"},
	))
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.ResourceID())
	return fctx, u
}

// createUser creates a user that holds a role with no policies, and a password holding
// username.
func createUser(ctx SpecContext, username string) user.User {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(
		ctx,
		db,
		rbacSvc,
		userSvc,
		user.User{Name: username},
	))
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return passwordSvc.NewWriter(tx).
			Set(ctx, u.ResourceID(), username, hash(passwordSvc, "p"))
	})).To(Succeed())
	return u
}

// grantOnUsers grants the action on every user to u.
func grantOnUsers(ctx SpecContext, u user.User, action access.Action) {
	GinkgoHelper()
	MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		rbacSvc,
		userSvc,
		u.Key,
		[]access.Action{action},
		resource.ID{Type: resource.TypeUser},
	))
}

// grantRoleChange grants update on every user and on the roles to u.
func grantRoleChange(ctx SpecContext, u user.User, roles ...role.Key) {
	GinkgoHelper()
	MustSucceed(rbactestutil.Grant(
		ctx,
		db,
		rbacSvc,
		userSvc,
		u.Key,
		[]access.Action{access.ActionUpdate},
		append([]resource.ID{{Type: resource.TypeUser}}, role.ResourceIDs(roles)...)...,
	))
}

// createRole creates a role that holds no policies.
func createRole(ctx SpecContext) role.Key {
	GinkgoHelper()
	r := &role.Role{Name: "role-" + uuid.New().String()}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rbacSvc.Role.NewWriter(tx, true).Create(ctx, r)
	})).To(Succeed())
	return r.Key
}

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// beDeniedUpdateOn matches the error that denies the subject update on the role.
func beDeniedUpdateOn(r role.Key) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring("cannot update %s", role.ResourceID(r))),
	)
}

// setUsername returns a dispatch that sets the username of the user with key.
func setUsername(key user.Key, username string) apiuser.DispatchRequest {
	return apiuser.DispatchRequest{
		Key: key,
		Actions: []user.Action{
			user.NewSetUsernameAction(user.SetUsernamePayload{Username: username}),
		},
	}
}

// setRoleOf returns a dispatch that gives the role r to the user with key.
func setRoleOf(key user.Key, r role.Key) apiuser.DispatchRequest {
	return apiuser.DispatchRequest{
		Key: key,
		Actions: []user.Action{
			user.NewSetAction(user.SetPayload{Path: []string{"role"}, Value: r}),
		},
	}
}

func roleOf(ctx SpecContext, key user.Key) role.Key {
	GinkgoHelper()
	var u user.User
	Expect(userSvc.NewRetrieve().
		Where(user.MatchKeys(key)).
		Entry(&u).
		Exec(ctx, db)).To(Succeed())
	return u.Role
}

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It(
			"Should register users and their credentials in a single call",
			func(ctx SpecContext) {
				username := uuid.New().String()
				res := MustSucceed(
					create(rootCtx(ctx), apiuser.CreateRequest{
						Users: []apiuser.NewUser{
							{
								Username: username,
								Password: "p",
								Name:     "First Last",
								Role:     rbacSvc.Builtin.ViewerKey,
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
			res := MustSucceed(create(rootCtx(ctx), apiuser.CreateRequest{
				Users: []apiuser.NewUser{{
					Username: username,
					Password: "p",
					Role:     rbacSvc.Builtin.ViewerKey,
				}},
			}))
			Expect(res.Users[0].Name).To(Equal(username))
		})
		It(
			"Should deny access when the subject lacks create permission",
			func(ctx SpecContext) {
				fctx, subject := nonRootCtx(ctx)
				key := uuid.New()
				Expect(create(fctx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Key:      key,
						Username: "should-not-exist-" + uuid.New().String(),
						Password: "p",
						Role:     rbacSvc.Builtin.ViewerKey,
					}},
				})).Error().To(denied(
					subject,
					access.ActionCreate,
					resource.ID{Type: resource.TypeUser},
				))
			},
		)
		It(
			"Should roll back the users when one fails to create",
			func(ctx SpecContext) {
				username := "rollback-" + uuid.New().String()
				viewer := rbacSvc.Builtin.ViewerKey
				Expect(create(rootCtx(ctx), apiuser.CreateRequest{
					Users: []apiuser.NewUser{
						{Username: username, Password: "p", Role: viewer},
						{Username: username, Password: "p", Role: viewer},
					},
				})).Error().To(MatchError(auth.ErrRepeatedUsername))
				Expect(login(ctx, username, "p")).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
	})

	Describe("Retrieve", func() {
		It("Should leave out the users that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			fctx, u := nonRootCtx(ctx)
			granted := createUser(ctx, "leave-out-granted-"+uuid.New().String())
			createUser(ctx, "leave-out-denied-"+uuid.New().String())
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				rbacSvc,
				userSvc,
				u.Key,
				[]access.Action{access.ActionRetrieve},
				granted.ResourceID(),
			))
			res := MustSucceed(retrieve(fctx, apiuser.RetrieveRequest{}))
			Expect(res.Users).To(ConsistOf(
				HaveField("Key", granted.Key),
				HaveField("Key", u.Key),
			))
		})

		Context("With no retrieve on users", func() {
			var (
				username, otherUsername string
				u, other                user.User
			)
			BeforeEach(func(ctx SpecContext) {
				username = "own-" + uuid.New().String()
				otherUsername = "other-" + uuid.New().String()
				u = createUser(ctx, username)
				other = createUser(ctx, otherUsername)
			})
			It("Should retrieve the subject's own user by key", func(ctx SpecContext) {
				res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiuser.RetrieveRequest{
					Where: &user.Where{Key: &filter.Set[user.Key]{Eq: &u.Key}},
				}))
				Expect(res.Users).To(ConsistOf(u))
			})
			It("Should retrieve the subject's own user by filter", func(
				ctx SpecContext,
			) {
				res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiuser.RetrieveRequest{
					Where: &user.Where{Usernames: []string{username}},
				}))
				Expect(res.Users).To(ConsistOf(u))
			})
			It("Should deny another user by key", func(ctx SpecContext) {
				Expect(retrieve(AuthedCtx(ctx, u), apiuser.RetrieveRequest{
					Where: &user.Where{Key: &filter.Set[user.Key]{Eq: &other.Key}},
				})).Error().To(denied(u, access.ActionRetrieve, other.ResourceID()))
			})
			It("Should leave out another user by filter", func(ctx SpecContext) {
				res := MustSucceed(retrieve(AuthedCtx(ctx, u), apiuser.RetrieveRequest{
					Where: &user.Where{Usernames: []string{username, otherUsername}},
				}))
				Expect(res.Users).To(ConsistOf(u))
			})
		})

		It(
			"Should retrieve users by key when the subject has retrieve access",
			func(ctx SpecContext) {
				u := createUser(ctx, "retrieve-by-key-"+uuid.New().String())
				res := MustSucceed(
					retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
						Where: &user.Where{
							Key: &filter.Set[user.Key]{In: []user.Key{u.Key}},
						},
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
			res := MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{Usernames: []string{username}},
			}))
			Expect(res.Users).To(ConsistOf(u))
		})
		It("Should match the username of the password, not the user's name", func(
			ctx SpecContext,
		) {
			name := "user-name-" + uuid.New().String()
			password := "password-name-" + uuid.New().String()
			u := MustSucceed(
				rbactestutil.CreateUser(
					ctx,
					db,
					rbacSvc,
					userSvc,
					user.User{Name: name},
				),
			)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return passwordSvc.NewWriter(tx).
					Set(ctx, u.ResourceID(), password, hash(passwordSvc, "p"))
			})).To(Succeed())
			res := MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{Usernames: []string{password}},
			}))
			Expect(res.Users).To(HaveLen(1))
			Expect(res.Users[0].Key).To(Equal(u.Key))
			res = MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{Usernames: []string{name}},
			}))
			Expect(res.Users).To(BeEmpty())
		})
		It("Should return no users for an unknown username", func(ctx SpecContext) {
			res := MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{
					Usernames: []string{"unknown-" + uuid.New().String()},
				},
			}))
			Expect(res.Users).To(BeEmpty())
		})
		It(
			"Should deny access when the subject lacks retrieve permission on any user",
			func(ctx SpecContext) {
				u := createUser(ctx, "retrieve-denied-"+uuid.New().String())
				fctx, _ := nonRootCtx(ctx)
				Expect(retrieve(fctx, apiuser.RetrieveRequest{
					Where: &user.Where{Key: &filter.Set[user.Key]{Eq: &u.Key}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should return no users for a key that does not exist",
			func(ctx SpecContext) {
				res := MustSucceed(
					retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
						Where: &user.Where{
							Key: &filter.Set[user.Key]{Eq: new(uuid.New())},
						},
					}),
				)
				Expect(res.Users).To(BeEmpty())
			},
		)
		It("Should search users by name", func(ctx SpecContext) {
			name := "searchable-" + uuid.New().String()
			u := MustSucceed(
				rbactestutil.CreateUser(
					ctx,
					db,
					rbacSvc,
					userSvc,
					user.User{Name: name},
				),
			)
			other := MustSucceed(
				rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
					Name: "hidden-" + uuid.New().String(),
				}),
			)
			res := MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{
					Key: &filter.Set[user.Key]{In: []user.Key{u.Key, other.Key}},
				},
				SearchTerm: name,
			}))
			Expect(res.Users).To(ConsistOf(u))
		})
	})

	Describe("Dispatch set username", func() {
		It(
			"Should rename the user's password",
			func(ctx SpecContext) {
				oldName := "change-username-" + uuid.New().String()
				newName := "change-username-new-" + uuid.New().String()
				u := createUser(ctx, oldName)

				Expect(
					dispatch(rootCtx(ctx), setUsername(u.Key, newName)),
				).Error().
					ToNot(HaveOccurred())

				res := MustSucceed(retrieve(
					rootCtx(ctx),
					apiuser.RetrieveRequest{
						Where: &user.Where{Usernames: []string{newName}},
					},
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
					dispatch(rootCtx(ctx), setUsername(u.Key, username)),
				).Error().
					ToNot(HaveOccurred())
			},
		)
		It(
			"Should reject a self-rename through the user service",
			func(ctx SpecContext) {
				fctx, subject := nonRootCtx(ctx)
				grantOnUsers(ctx, subject, access.ActionUpdate)
				Expect(
					dispatch(fctx, setUsername(subject.Key, "anything")),
				).Error().
					To(MatchError(ContainSubstring("change your own username")))
			},
		)
		It(
			"Should return an error when the target user does not exist",
			func(ctx SpecContext) {
				Expect(
					dispatch(
						rootCtx(ctx),
						setUsername(uuid.New(), "does-not-matter-"+uuid.New().String()),
					),
				).Error().
					To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should deny access when the subject lacks update permission",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-username-denied-"+uuid.New().String())
				fctx, subject := nonRootCtx(ctx)
				Expect(
					dispatch(
						fctx,
						setUsername(u.Key, "should-not-apply-"+uuid.New().String()),
					),
				).Error().
					To(denied(subject, access.ActionUpdate, u.ResourceID()))
			},
		)
		It(
			"Should change the username for a subject with update on the user",
			func(ctx SpecContext) {
				u := createUser(ctx, "change-username-granted-"+uuid.New().String())
				fctx, subject := nonRootCtx(ctx)
				grantOnUsers(ctx, subject, access.ActionUpdate)
				newName := "change-username-granted-new-" + uuid.New().String()
				MustSucceed(dispatch(fctx, setUsername(u.Key, newName)))
				MustSucceed(login(ctx, newName, "p"))
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
					dispatch(rootCtx(ctx), setUsername(target.Key, taken)),
				).Error().
					To(MatchError(auth.ErrRepeatedUsername))
			},
		)
	})

	Describe("Role", func() {
		It("Should give new users the provided role", func(ctx SpecContext) {
			viewer := rbacSvc.Builtin.ViewerKey
			res := MustSucceed(create(rootCtx(ctx), apiuser.CreateRequest{
				Users: []apiuser.NewUser{{
					Username: uuid.New().String(),
					Password: "p",
					Role:     viewer,
				}},
			}))
			Expect(roleOf(ctx, res.Users[0].Key)).To(Equal(viewer))
		})
		It("Should refuse to create a user with no role", func(ctx SpecContext) {
			username := uuid.New().String()
			Expect(create(rootCtx(ctx), apiuser.CreateRequest{
				Users: []apiuser.NewUser{{Username: username, Password: "p"}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring("role: must be non-zero")),
			))
			Expect(login(ctx, username, "p")).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It(
			"Should refuse to create a user with a role that does not exist",
			func(ctx SpecContext) {
				Expect(create(rootCtx(ctx), apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Username: uuid.New().String(),
						Password: "p",
						Role:     uuid.New(),
					}},
				})).Error().To(SatisfyAll(
					BeAssignableToTypeOf(validate.PathError{}),
					MatchError(ContainSubstring("does not exist")),
				))
			},
		)
		It(
			"Should deny giving a role to a new user without update access",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				grantOnUsers(ctx, u, access.ActionCreate)
				Expect(create(fctx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Username: uuid.New().String(),
						Password: "p",
						Role:     rbacSvc.Builtin.OwnerKey,
					}},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should deny giving a new user a role the subject cannot update",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				grantOnUsers(ctx, u, access.ActionCreate)
				grantOnUsers(ctx, u, access.ActionUpdate)
				r := createRole(ctx)
				Expect(create(fctx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Username: uuid.New().String(),
						Password: "p",
						Role:     r,
					}},
				})).Error().To(beDeniedUpdateOn(r))
			},
		)
		It(
			"Should give a new user a role the subject can update",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				grantOnUsers(ctx, u, access.ActionCreate)
				r := createRole(ctx)
				grantRoleChange(ctx, u, r)
				res := MustSucceed(create(fctx, apiuser.CreateRequest{
					Users: []apiuser.NewUser{{
						Username: uuid.New().String(),
						Password: "p",
						Role:     r,
					}},
				}))
				Expect(roleOf(ctx, res.Users[0].Key)).To(Equal(r))
			},
		)
		It(
			"Should deny setting a role when the subject lacks update access",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				before := roleOf(ctx, u.Key)
				Expect(
					dispatch(fctx, setRoleOf(u.Key, rbacSvc.Builtin.OwnerKey)),
				).Error().
					To(MatchError(access.ErrDenied))
				Expect(roleOf(ctx, u.Key)).To(Equal(before))
			},
		)
		It(
			"Should deny giving a role the subject cannot update",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				target := createUser(ctx, "set-role-"+uuid.New().String())
				before := roleOf(ctx, target.Key)
				grantRoleChange(ctx, u, before)
				r := createRole(ctx)
				Expect(
					dispatch(fctx, setRoleOf(target.Key, r)),
				).Error().
					To(beDeniedUpdateOn(r))
				Expect(roleOf(ctx, target.Key)).To(Equal(before))
			},
		)
		It(
			"Should deny taking away a role the subject cannot update",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				target := createUser(ctx, "set-role-"+uuid.New().String())
				before := roleOf(ctx, target.Key)
				r := createRole(ctx)
				grantRoleChange(ctx, u, r)
				Expect(
					dispatch(fctx, setRoleOf(target.Key, r)),
				).Error().
					To(beDeniedUpdateOn(before))
				Expect(roleOf(ctx, target.Key)).To(Equal(before))
			},
		)
		It(
			"Should change the role when the subject can update the user and both roles",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				target := createUser(ctx, "set-role-"+uuid.New().String())
				r := createRole(ctx)
				grantRoleChange(ctx, u, roleOf(ctx, target.Key), r)
				Expect(
					dispatch(fctx, setRoleOf(target.Key, r)),
				).Error().
					ToNot(HaveOccurred())
				Expect(roleOf(ctx, target.Key)).To(Equal(r))
			},
		)
		It("Should refuse to change the role of the root user", func(ctx SpecContext) {
			fctx, u := nonRootCtx(ctx)
			grantRoleChange(ctx, u, rbacSvc.Builtin.OwnerKey, rbacSvc.Builtin.ViewerKey)
			Expect(
				dispatch(fctx, setRoleOf(root.Key, rbacSvc.Builtin.ViewerKey)),
			).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot change the role of root user "+root.Name,
					)),
				))
			Expect(roleOf(ctx, root.Key)).To(Equal(rbacSvc.Builtin.OwnerKey))
		})
		It(
			"Should not reveal the root user to a subject that cannot update it",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				Expect(
					dispatch(fctx, setRoleOf(root.Key, rbacSvc.Builtin.ViewerKey)),
				).Error().
					To(SatisfyAll(
						denied(u, access.ActionUpdate, root.ResourceID()),
						Not(MatchError(ContainSubstring("root user"))),
					))
			},
		)
		It(
			"Should give a role with no update on the roles when the role stays",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				target := createUser(ctx, "set-role-"+uuid.New().String())
				grantOnUsers(ctx, u, access.ActionUpdate)
				MustSucceed(
					dispatch(fctx, setRoleOf(target.Key, roleOf(ctx, target.Key))),
				)
			},
		)
		It("Should return not found for a user that does not exist", func(
			ctx SpecContext,
		) {
			Expect(
				dispatch(
					rootCtx(ctx),
					setRoleOf(uuid.New(), rbacSvc.Builtin.ViewerKey),
				),
			).Error().
				To(MatchError(query.ErrNotFound))
		})
		It(
			"Should move a user that is not the root user off the Owner role",
			func(ctx SpecContext) {
				owner := createUser(ctx, "owner-"+uuid.New().String())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return userSvc.NewWriter(tx).
						SetRole(ctx, rbacSvc.Builtin.OwnerKey, owner.Key)
				})).
					To(Succeed())
				Expect(
					dispatch(
						rootCtx(ctx),
						setRoleOf(owner.Key, rbacSvc.Builtin.ViewerKey),
					),
				).Error().
					ToNot(HaveOccurred())
				Expect(roleOf(ctx, owner.Key)).To(Equal(rbacSvc.Builtin.ViewerKey))
			},
		)
		It("Should retrieve the users that hold any of the roles", func(
			ctx SpecContext,
		) {
			r := createRole(ctx)
			u := createUser(ctx, "by-role-"+uuid.New().String())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return userSvc.NewWriter(tx).SetRole(ctx, r, u.Key)
			})).To(Succeed())
			res := MustSucceed(retrieve(rootCtx(ctx), apiuser.RetrieveRequest{
				Where: &user.Where{Roles: []role.Key{r, uuid.New()}},
			}))
			Expect(res.Users).To(HaveExactElements(HaveField("Key", u.Key)))
		})
	})

	Describe("Dispatch", func() {
		set := func(path string, value any) user.Action {
			return user.NewSetAction(
				user.SetPayload{Path: []string{path}, Value: value},
			)
		}
		request := func(key user.Key, acts ...user.Action) apiuser.DispatchRequest {
			return apiuser.DispatchRequest{Key: key, Actions: acts}
		}
		It("Should apply the actions to the user", func(ctx SpecContext) {
			u := createUser(ctx, "dispatch-"+uuid.New().String())
			engineer := rbacSvc.Builtin.EngineerKey
			MustSucceed(dispatch(
				rootCtx(ctx),
				request(u.Key, set("name", "Dispatched"), set("role", engineer)),
			))
			Expect(roleOf(ctx, u.Key)).To(Equal(engineer))
		})
	})

	Describe("Delete", func() {
		It(
			"Should be a no-op when none of the supplied keys exist",
			func(ctx SpecContext) {
				Expect(
					del(
						rootCtx(ctx),
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
					rbactestutil.CreateUser(
						ctx,
						db,
						rbacSvc,
						userSvc,
						user.User{Name: username},
					),
				)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return passwordSvc.NewWriter(tx).Set(
						ctx,
						created.ResourceID(),
						username,
						hash(passwordSvc, "password"),
					)
				})).To(Succeed())
				Expect(del(
					rootCtx(ctx),
					apiuser.DeleteRequest{Keys: []user.Key{created.Key, uuid.New()}},
				)).Error().ToNot(HaveOccurred())
				Expect(
					userSvc.NewRetrieve().
						Where(user.MatchKeys(created.Key)).
						Exists(ctx, db),
				).
					To(BeFalse())
				Expect(login(ctx, username, "password")).Error().
					To(MatchError(auth.ErrInvalidCredentials))
			},
		)
		It(
			"Should deny access when the subject lacks delete permission",
			func(ctx SpecContext) {
				u := MustSucceed(
					rbactestutil.CreateUser(
						ctx,
						db,
						rbacSvc,
						userSvc,
						user.User{Name: "delete-denied"},
					),
				)
				fctx, subject := nonRootCtx(ctx)
				grantRoleChange(ctx, subject, roleOf(ctx, u.Key))
				Expect(
					del(
						fctx,
						apiuser.DeleteRequest{Keys: []user.Key{u.Key}},
					),
				).
					Error().
					To(denied(subject, access.ActionDelete, u.ResourceID()))
				Expect(
					userSvc.NewRetrieve().Where(user.MatchKeys(u.Key)).Exists(ctx, db),
				).
					To(BeTrue())
			},
		)
		It("Should require delete on the user and update on its role", func(
			ctx SpecContext,
		) {
			u := createUser(ctx, "delete-role-"+uuid.New().String())
			fctx, subject := nonRootCtx(ctx)
			grantOnUsers(ctx, subject, access.ActionDelete)
			Expect(del(fctx, apiuser.DeleteRequest{Keys: []user.Key{u.Key}})).
				Error().To(beDeniedUpdateOn(roleOf(ctx, u.Key)))
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				rbacSvc,
				userSvc,
				subject.Key,
				[]access.Action{access.ActionUpdate},
				role.ResourceID(roleOf(ctx, u.Key)),
			))
			MustSucceed(del(fctx, apiuser.DeleteRequest{Keys: []user.Key{u.Key}}))
			Expect(login(ctx, u.Name, "p")).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should refuse to delete the root user", func(ctx SpecContext) {
			fctx, u := nonRootCtx(ctx)
			grantOnUsers(ctx, u, access.ActionDelete)
			Expect(del(fctx, apiuser.DeleteRequest{
				Keys: []user.Key{root.Key},
			})).Error().To(MatchError(ContainSubstring("cannot delete root user")))
			Expect(
				userSvc.NewRetrieve().Where(user.MatchKeys(root.Key)).Exists(ctx, db),
			).To(BeTrue())
		})
		It(
			"Should not reveal the root user to a subject that cannot delete it",
			func(ctx SpecContext) {
				fctx, u := nonRootCtx(ctx)
				Expect(del(fctx, apiuser.DeleteRequest{
					Keys: []user.Key{root.Key},
				})).Error().To(SatisfyAll(
					denied(u, access.ActionDelete, root.ResourceID()),
					Not(MatchError(ContainSubstring("root user"))),
				))
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
