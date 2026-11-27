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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apiapikey "github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

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

var _ = Describe("ServiceConfig", func() {
	full := func() apiapikey.ServiceConfig {
		return apiapikey.ServiceConfig{Auth: &auth.Service{}, APIKey: &apikey.Service{}}
	}

	Describe("Validate", func() {
		It("Should succeed when every service is set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a service is missing",
			func(unset func(*apiapikey.ServiceConfig), msg string) {
				cfg := full()
				unset(&cfg)
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
		)
	})

	Describe("Override", func() {
		It("Should take the other config's services when set", func() {
			cfg := full()
			merged := apiapikey.ServiceConfig{}.Override(cfg)
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.APIKey).To(BeIdenticalTo(cfg.APIKey))
		})

		It("Should keep the receiver's services when the other's are nil", func() {
			cfg := full()
			merged := cfg.Override(apiapikey.ServiceConfig{})
			Expect(merged.Auth).To(BeIdenticalTo(cfg.Auth))
			Expect(merged.APIKey).To(BeIdenticalTo(cfg.APIKey))
		})
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
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		disabled := newService(MustOpen(auth.OpenService(ctx, auth.ServiceConfig{
			DB: other,
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

var _ = Describe("Maximum lifetime", func() {
	const maxLifetime = 30 * telem.Day
	var capped *apiapikey.Service
	BeforeEach(func(ctx SpecContext) {
		other := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(other.Run)
		capped = MustSucceed(apiapikey.NewService(apiapikey.ServiceConfig{
			Auth: authSvc,
			APIKey: MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{
				DB:          other,
				MaxLifetime: maxLifetime,
			})),
		}))
	})
	DescribeTable(
		"Should refuse a key that outlives the maximum lifetime before it writes",
		func(ctx SpecContext, expires func() telem.TimeStamp) {
			Expect(capped.Create(
				requestCtx(ctx),
				nil,
				apiapikey.CreateRequest{Name: "capped", Expires: expires()},
			)).Error().To(MatchError(ContainSubstring("expires: must be at most")))
		},
		Entry("no expiry", func() telem.TimeStamp { return 0 }),
		Entry("the maximum timestamp", func() telem.TimeStamp {
			return telem.TimeStampMax
		}),
		Entry("past the lifetime", func() telem.TimeStamp {
			return telem.Now().Add(maxLifetime + telem.Hour)
		}),
	)
})

var _ = Describe("API keys", func() {
	var (
		bob, eng           user.User
		bobToken, engToken string
		rootToken          string
		rackID             resource.ID
	)
	create := func(
		ctx context.Context,
		token string,
		subject resource.ID,
	) (apiapikey.CreateResponse, error) {
		return createKey(
			authed(ctx, token),
			apiapikey.CreateRequest{Subject: subject, Name: "ci"},
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
	Describe("Create", func() {
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
		It("Should return the key's expiry", func(ctx SpecContext) {
			expires := telem.Now().Add(telem.TimeSpan(time.Hour))
			res := MustSucceed(createKey(
				authed(ctx, bobToken),
				apiapikey.CreateRequest{Name: "ci", Expires: expires},
			))
			Expect(res.Key.Expires).To(Equal(expires))
			retrieved := MustSucceed(retrieveKeys(
				authed(ctx, bobToken),
				apiapikey.RetrieveRequest{},
			))
			Expect(retrieved.Keys).To(ContainElement(res.Key))
		})
		It("Should give a key with no expiry the maximum expiry", func(
			ctx SpecContext,
		) {
			res := MustSucceed(createKey(
				authed(ctx, bobToken),
				apiapikey.CreateRequest{Name: "ci"},
			))
			Expect(res.Key.Expires).To(Equal(telem.TimeStampMax))
		})
		It("Should reject an expiry in the past", func(ctx SpecContext) {
			Expect(createKey(
				authed(ctx, bobToken),
				apiapikey.CreateRequest{Name: "ci", Expires: telem.TimeStamp(1)},
			)).Error().
				To(MatchError(ContainSubstring("expires: must be in the future")))
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
	Describe("Retrieve", func() {
		It("Should list the caller's own keys by default", func(ctx SpecContext) {
			own := MustSucceed(create(ctx, bobToken, resource.ID{})).Key
			MustSucceed(create(ctx, rootToken, resource.ID{}))
			res := MustSucceed(retrieveKeys(
				authed(ctx, bobToken),
				apiapikey.RetrieveRequest{},
			))
			Expect(res.Keys).To(ConsistOf(own))
		})
		It("Should deny a user listing another subject's keys", func(ctx SpecContext) {
			Expect(retrieveKeys(
				authed(ctx, bobToken),
				apiapikey.RetrieveRequest{
					Subjects: []resource.ID{root.ResourceID()},
				},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
	Describe("Delete", func() {
		del := func(ctx context.Context, token string, keys ...apikey.Key) error {
			_, err := deleteKeys(
				authed(ctx, token),
				apiapikey.DeleteRequest{Keys: keys},
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
			Expect(authSvc.Login(ctx, keyProof(res.Token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should deny a user deleting another subject's key", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, rootToken, resource.ID{}))
			Expect(del(ctx, bobToken, res.Key.Key)).To(MatchError(access.ErrDenied))
			login(ctx, keyProof(res.Token))
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
			Expect(authSvc.Login(ctx, keyProof(res.Token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
		})
		It("Should ignore unknown keys", func(ctx SpecContext) {
			res := MustSucceed(create(ctx, bobToken, resource.ID{}))
			Expect(del(ctx, bobToken, uuid.New(), res.Key.Key)).To(Succeed())
			Expect(authSvc.Login(ctx, keyProof(res.Token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
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
	Describe("Rotate", func() {
		rotate := func(ctx context.Context, token string) (apiapikey.CreateResponse, error) {
			return rotateKey(authed(ctx, token), struct{}{})
		}
		It("Should replace the key and keep the sessions of the old key", func(
			ctx SpecContext,
		) {
			old := MustSucceed(create(ctx, rootToken, rackID))
			rackToken := login(ctx, keyProof(old.Token))
			other := login(ctx, keyProof(old.Token))
			res := MustSucceed(rotate(ctx, rackToken))
			Expect(res.Key.Key).ToNot(Equal(old.Key.Key))
			Expect(res.Key.Subject).To(Equal(rackID))
			Expect(res.Key.Name).To(Equal(old.Key.Name))
			for _, token := range []string{rackToken, other} {
				session := MustSucceed(authSvc.Authenticate(ctx, token))
				Expect(session.Credential).To(Equal(res.Key.Key))
			}
			Expect(authSvc.Login(ctx, keyProof(old.Token))).Error().
				To(MatchError(auth.ErrInvalidCredentials))
			session := MustSucceed(authSvc.Authenticate(
				ctx,
				login(ctx, keyProof(res.Token)),
			))
			Expect(session.Subject).To(Equal(rackID))
		})
		It("Should keep the lifetime of the old key", func(ctx SpecContext) {
			lifetime := telem.TimeSpan(30 * 24 * time.Hour)
			old := MustSucceed(createKey(
				authed(ctx, bobToken),
				apiapikey.CreateRequest{
					Name:    "ci",
					Expires: telem.Now().Add(lifetime),
				},
			))
			res := MustSucceed(rotate(ctx, login(ctx, keyProof(old.Token))))
			Expect(res.Key.Created.Span(res.Key.Expires)).
				To(BeNumerically("~", lifetime, telem.Second))
		})
		It("Should replace a key that never expires with one that never expires", func(
			ctx SpecContext,
		) {
			old := MustSucceed(create(ctx, bobToken, resource.ID{}))
			res := MustSucceed(rotate(ctx, login(ctx, keyProof(old.Token))))
			Expect(res.Key.Expires).To(Equal(telem.TimeStampMax))
		})
		It("Should refuse a session that logged in with a password", func(
			ctx SpecContext,
		) {
			Expect(rotate(ctx, bobToken)).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
