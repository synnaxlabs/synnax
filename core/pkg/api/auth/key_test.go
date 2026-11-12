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
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func keyProof(key string) auth.Proof {
	return auth.Proof{Variant: auth.APIKeyProof{
		APIKeyCredentials: auth.APIKeyCredentials{Key: key},
	}}
}

func setBuiltinRole(ctx context.Context, subject user.Key, name string) {
	GinkgoHelper()
	var r role.Role
	Expect(rbacSvc.Role.NewRetrieve().
		Where(role.Match(func(_ gorp.Context, _ role.Retrieve, r *role.Role) (bool, error) {
			return r.Name == name, nil
		})).
		Entry(&r).
		Exec(ctx, db)).To(Succeed())
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return userSvc.NewWriter(tx).SetRole(ctx, r.Key, subject)
	})).To(Succeed())
}

// grant grants the action on the object to the subject.
func grant(
	ctx context.Context,
	subject user.Key,
	action access.Action,
	object resource.ID,
) {
	GinkgoHelper()
	actions := []access.Action{action}
	MustSucceed(rbactestutil.Grant(ctx, db, rbacSvc, userSvc, subject, actions, object))
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
		return createKey(
			authed(ctx, token),
			apiauth.KeyCreateRequest{Subject: subject, Name: "ci"},
		)
	}
	BeforeEach(func(ctx SpecContext) {
		bob = createUser(ctx, "bob-"+uuid.New().String(), "p")
		eng = createUser(ctx, "eng-"+uuid.New().String(), "p")
		setBuiltinRole(ctx, eng.Key, "Engineer")
		bobToken = login(ctx, passwordProof(bob.Name, "p"))
		engToken = login(ctx, passwordProof(eng.Name, "p"))
		rootToken = login(ctx, passwordProof(creds.Username, creds.Password))
		r := rack.Rack{Name: "rack-" + uuid.New().String()}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return rackSvc.NewWriter(tx).Create(ctx, &r)
		})).To(Succeed())
		rackID = r.ResourceID()
	})
	It("Should send an empty username for a user with no password", func(
		ctx SpecContext,
	) {
		u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
			Name: "nopass-" + uuid.New().String(),
		}))
		key := MustSucceed(create(ctx, rootToken, u.ResourceID()))
		created := MustSucceed(apiSvc.CreateSession(
			requestCtx(ctx),
			apiauth.SessionCreateRequest{Proof: keyProof(key.Token)},
		))
		expectLegacyUser(*created.User, "")
		renewed := MustSucceed(apiSvc.RenewSession(
			authed(ctx, created.Token),
			struct{}{},
		))
		expectLegacyUser(*renewed.User, "")
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
		It("Should let a caller that may create on a user create a key for it", func(
			ctx SpecContext,
		) {
			grant(ctx, bob.Key, access.ActionCreate, eng.ResourceID())
			res := MustSucceed(create(ctx, bobToken, eng.ResourceID()))
			Expect(res.Key.Subject).To(Equal(eng.ResourceID()))
		})
		It("Should deny a key for a user the caller may only update", func(
			ctx SpecContext,
		) {
			grant(ctx, bob.Key, access.ActionUpdate, eng.ResourceID())
			Expect(create(ctx, bobToken, eng.ResourceID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a user without a role a key for a rack", func(ctx SpecContext) {
			Expect(create(ctx, bobToken, rackID)).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should deny a rack a key for another rack", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, rootToken, rackID))
			rackToken := login(ctx, keyProof(own.Token))
			other := rack.Rack{Name: "rack-" + uuid.New().String()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			Expect(create(ctx, rackToken, other.ResourceID())).Error().
				To(MatchError(access.ErrDenied))
		})
		It("Should let the Owner create a key for another user", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, bob.ResourceID()))
			Expect(res.Key.Subject).To(Equal(bob.ResourceID()))
		})
		DescribeTable("Should reject a subject that is malformed or missing",
			func(ctx SpecContext, subject resource.ID, message string) {
				Expect(create(ctx, rootToken, subject)).Error().
					To(MatchError(ContainSubstring(message)))
			},
			Entry(
				"malformed user",
				resource.ID{Type: resource.TypeUser, Key: "missing"},
				`subject: "missing" is not a key of user`,
			),
			Entry(
				"malformed rack",
				resource.ID{Type: resource.TypeRack, Key: "missing"},
				`subject: "missing" is not a key of rack`,
			),
			Entry(
				"missing user",
				resource.ID{
					Type: resource.TypeUser,
					Key:  "00000000-0000-0000-0000-000000000001",
				},
				"subject: user 00000000-0000-0000-0000-000000000001 does not exist",
			),
			Entry(
				"missing rack",
				resource.ID{Type: resource.TypeRack, Key: "4000000000"},
				"subject: rack 4000000000 does not exist",
			),
		)
		It("Should reject a channel subject before it checks access", func(
			ctx SpecContext,
		) {
			channel := resource.ID{Type: resource.TypeChannel, Key: "1"}
			Expect(create(ctx, bobToken, channel)).Error().To(MatchError(
				ContainSubstring("subject: APIKey cannot reference channel 1"),
			))
		})
	})
	Describe("RetrieveKeys", func() {
		It("Should list the caller's own keys by default", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, bobToken, resource.ID{})).Key
			MustSucceed(create(ctx, rootToken, resource.ID{}))
			res := MustSucceed(retrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{},
			))
			Expect(res.Keys).To(ConsistOf(own))
		})
		It("Should deny a user listing another subject's keys", func(ctx SpecContext) {
			Expect(retrieveKeys(
				authed(ctx, bobToken),
				apiauth.KeyRetrieveRequest{Subjects: []resource.ID{root.ResourceID()}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
	Describe("DeleteKeys", func() {
		del := func(ctx context.Context, token string, keys ...apikey.Key) error {
			_, err := deleteKeys(
				authed(ctx, token),
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
		It("Should deny a rack deleting another rack's key", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, rootToken, rackID))
			rackToken := login(ctx, keyProof(own.Token))
			other := rack.Rack{Name: "rack-" + uuid.New().String()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return rackSvc.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			res := MustSucceed(create(ctx, rootToken, other.ResourceID()))
			Expect(del(ctx, rackToken, res.Key.Key)).To(MatchError(access.ErrDenied))
		})
		It("Should let a caller that may delete on a subject delete its key", func(
			ctx SpecContext,
		) {
			res := MustSucceed(create(ctx, rootToken, eng.ResourceID()))
			grant(ctx, bob.Key, access.ActionDelete, eng.ResourceID())
			Expect(del(ctx, bobToken, res.Key.Key)).To(Succeed())
			Expect(apiSvc.CreateSession(
				requestCtx(ctx),
				apiauth.SessionCreateRequest{Proof: keyProof(res.Token)},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, bobToken, resource.ID{}))
			Expect(del(ctx, bobToken, uuid.New(), res.Key.Key)).To(Succeed())
			Expect(apiSvc.CreateSession(
				requestCtx(ctx),
				apiauth.SessionCreateRequest{Proof: keyProof(res.Token)},
			)).Error().To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should not end the sessions of a password whose key is sent", func(
			ctx SpecContext,
		) {
			var p password.Password
			Expect(passwordSvc.NewRetrieve().
				Where(password.MatchSubjects(bob.ResourceID())).
				Entry(&p).
				Exec(ctx, db)).To(Succeed())
			Expect(del(ctx, bobToken, p.Key)).To(Succeed())
			MustSucceed(authSvc.Authenticate(ctx, bobToken))
		})
	})
})
