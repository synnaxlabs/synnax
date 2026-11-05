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
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
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

func assignBuiltin(ctx context.Context, subject user.Key, name string) {
	GinkgoHelper()
	var r role.Role
	Expect(rbacSvc.Role.NewRetrieve().
		Where(role.Match(func(_ gorp.Context, _ role.Retrieve, r *role.Role) (bool, error) {
			return r.Name == name, nil
		})).
		Entry(&r).
		Exec(ctx, nil)).To(Succeed())
	Expect(userSvc.NewWriter(nil).AssignRole(ctx, r.Key, subject)).To(Succeed())
}

// grant grants the action on the object to the subject through a new role.
func grant(
	ctx context.Context,
	subject user.Key,
	action access.Action,
	object resource.ID,
) {
	GinkgoHelper()
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, rbacSvc, userSvc, subject, actions, object))
}

var _ = Describe("API keys", func() {
	var (
		bob, eng           user.User
		bobToken, engToken string
		rootToken          string
		rackID             resource.ID
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
		subject resource.ID,
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
		assignBuiltin(ctx, eng.Key, "Engineer")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		engToken = login(ctx, passwordProof(eng.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
		r := rack.Rack{Name: "rack-" + uuid.New().String()}
		Expect(rackSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		rackID = r.ResourceID()
	})
	Describe("CreateKey", func() {
		It("Should create a key for the caller that logs in as the caller", func(
			ctx SpecContext,
		) {
			res := MustSucceed(create(ctx, bobToken, resource.ID{}))
			Expect(res.Token).To(HavePrefix("syk_"))
			Expect(res.Key.Subject).To(Equal(bob.ResourceID()))
			Expect(res.Key.Name).To(Equal("ci"))
			session := MustSucceed(authSvc.Authenticate(
				ctx,
				login(ctx, keyProof(res.Token)),
			))
			Expect(session.Subject).To(Equal(bob.ResourceID()))
			Expect(session.Credential).To(Equal(res.Key.Key))
		})
		It("Should let an Engineer create a key for a rack", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, engToken, rackID))
			session := MustSucceed(authSvc.Authenticate(
				ctx,
				login(ctx, keyProof(res.Token)),
			))
			Expect(session.Subject).To(Equal(rackID))
		})
		It("Should deny an Engineer a key for another user", func(ctx SpecContext) {
			Expect(create(ctx, engToken, bob.ResourceID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a key for a user the caller may only update", func(
			ctx SpecContext,
		) {
			grant(ctx, bob.Key, access.ActionUpdate, eng.ResourceID())
			Expect(create(ctx, bobToken, eng.ResourceID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a key for a rack the caller may only update", func(
			ctx SpecContext,
		) {
			grant(
				ctx,
				bob.Key,
				access.ActionUpdate,
				resource.ID{Type: resource.TypeRack},
			)
			Expect(create(ctx, bobToken, rackID)).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a user without a role a key for a rack", func(ctx SpecContext) {
			Expect(create(ctx, bobToken, rackID)).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should let the Owner create a key for another user", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, bob.ResourceID()))
			Expect(res.Key.Subject).To(Equal(bob.ResourceID()))
		})
		It("Should reject a subject that does not exist", func(ctx SpecContext) {
			missing := resource.ID{Type: resource.TypeRack, Key: "4000000000"}
			Expect(create(ctx, rootToken, missing)).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should reject a subject with a malformed key", func(ctx SpecContext) {
			malformed := resource.ID{Type: resource.TypeRack, Key: "missing"}
			Expect(create(ctx, rootToken, malformed)).Error().
				To(MatchError(validate.ErrValidation))
		})
		It("Should reject a subject that is not a user or rack", func(ctx SpecContext) {
			channel := resource.ID{Type: resource.TypeChannel, Key: "1"}
			Expect(create(ctx, rootToken, channel)).Error().
				To(MatchError(validate.ErrValidation))
		})
	})
	Describe("RetrieveKeys", func() {
		It("Should list the caller's own keys by default", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, bobToken, resource.ID{})).Key
			MustSucceed(create(ctx, rootToken, resource.ID{}))
			res := MustSucceed(apiSvc.RetrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{},
			))
			Expect(res.Keys).To(ConsistOf(own))
		})
		It("Should deny a user listing another subject's keys", func(ctx SpecContext) {
			Expect(apiSvc.RetrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{Subjects: []resource.ID{root.ResourceID()}},
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
			res := MustSucceed(create(ctx, bobToken, resource.ID{}))
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
			res := MustSucceed(create(ctx, rootToken, resource.ID{}))
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
