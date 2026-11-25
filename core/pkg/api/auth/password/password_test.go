// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package password_test

import (
	"context"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apipassword "github.com/synnaxlabs/synnax/pkg/api/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apipassword.ServiceConfig {
		return apipassword.ServiceConfig{
			DB:       &gorp.DB{},
			Auth:     &auth.Service{},
			Password: &password.Service{},
			User:     &user.Service{},
			RBAC:     &rbac.Service{},
		}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(unset func(*apipassword.ServiceConfig), msg string) {
				cfg := full()
				unset(&cfg)
				Expect(cfg.Validate()).To(MatchError(msg))
			},
			Entry(
				"db",
				func(c *apipassword.ServiceConfig) { c.DB = nil },
				"db: must be non-nil",
			),
			Entry(
				"auth",
				func(c *apipassword.ServiceConfig) { c.Auth = nil },
				"auth: must be non-nil",
			),
			Entry(
				"password",
				func(c *apipassword.ServiceConfig) { c.Password = nil },
				"password: must be non-nil",
			),
			Entry(
				"user",
				func(c *apipassword.ServiceConfig) { c.User = nil },
				"user: must be non-nil",
			),
			Entry(
				"rbac",
				func(c *apipassword.ServiceConfig) { c.RBAC = nil },
				"rbac: must be non-nil",
			),
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apipassword.ServiceConfig{}.Override(cfg)
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.Password).To(BeIdenticalTo(cfg.Password))
			Expect(merged.User).To(BeIdenticalTo(cfg.User))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apipassword.ServiceConfig{})
			Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.Password).To(BeIdenticalTo(cfg.Password))
			Expect(merged.User).To(BeIdenticalTo(cfg.User))
			Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		})
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apipassword.NewService(apipassword.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("db: must be non-nil")))
	})
})

var _ = Describe("Disabled password method", func() {
	var (
		own      auth.Credentials
		disabled *apipassword.Service
	)
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		createUser(ctx, own.Username, own.Password)
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		disabled = newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodAPIKey: apiKeySvc,
			},
		})))
	})
	It("Should refuse a password change and keep the password", func(ctx SpecContext) {
		Expect(disabled.PrepareChange(
			requestCtx(ctx),
			apipassword.ChangeRequest{Credentials: own, NewPassword: "new"},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to set a password and keep the old one", func(ctx SpecContext) {
		Expect(disabled.PrepareSet(
			requestCtx(ctx),
			apipassword.SetRequest{Password: "new", CurrentPassword: "old"},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to dispatch to a password", func(ctx SpecContext) {
		Expect(disabled.Dispatch(
			requestCtx(ctx),
			nil,
			apipassword.DispatchRequest{Key: uuid.New()},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse to create a password", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(disabled.PrepareCreate(
			requestCtx(ctx),
			apipassword.CreateRequest{
				Subject:  u.ResourceID(),
				Username: u.Name, Password: "p",
			},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("Change", func() {
	var (
		own      auth.Credentials
		ownToken string
	)
	change := func(
		ctx context.Context,
		token string,
		creds auth.Credentials,
	) error {
		_, err := changePassword(
			authed(ctx, token),
			apipassword.ChangeRequest{Credentials: creds, NewPassword: "new"},
		)
		return err
	}
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: "own-" + uuid.New().String(), Password: "old"}
		createUser(ctx, own.Username, own.Password)
		ownToken = login(ctx, passwordProof(own.Username, own.Password))
	})
	It("Should replace the password when the current one is right", func(
		ctx SpecContext,
	) {
		Expect(change(ctx, ownToken, own)).To(Succeed())
		login(ctx, passwordProof(own.Username, "new"))
		Expect(authSvc.Login(ctx, passwordProof(own.Username, own.Password))).Error().
			To(MatchError(auth.ErrInvalidCredentials))
	})
	It("Should keep the password when the current one is wrong", func(ctx SpecContext) {
		wrong := auth.Credentials{Username: own.Username, Password: "wrong"}
		Expect(change(ctx, ownToken, wrong)).To(MatchError(auth.ErrInvalidCredentials))
		login(ctx, passwordProof(own.Username, own.Password))
	})
	It(
		"Should end every session the password opened, the caller's included",
		func(ctx SpecContext) {
			other := login(ctx, passwordProof(own.Username, own.Password))
			Expect(change(ctx, ownToken, own)).To(Succeed())
			for _, token := range []string{ownToken, other} {
				Expect(authSvc.Authenticate(ctx, token)).Error().
					To(MatchError(auth.ErrInvalidToken))
			}
		},
	)
	It("Should deny a caller that may not update the user of the credentials", func(
		ctx SpecContext,
	) {
		bob := createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken := login(ctx, passwordProof(bob.Name, "p"))
		Expect(change(ctx, bobToken, own)).To(MatchError(access.ErrDenied))
		login(ctx, passwordProof(own.Username, own.Password))
	})
})

var _ = Describe("Set", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	set := func(
		ctx context.Context,
		token string,
		req apipassword.SetRequest,
	) (apipassword.SetResponse, error) {
		return setPassword(authed(ctx, token), req)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, passwordProof(bob.Name, "old"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})
	It("Should replace the caller's session when it sets its own password", func(
		ctx SpecContext,
	) {
		other := login(ctx, passwordProof(bob.Name, "old"))
		res := MustSucceed(set(ctx, bobToken, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "old",
		}))
		Expect(res.Session).ToNot(BeNil())
		Expect(res.Session.Subject).To(Equal(bob.ResourceID()))
		Expect(authSvc.Authenticate(ctx, res.Token)).To(Equal(*res.Session))
		for _, token := range []string{bobToken, other} {
			Expect(authSvc.Authenticate(ctx, token)).Error().
				To(MatchError(auth.ErrInvalidToken))
		}
		login(ctx, passwordProof(bob.Name, "new"))
	})
	It("Should put the caller's new token in a cookie when asked", func(
		ctx SpecContext,
	) {
		fctx := authed(ctx, bobToken)
		res := MustSucceed(setPassword(fctx, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "old",
			Cookie:          true,
		}))
		Expect(res.Token).To(BeEmpty())
		v := MustBeOk(fctx.Get(apiauth.SessionCookieParam))
		cookie, ok := v.(apiauth.SessionCookie)
		Expect(ok).To(BeTrue())
		Expect(authSvc.Authenticate(ctx, cookie.Token)).To(Equal(*res.Session))
	})
	It("Should open no session for a caller logged in with an API key", func(
		ctx SpecContext,
	) {
		keyToken := login(ctx, keyProof(createKey(ctx, bob.ResourceID())))
		res := MustSucceed(set(ctx, keyToken, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "old",
		}))
		Expect(res).To(Equal(apipassword.SetResponse{}))
		MustSucceed(authSvc.Authenticate(ctx, keyToken))
		Expect(authSvc.Authenticate(ctx, bobToken)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It("Should refuse the caller's own password with a wrong current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "wrong",
		})).Error().To(MatchError(auth.ErrInvalidCredentials))
		login(ctx, passwordProof(bob.Name, "old"))
	})
	It("Should refuse the caller's own password without the current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{Password: "new"})).
			Error().To(MatchError(ContainSubstring("password: required")))
		login(ctx, passwordProof(bob.Name, "old"))
	})
	It("Should let the owner set another user's password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  bob.ResourceID(),
			Password: "new",
		})).To(Equal(apipassword.SetResponse{}))
		MustSucceed(authSvc.Authenticate(ctx, rootToken))
		login(ctx, passwordProof(bob.Name, "new"))
		Expect(authSvc.Authenticate(ctx, bobToken)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It(
		"Should let a caller that may update a signed-out user set its password",
		func(ctx SpecContext) {
			eng := createUser(ctx, "eng-"+uuid.New().String(), "p")
			grant(ctx, bob.Key, access.ActionUpdate, eng.ResourceID())
			Expect(set(ctx, bobToken, apipassword.SetRequest{
				Subject:  eng.ResourceID(),
				Password: "new",
			})).To(Equal(apipassword.SetResponse{}))
			login(ctx, passwordProof(eng.Name, "new"))
		},
	)
	It(
		"Should require delete on a signed-in user to set its password",
		func(ctx SpecContext) {
			eng := createUser(ctx, "eng-"+uuid.New().String(), "p")
			engToken := login(ctx, passwordProof(eng.Name, "p"))
			grant(ctx, bob.Key, access.ActionUpdate, eng.ResourceID())
			req := apipassword.SetRequest{
				Subject:  eng.ResourceID(),
				Password: "new",
			}
			Expect(set(ctx, bobToken, req)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("cannot delete %s", eng.ResourceID())),
			))
			MustSucceed(authSvc.Authenticate(ctx, engToken))
			grant(ctx, bob.Key, access.ActionDelete, eng.ResourceID())
			Expect(set(ctx, bobToken, req)).To(Equal(apipassword.SetResponse{}))
			Expect(authSvc.Authenticate(ctx, engToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
		},
	)
	It("Should deny a user setting another user's password", func(ctx SpecContext) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{
			Subject:  root.ResourceID(),
			Password: "new",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  resource.ID{Type: resource.TypeRack, Key: "1"},
			Password: "new",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user with no password as not found", func(ctx SpecContext) {
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  u.ResourceID(),
			Password: "new",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject: bob.ResourceID(),
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
})

var _ = Describe("Create", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
		nopass    user.User
	)
	create := func(
		ctx context.Context,
		token string,
		req apipassword.CreateRequest,
	) (struct{}, error) {
		return createPassword(authed(ctx, token), req)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, passwordProof(bob.Name, "old"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
		nopass = MustSucceed(rbactestutil.CreateUser(
			ctx,
			db,
			rbacSvc,
			userSvc,
			user.User{Name: "nopass-" + uuid.New().String()},
		))
	})
	It("Should let the owner give a user its first password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: nopass.Name, Password: "p",
		})).Error().ToNot(HaveOccurred())
		login(ctx, passwordProof(nopass.Name, "p"))
	})
	It("Should let a user give itself its first password", func(ctx SpecContext) {
		token := login(ctx, keyProof(createKey(ctx, nopass.ResourceID())))
		Expect(create(ctx, token, apipassword.CreateRequest{
			Username: nopass.Name, Password: "p",
		})).Error().ToNot(HaveOccurred())
		login(ctx, passwordProof(nopass.Name, "p"))
	})
	It("Should deny a user giving another user a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: nopass.Name, Password: "p",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should deny a caller that may only create users", func(ctx SpecContext) {
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			bob.Key,
			[]access.Action{access.ActionCreate},
			resource.ID{Type: resource.TypeUser},
		))
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: nopass.Name, Password: "p",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should deny a caller that may only update users", func(ctx SpecContext) {
		MustSucceed(rbactestutil.Grant(
			ctx,
			db,
			rbacSvc,
			userSvc,
			bob.Key,
			[]access.Action{access.ActionUpdate},
			resource.ID{Type: resource.TypeUser},
		))
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: nopass.Name, Password: "p",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse a user named by a key in another case", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject: resource.ID{
				Type: resource.TypeUser,
				Key:  strings.ToUpper(bob.Key.String()),
			},
			Username: "other", Password: "p",
		})).Error().To(MatchError(ContainSubstring("already has a password")))
	})
	It("Should refuse a user that already has a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Username: "other", Password: "new",
		})).Error().To(MatchError(ContainSubstring("already has a password")))
	})
	It("Should refuse a username another user holds", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: creds.Username, Password: "p",
		})).Error().To(MatchError(auth.ErrRepeatedUsername))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  resource.ID{Type: resource.TypeRack, Key: "1"},
			Username: "rack", Password: "p",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user that does not exist as not found", func(
		ctx SpecContext,
	) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  user.ResourceID(uuid.New()),
			Username: "ghost", Password: "p",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Username: nopass.Name,
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
	It("Should reject an empty username", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.ResourceID(),
			Password: "p",
		})).Error().To(MatchError(ContainSubstring("username: required")))
	})
})

var _ = Describe("Passwords", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	keyOf := func(ctx context.Context, subject resource.ID) password.Key {
		GinkgoHelper()
		var p password.Password
		Expect(passwordSvc.NewRetrieve().
			Where(password.MatchSubjects(subject)).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		return p.Key
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
	})
	Describe("Retrieve", func() {
		retrieve := func(
			ctx context.Context,
			token string,
			req apipassword.RetrieveRequest,
		) ([]apipassword.Password, error) {
			res, err := retrievePasswords(authed(ctx, token), req)
			return res.Passwords, err
		}
		It("Should return the caller's password by default", func(ctx SpecContext) {
			Expect(retrieve(ctx, bobToken, apipassword.RetrieveRequest{})).
				To(ConsistOf(apipassword.Password{
					Key:      keyOf(ctx, bob.ResourceID()),
					Subject:  bob.ResourceID(),
					Username: bob.Name,
				}))
		})
		It("Should find a password by its username", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Usernames: []string{bob.Name},
			})).To(ConsistOf(HaveField("Subject", bob.ResourceID())))
		})
		It("Should find a password by its subject", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Subjects: []resource.ID{bob.ResourceID()},
			})).To(ConsistOf(HaveField("Username", bob.Name)))
		})
		It("Should ignore an unknown username", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Usernames: []string{"unknown-" + uuid.New().String()},
			})).To(BeEmpty())
		})
		It("Should deny a user another user's password by subject", func(
			ctx SpecContext,
		) {
			Expect(retrieve(ctx, bobToken, apipassword.RetrieveRequest{
				Subjects: []resource.ID{root.ResourceID()},
			})).Error().To(MatchError(access.ErrDenied))
		})
		It("Should leave out another user's password by username", func(
			ctx SpecContext,
		) {
			Expect(retrieve(ctx, bobToken, apipassword.RetrieveRequest{
				Usernames: []string{creds.Username, bob.Name},
			})).To(ConsistOf(HaveField("Subject", bob.ResourceID())))
		})
	})
	Describe("Dispatch", func() {
		rename := func(
			ctx context.Context,
			token string,
			key password.Key,
			username string,
		) error {
			action := password.NewSetUsernameAction(
				password.SetUsernamePayload{Username: username},
			)
			_, err := dispatchPassword(
				authed(ctx, token),
				apipassword.DispatchRequest{
					Key:     key,
					Actions: []password.Action{action},
				},
			)
			return err
		}
		It("Should let the owner rename another user's password", func(
			ctx SpecContext,
		) {
			username := "renamed-" + uuid.New().String()
			Expect(rename(ctx, rootToken, keyOf(ctx, bob.ResourceID()), username)).
				To(Succeed())
			login(ctx, passwordProof(username, "p"))
			MustSucceed(authSvc.Authenticate(ctx, bobToken))
			Expect(authSvc.Login(ctx, passwordProof(bob.Name, "p"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should let a user that may update itself rename its own password", func(
			ctx SpecContext,
		) {
			MustSucceed(rbactestutil.Grant(
				ctx,
				db,
				rbacSvc,
				userSvc,
				bob.Key,
				[]access.Action{access.ActionUpdate},
				bob.ResourceID(),
			))
			username := "self-" + uuid.New().String()
			Expect(rename(ctx, bobToken, keyOf(ctx, bob.ResourceID()), username)).
				To(Succeed())
			login(ctx, passwordProof(username, "p"))
		})
		It("Should deny a self-rename without update on the user", func(
			ctx SpecContext,
		) {
			Expect(rename(
				ctx,
				bobToken,
				keyOf(ctx, bob.ResourceID()),
				"self-"+uuid.New().String(),
			)).To(MatchError(access.ErrDenied))
			login(ctx, passwordProof(bob.Name, "p"))
		})
		It("Should keep the username when it does not change", func(ctx SpecContext) {
			Expect(rename(ctx, rootToken, keyOf(ctx, bob.ResourceID()), bob.Name)).
				To(Succeed())
			login(ctx, passwordProof(bob.Name, "p"))
		})
		It("Should refuse a username another user holds", func(ctx SpecContext) {
			Expect(rename(
				ctx,
				rootToken,
				keyOf(ctx, bob.ResourceID()),
				creds.Username,
			)).To(MatchError(auth.ErrRepeatedUsername))
		})
		It("Should deny a user renaming another user's password", func(
			ctx SpecContext,
		) {
			Expect(rename(
				ctx,
				bobToken,
				keyOf(ctx, root.ResourceID()),
				"taken-"+uuid.New().String(),
			)).To(MatchError(access.ErrDenied))
			login(ctx, passwordProof(creds.Username, creds.Password))
		})
		It("Should report a password that does not exist as not found", func(
			ctx SpecContext,
		) {
			Expect(rename(ctx, rootToken, uuid.New(), "nopass-"+uuid.New().String())).
				To(MatchError(query.ErrNotFound))
		})
		It("Should reject an empty username", func(ctx SpecContext) {
			Expect(rename(ctx, rootToken, keyOf(ctx, bob.ResourceID()), "")).
				To(MatchError(validate.ErrValidation))
		})
	})
})
