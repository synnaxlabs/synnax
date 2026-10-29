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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apipassword "github.com/synnaxlabs/synnax/pkg/api/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ServiceConfig", func() {
	full := func() apipassword.ServiceConfig {
		return apipassword.ServiceConfig{
			Auth:     &auth.Service{},
			Password: &password.Service{},
			Ontology: &ontology.Ontology{},
			RBAC:     &rbac.Service{},
		}
	}
	It("Should succeed when every service is set", func() {
		Expect(full().Validate()).To(Succeed())
	})
	DescribeTable(
		"Should fail when a service is missing",
		func(clear func(*apipassword.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
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
			"ontology",
			func(c *apipassword.ServiceConfig) { c.Ontology = nil },
			"ontology: must be non-nil",
		),
		Entry(
			"rbac",
			func(c *apipassword.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil",
		),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		merged := apipassword.ServiceConfig{}.Override(cfg)
		Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
		Expect(merged.Password).To(BeIdenticalTo(cfg.Password))
		Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apipassword.NewService(apipassword.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
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
		disabled = newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: node.DB,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodAPIKey: apiKeySvc,
			},
		})))
	})
	It("Should refuse a password change and keep the password", func(ctx SpecContext) {
		Expect(disabled.Change(
			requestCtx(ctx),
			nil,
			apipassword.ChangeRequest{Credentials: own, NewPassword: "new"},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to set a password and keep the old one", func(ctx SpecContext) {
		Expect(disabled.Set(
			requestCtx(ctx),
			nil,
			apipassword.SetRequest{Password: "new", CurrentPassword: own.Password},
		)).Error().To(MatchError(validate.ErrValidation))
		MustSucceed(passwordSvc.Authenticate(ctx, passwordProof(own.Username, "old")))
	})
	It("Should refuse to rename a password", func(ctx SpecContext) {
		Expect(disabled.Rename(
			requestCtx(ctx),
			nil,
			apipassword.RenameRequest{Username: "off"},
		)).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should refuse to create a password", func(ctx SpecContext) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(disabled.Create(
			requestCtx(ctx),
			nil,
			apipassword.CreateRequest{
				Subject:  u.OntologyID(),
				Username: u.Name, Password: "p",
			},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("ChangePassword", func() {
	var own auth.Credentials
	BeforeEach(func(ctx SpecContext) {
		own = auth.Credentials{Username: uuid.New().String(), Password: "old"}
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: own.Username,
		}))
		Expect(passwordSvc.NewWriter(nil).Set(ctx, u.OntologyID(), own)).To(Succeed())
	})
	It(
		"Should replace the password when the current one is right",
		func(ctx SpecContext) {
			Expect(
				apiSvc.Change(
					requestCtx(ctx),
					nil,
					apipassword.ChangeRequest{
						Credentials: own,
						NewPassword: "new",
					},
				),
			).Error().
				ToNot(HaveOccurred())
			login(ctx, own.Username, "new")
			Expect(authSvc.Login(ctx, passwordProof(own.Username, own.Password))).
				Error().To(MatchError(auth.ErrInvalidCredentials))
		},
	)
	It("Should keep the password when the current one is wrong", func(ctx SpecContext) {
		creds := auth.Credentials{Username: own.Username, Password: "wrong"}
		Expect(
			apiSvc.Change(requestCtx(ctx), nil, apipassword.ChangeRequest{
				Credentials: creds,
				NewPassword: "new",
			}),
		).Error().
			To(MatchError(auth.ErrInvalidCredentials))
		login(ctx, own.Username, own.Password)
	})
	It(
		"Should end the password's other sessions and keep the caller's",
		func(ctx SpecContext) {
			kept := login(ctx, own.Username, own.Password)
			other := login(ctx, own.Username, own.Password)
			fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
			fctx.Set("Session", MustSucceed(authSvc.Authenticate(ctx, kept)))
			Expect(apiSvc.Change(fctx, nil, apipassword.ChangeRequest{
				Credentials: own,
				NewPassword: "new",
			})).Error().ToNot(HaveOccurred())
			MustSucceed(authSvc.Authenticate(ctx, kept))
			Expect(authSvc.Authenticate(ctx, other)).Error().
				To(MatchError(auth.ErrInvalidToken))
		},
	)
})

var _ = Describe("SetPassword", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	set := func(
		ctx context.Context,
		token string,
		req apipassword.SetRequest,
	) (struct{}, error) {
		return apiSvc.Set(authed(ctx, token), nil, req)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, bob.Name, "old")
		rootToken = login(ctx, creds.Username, creds.Password)
	})
	It("Should set the caller's own password given the current one", func(
		ctx SpecContext,
	) {
		other := login(ctx, bob.Name, "old")
		Expect(set(ctx, bobToken, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "old",
		})).Error().ToNot(HaveOccurred())
		login(ctx, bob.Name, "new")
		MustSucceed(authSvc.Authenticate(ctx, bobToken))
		Expect(authSvc.Authenticate(ctx, other)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It("Should refuse the caller's own password with a wrong current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{
			Password:        "new",
			CurrentPassword: "wrong",
		})).Error().To(MatchError(auth.ErrInvalidCredentials))
		login(ctx, bob.Name, "old")
	})
	It("Should refuse the caller's own password without the current one", func(
		ctx SpecContext,
	) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{Password: "new"})).
			Error().To(MatchError(ContainSubstring("password: required")))
		login(ctx, bob.Name, "old")
	})
	It("Should let the owner set another user's password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  bob.OntologyID(),
			Password: "new",
		})).Error().ToNot(HaveOccurred())
		login(ctx, bob.Name, "new")
		Expect(authSvc.Authenticate(ctx, bobToken)).Error().
			To(MatchError(auth.ErrInvalidToken))
	})
	It("Should deny a user setting another user's password", func(ctx SpecContext) {
		Expect(set(ctx, bobToken, apipassword.SetRequest{
			Subject:  root.OntologyID(),
			Password: "new",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  ontology.ID{Type: ontology.ResourceTypeRack, Key: "1"},
			Password: "new",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user with no password as not found", func(ctx SpecContext) {
		u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject:  u.OntologyID(),
			Password: "new",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(set(ctx, rootToken, apipassword.SetRequest{
			Subject: bob.OntologyID(),
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
})

var _ = Describe("CreatePassword", func() {
	var (
		bobToken  string
		rootToken string
		nopass    user.User
	)
	create := func(
		ctx context.Context,
		token string,
		req apipassword.CreateRequest,
	) (struct{}, error) {
		return apiSvc.Create(authed(ctx, token), nil, req)
	}
	BeforeEach(func(ctx SpecContext) {
		bob := createUser(ctx, "bob-"+uuid.New().String(), "old")
		bobToken = login(ctx, bob.Name, "old")
		rootToken = login(ctx, creds.Username, creds.Password)
		nopass = MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
	})
	It("Should let the owner give a user its first password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name, Password: "p",
		})).Error().ToNot(HaveOccurred())
		login(ctx, nopass.Name, "p")
	})
	It("Should deny a user giving another user a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name, Password: "p",
		})).Error().To(MatchError(access.ErrDenied))
	})
	It("Should refuse a user that already has a password", func(ctx SpecContext) {
		Expect(create(ctx, bobToken, apipassword.CreateRequest{
			Username: "other", Password: "new",
		})).Error().To(MatchError(ContainSubstring("already has a password")))
	})
	It("Should refuse a username another user holds", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.OntologyID(),
			Username: creds.Username, Password: "p",
		})).Error().To(MatchError(auth.ErrRepeatedUsername))
	})
	It("Should reject a subject that is not a user", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  ontology.ID{Type: ontology.ResourceTypeRack, Key: "1"},
			Username: "rack", Password: "p",
		})).Error().To(MatchError(validate.ErrValidation))
	})
	It("Should report a user that does not exist as not found", func(
		ctx SpecContext,
	) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  user.OntologyID(uuid.New()),
			Username: "ghost", Password: "p",
		})).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject an empty password", func(ctx SpecContext) {
		Expect(create(ctx, rootToken, apipassword.CreateRequest{
			Subject:  nopass.OntologyID(),
			Username: nopass.Name,
		})).Error().To(MatchError(ContainSubstring("password: required")))
	})
})

var _ = Describe("Passwords", func() {
	var (
		bob       user.User
		bobToken  string
		rootToken string
	)
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		bobToken = login(ctx, bob.Name, "p")
		rootToken = login(ctx, creds.Username, creds.Password)
	})
	Describe("RetrievePasswords", func() {
		retrieve := func(
			ctx context.Context,
			token string,
			req apipassword.RetrieveRequest,
		) ([]apipassword.Password, error) {
			res, err := apiSvc.Retrieve(authed(ctx, token), req)
			return res.Passwords, err
		}
		It("Should return the caller's password by default", func(ctx SpecContext) {
			Expect(retrieve(ctx, bobToken, apipassword.RetrieveRequest{})).
				To(ConsistOf(apipassword.Password{
					Subject:  bob.OntologyID(),
					Username: bob.Name,
				}))
		})
		It("Should find a password by its username", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Usernames: []string{bob.Name},
			})).To(ConsistOf(HaveField("Subject", bob.OntologyID())))
		})
		It("Should find a password by its subject", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Subjects: []ontology.ID{bob.OntologyID()},
			})).To(ConsistOf(HaveField("Username", bob.Name)))
		})
		It("Should ignore an unknown username", func(ctx SpecContext) {
			Expect(retrieve(ctx, rootToken, apipassword.RetrieveRequest{
				Usernames: []string{"unknown-" + uuid.New().String()},
			})).To(BeEmpty())
		})
		It("Should deny a user another user's password", func(ctx SpecContext) {
			Expect(retrieve(ctx, bobToken, apipassword.RetrieveRequest{
				Usernames: []string{creds.Username},
			})).Error().To(MatchError(access.ErrDenied))
		})
	})
	Describe("RenamePassword", func() {
		rename := func(
			ctx context.Context,
			token string,
			req apipassword.RenameRequest,
		) (struct{}, error) {
			return apiSvc.Rename(authed(ctx, token), nil, req)
		}
		It("Should let the owner rename another user's password", func(
			ctx SpecContext,
		) {
			username := "renamed-" + uuid.New().String()
			Expect(rename(ctx, rootToken, apipassword.RenameRequest{
				Subject:  bob.OntologyID(),
				Username: username,
			})).Error().ToNot(HaveOccurred())
			login(ctx, username, "p")
			MustSucceed(authSvc.Authenticate(ctx, bobToken))
			Expect(authSvc.Login(ctx, passwordProof(bob.Name, "p"))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should let a user rename its own password", func(ctx SpecContext) {
			username := "self-" + uuid.New().String()
			Expect(rename(ctx, bobToken, apipassword.RenameRequest{
				Username: username,
			})).Error().ToNot(HaveOccurred())
			login(ctx, username, "p")
		})
		It("Should keep the username when it does not change", func(ctx SpecContext) {
			Expect(rename(ctx, bobToken, apipassword.RenameRequest{
				Username: bob.Name,
			})).Error().ToNot(HaveOccurred())
			login(ctx, bob.Name, "p")
		})
		It("Should refuse a username another user holds", func(ctx SpecContext) {
			Expect(rename(ctx, bobToken, apipassword.RenameRequest{
				Username: creds.Username,
			})).Error().To(MatchError(auth.ErrRepeatedUsername))
		})
		It("Should deny a user renaming another user's password", func(
			ctx SpecContext,
		) {
			Expect(rename(ctx, bobToken, apipassword.RenameRequest{
				Subject:  root.OntologyID(),
				Username: "taken-" + uuid.New().String(),
			})).Error().To(MatchError(access.ErrDenied))
		})
		It("Should report a user with no password as not found", func(
			ctx SpecContext,
		) {
			u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
				Name: "nopass",
			}))
			Expect(rename(ctx, rootToken, apipassword.RenameRequest{
				Subject:  u.OntologyID(),
				Username: "nopass-" + uuid.New().String(),
			})).Error().To(MatchError(query.ErrNotFound))
		})
		It("Should reject an empty username", func(ctx SpecContext) {
			Expect(rename(ctx, bobToken, apipassword.RenameRequest{})).Error().
				To(MatchError(validate.ErrValidation))
		})
	})
})
