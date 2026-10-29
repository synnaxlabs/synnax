// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiapikey "github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func keyProof(key string) auth.Proof {
	return auth.Proof{Variant: auth.APIKeyProof{
		APIKeyCredentials: auth.APIKeyCredentials{Key: key},
	}}
}

func assignBuiltin(ctx context.Context, subject ontology.ID, name string) {
	GinkgoHelper()
	var r role.Role
	Expect(rbacSvc.Role.NewRetrieve().
		Where(role.Match(func(_ gorp.Context, _ role.Retrieve, r *role.Role) (bool, error) {
			return r.Name == name, nil
		})).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	Expect(rbacSvc.Role.NewWriter(nil, true).AssignRole(ctx, subject, r.Key)).
		To(Succeed())
}

func grant(
	ctx context.Context,
	subject ontology.ID,
	action access.Action,
	object ontology.ID,
) {
	GinkgoHelper()
	r := &role.Role{Name: string(action) + "-" + uuid.New().String()}
	Expect(rbacSvc.Role.NewWriter(nil, true).Create(ctx, r)).To(Succeed())
	p := &policy.Policy{
		Name:    string(action) + "-policy-" + uuid.New().String(),
		Objects: []ontology.ID{object},
		Actions: []access.Action{action},
	}
	policies := rbacSvc.Policy.NewWriter(nil, true)
	Expect(policies.Create(ctx, p)).To(Succeed())
	Expect(policies.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(
		rbacSvc.Role.NewWriter(nil, true).AssignRole(ctx, subject, r.Key),
	).To(Succeed())
}

var _ = Describe("ServiceConfig", func() {
	full := func() apiapikey.ServiceConfig {
		return apiapikey.ServiceConfig{
			Auth:     &auth.Service{},
			APIKey:   &apikey.Service{},
			Ontology: &ontology.Ontology{},
			RBAC:     &rbac.Service{},
		}
	}
	It("Should succeed when every service is set", func() {
		Expect(full().Validate()).To(Succeed())
	})
	DescribeTable(
		"Should fail when a service is missing",
		func(clear func(*apiapikey.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry(
			"auth",
			func(c *apiapikey.ServiceConfig) { c.Auth = nil },
			"auth: must be non-nil",
		),
		Entry(
			"api key",
			func(c *apiapikey.ServiceConfig) { c.APIKey = nil },
			"api_key: must be non-nil",
		),
		Entry(
			"ontology",
			func(c *apiapikey.ServiceConfig) { c.Ontology = nil },
			"ontology: must be non-nil",
		),
		Entry(
			"rbac",
			func(c *apiapikey.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil",
		),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		merged := apiapikey.ServiceConfig{}.Override(cfg)
		Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
		Expect(merged.APIKey).To(BeIdenticalTo(cfg.APIKey))
		Expect(merged.Ontology).To(BeIdenticalTo(cfg.Ontology))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("NewService", func() {
	It("Should return an error when the config is invalid", func() {
		Expect(apiapikey.NewService(apiapikey.ServiceConfig{})).Error().
			To(MatchError(ContainSubstring("auth: must be non-nil")))
	})
})

var _ = Describe("Disabled API key method", func() {
	It("Should refuse a new API key", func(ctx SpecContext) {
		disabled := newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: node.DB,
			Authenticators: map[auth.Method]auth.Authenticator{
				auth.MethodPassword: passwordSvc,
			},
		})))
		Expect(disabled.Create(
			requestCtx(ctx),
			nil,
			apiapikey.CreateRequest{Name: "ci"},
		)).Error().To(MatchError(validate.ErrValidation))
	})
})

var _ = Describe("API keys", func() {
	var (
		bob, eng           user.User
		bobToken, engToken string
		rootToken          string
		rack               ontology.ID
	)
	create := func(
		ctx context.Context,
		token string,
		subject ontology.ID,
	) (apiapikey.CreateResponse, error) {
		return apiSvc.Create(
			authed(ctx, token),
			nil,
			apiapikey.CreateRequest{Subject: subject, Name: "ci"},
		)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		eng = createUser(ctx, "eng-"+uuid.New().String(), "p")
		assignBuiltin(ctx, eng.OntologyID(), "Engineer")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		engToken = login(ctx, passwordProof(eng.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
		rack = ontology.ID{Type: ontology.ResourceTypeRack, Key: uuid.New().String()}
		Expect(otg.NewWriter(nil).DefineResources(ctx, rack)).To(Succeed())
	})
	Describe("Create", func() {
		It("Should create a key for the caller that logs in as the caller", func(
			ctx SpecContext,
		) {
			res := MustSucceed(create(ctx, bobToken, ontology.ID{}))
			Expect(res.Token).To(HavePrefix("syk_"))
			Expect(res.Key.Subject).To(Equal(bob.OntologyID()))
			Expect(res.Key.Name).To(Equal("ci"))
			session := MustSucceed(authSvc.Authenticate(
				ctx,
				login(ctx, keyProof(res.Token)),
			))
			Expect(session.Subject).To(Equal(bob.OntologyID()))
			Expect(session.Credential).To(Equal(res.Key.Key))
		})
		It("Should let an Engineer create a key for a rack", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, engToken, rack))
			session := MustSucceed(authSvc.Authenticate(
				ctx,
				login(ctx, keyProof(res.Token)),
			))
			Expect(session.Subject).To(Equal(rack))
		})
		It("Should deny an Engineer a key for another user", func(ctx SpecContext) {
			Expect(create(ctx, engToken, bob.OntologyID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a key for a user the caller may only update", func(
			ctx SpecContext,
		) {
			grant(ctx, bob.OntologyID(), access.ActionUpdate, eng.OntologyID())
			Expect(create(ctx, bobToken, eng.OntologyID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a key for a rack the caller may only update", func(
			ctx SpecContext,
		) {
			grant(
				ctx,
				bob.OntologyID(),
				access.ActionUpdate,
				ontology.ID{Type: ontology.ResourceTypeRack},
			)
			Expect(create(ctx, bobToken, rack)).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a user without a role a key for a rack", func(ctx SpecContext) {
			Expect(create(ctx, bobToken, rack)).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should let the Owner create a key for another user", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, bob.OntologyID()))
			Expect(res.Key.Subject).To(Equal(bob.OntologyID()))
		})
		It("Should reject a subject that does not exist", func(ctx SpecContext) {
			missing := ontology.ID{Type: ontology.ResourceTypeRack, Key: "missing"}
			Expect(create(ctx, rootToken, missing)).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should reject a subject that is not a user or rack", func(ctx SpecContext) {
			channel := ontology.ID{Type: ontology.ResourceTypeChannel, Key: "1"}
			Expect(create(ctx, rootToken, channel)).Error().
				To(MatchError(validate.ErrValidation))
		})
	})
	Describe("Retrieve", func() {
		It("Should list the caller's own keys by default", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, bobToken, ontology.ID{})).Key
			MustSucceed(create(ctx, rootToken, ontology.ID{}))
			res := MustSucceed(apiSvc.Retrieve(
				authed(ctx, bobToken),
				apiapikey.RetrieveRequest{},
			))
			Expect(res.Keys).To(ConsistOf(own))
		})
		It("Should deny a user listing another subject's keys", func(ctx SpecContext) {
			Expect(apiSvc.Retrieve(
				authed(ctx, bobToken),
				apiapikey.RetrieveRequest{
					Subjects: []ontology.ID{root.OntologyID()},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
	Describe("Delete", func() {
		del := func(ctx context.Context, token string, keys ...apikey.Key) error {
			_, err := apiSvc.Delete(
				authed(ctx, token),
				nil,
				apiapikey.DeleteRequest{Keys: keys},
			)
			return err
		}
		It("Should delete the key and end the sessions it opened", func(
			ctx SpecContext,
		) {
			res := MustSucceed(create(ctx, bobToken, ontology.ID{}))
			keyToken := login(ctx, keyProof(res.Token))
			Expect(del(ctx, bobToken, res.Key.Key)).To(Succeed())
			Expect(authSvc.Authenticate(ctx, keyToken)).Error().
				To(MatchError(auth.ErrInvalidToken))
			MustSucceed(authSvc.Authenticate(ctx, bobToken))
			Expect(authSvc.Login(ctx, keyProof(res.Token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should deny a user deleting another subject's key", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, ontology.ID{}))
			Expect(del(ctx, bobToken, res.Key.Key)).To(MatchError(access.ErrDenied))
			login(ctx, keyProof(res.Token))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(del(ctx, bobToken, uuid.New())).To(Succeed())
		})
	})
})
