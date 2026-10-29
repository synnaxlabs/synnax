// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/access"
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

var _ = Describe("API keys", func() {
	var (
		bob, eng           user.User
		bobToken, engToken string
		rootToken          string
		rack               ontology.ID
	)
	login := func(ctx context.Context, proof auth.Proof) string {
		GinkgoHelper()
		return MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: proof},
		)).Token
	}
	create := func(
		ctx context.Context,
		token string,
		subject ontology.ID,
	) (apiauth.KeyCreateResponse, error) {
		return apiSvc.CreateKey(
			authed(ctx, token),
			nil,
			apiauth.KeyCreateRequest{Subject: subject, Name: "ci"},
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
	Describe("CreateKey", func() {
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
	Describe("RetrieveKeys", func() {
		It("Should list the caller's own keys by default", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, bobToken, ontology.ID{})).Key
			MustSucceed(create(ctx, rootToken, ontology.ID{}))
			res := MustSucceed(apiSvc.RetrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{},
			))
			Expect(res.Keys).To(ConsistOf(own))
		})
		It("Should deny a user listing another subject's keys", func(ctx SpecContext) {
			Expect(apiSvc.RetrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{Subjects: []ontology.ID{root.OntologyID()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
	Describe("DeleteKeys", func() {
		del := func(ctx context.Context, token string, keys ...apikey.Key) error {
			_, err := apiSvc.DeleteKeys(
				authed(ctx, token),
				nil,
				apiauth.KeyDeleteRequest{Keys: keys},
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
			Expect(apiSvc.CreateSession(
				requestCtx(ctx),
				apiauth.SessionCreateRequest{Proof: keyProof(res.Token)},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should deny a user deleting another subject's key", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, ontology.ID{}))
			Expect(del(ctx, bobToken, res.Key.Key)).To(MatchError(access.ErrDenied))
			MustSucceed(apiSvc.CreateSession(
				requestCtx(ctx),
				apiauth.SessionCreateRequest{Proof: keyProof(res.Token)},
			))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			Expect(del(ctx, bobToken, uuid.New())).To(Succeed())
		})
	})
})
