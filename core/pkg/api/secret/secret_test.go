// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json/v2"
	"encoding/pem"
	"math/big"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	secretType      = resource.ID{Type: resource.TypeSecret}
	deviceType      = resource.ID{Type: resource.TypeDevice}
	certificateType = resource.ID{Type: resource.TypeCertificate}
	all             = access.AllActions
)

func whereKeys(keys ...secret.Key) apisecret.RetrieveRequest {
	return apisecret.RetrieveRequest{
		Where: &secret.Where{Key: &filter.Set[secret.Key]{In: keys}},
	}
}

func whereKeysWithUses(keys ...secret.Key) apisecret.RetrieveRequest {
	req := whereKeys(keys...)
	req.IncludeUses = true
	return req
}

func createSecret(ctx SpecContext, name, value string) secret.Secret {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: name}}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Secret.NewWriter(tx).Set(ctx, &s, []byte(value))
	})).To(Succeed())
	return s
}

func retrieveSecret(ctx SpecContext, key secret.Key) secret.Secret {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.Secret.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, db)).To(Succeed())
	return s
}

// createCertificate creates a stored certificate with a private key, whose secret the
// certificate uses.
func createCertificate(ctx SpecContext) certificate.Certificate {
	GinkgoHelper()
	key := MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "plant"},
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
	}
	der := MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
	)
	keyDER := MustSucceed(x509.MarshalPKCS8PrivateKey(key))
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: "Plant"},
		Pem:  string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
	}}
	privateKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Certificate.NewWriter(tx).Set(ctx, &c, privateKey)
	})).To(Succeed())
	return c
}

var _ = Describe("ServiceConfig", func() {
	full := func() apisecret.ServiceConfig {
		return apisecret.ServiceConfig{
			Secret: &secret.Service{},
			RBAC:   &rbac.Service{},
		}
	}
	DescribeTable("Should fail when a service is missing",
		func(clear func(*apisecret.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("secret", func(c *apisecret.ServiceConfig) { c.Secret = nil },
			"secret: must be non-nil"),
		Entry("rbac", func(c *apisecret.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apisecret.ServiceConfig{}.Override(cfg)
		Expect(merged.Secret).To(BeIdenticalTo(cfg.Secret))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("Service", func() {
	Describe("Set", func() {
		It("Should create stored and file secrets and never return a value",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				res := MustSucceed(set(AuthedCtx(ctx, u),
					apisecret.SetRequest{Secrets: []apisecret.Input{
						{
							Source: secret.SourceStored,
							Name:   "Password",
							Value:  "hunter2",
						},
						{Source: secret.SourceFile, Name: "Key", FileName: "k.pem"},
					}},
				))
				Expect(res.Secrets).To(HaveLen(2))
				Expect(res.Secrets[0].Base().Key).ToNot(Equal(uuid.Nil()))
				Expect(retrieveSecret(ctx, res.Secrets[1].Base().Key).Variant).
					To(BeAssignableToTypeOf(secret.FileSecret{}))
				encoded := string(MustSucceed(json.Marshal(res)))
				Expect(encoded).ToNot(ContainSubstring("value"))
				Expect(encoded).ToNot(ContainSubstring("hunter2"))
			},
		)
		It(
			"Should replace the secret whose key the request gives",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				s := createSecret(ctx, "Before", "a")
				MustSucceed(set(AuthedCtx(ctx, u),
					apisecret.SetRequest{Secrets: []apisecret.Input{{
						Key:    s.Base().Key,
						Source: secret.SourceStored,
						Name:   "After",
						Value:  "b",
					}}},
				))
				Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("After"))
			},
		)
		DescribeTable("Should reject an invalid input",
			func(ctx SpecContext, in apisecret.Input, msg string) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				Expect(set(AuthedCtx(ctx, u),
					apisecret.SetRequest{Secrets: []apisecret.Input{in}},
				)).Error().To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"an unknown source",
				apisecret.Input{Source: "cloud", Name: "A", Value: "v"},
				`source: must be stored or file, not "cloud"`,
			),
			Entry(
				"a stored secret with a file name",
				apisecret.Input{
					Source:   secret.SourceStored,
					Name:     "A",
					Value:    "v",
					FileName: "a",
				},
				"file_name: must be empty for a stored secret",
			),
			Entry(
				"a file secret with a value",
				apisecret.Input{
					Source:   secret.SourceFile,
					Name:     "A",
					Value:    "v",
					FileName: "a",
				},
				"value: must be empty for a file secret",
			),
		)
		It("Should deny a user without create access", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(set(AuthedCtx(ctx, u),
				apisecret.SetRequest{Secrets: []apisecret.Input{
					{Source: secret.SourceStored, Name: "A", Value: "v"},
				}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		Describe("Replace", func() {
			var (
				u        user.User
				existing secret.Secret
			)
			BeforeEach(func(ctx SpecContext) {
				u = createUser(ctx)
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionCreate},
					secretType,
				)
				existing = createSecret(ctx, "Before", "a")
			})
			replace := func(
				ctx SpecContext,
				subject user.User,
			) (apisecret.SetResponse, error) {
				return set(
					AuthedCtx(ctx, subject),
					apisecret.SetRequest{
						Secrets: []apisecret.Input{{
							Key:    existing.Base().Key,
							Source: secret.SourceStored,
							Name:   "After",
							Value:  "b",
						}},
					},
				)
			}
			It(
				"Should let a user without update access create new secrets",
				func(ctx SpecContext) {
					key := uuid.New()
					res := MustSucceed(set(AuthedCtx(ctx, u),
						apisecret.SetRequest{Secrets: []apisecret.Input{
							{Source: secret.SourceStored, Name: "New", Value: "v"},
							{
								Key:      key,
								Source:   secret.SourceFile,
								Name:     "Keyed",
								FileName: "k",
							},
						}},
					))
					Expect(res.Secrets[1].Base().Key).To(Equal(key))
					Expect(retrieveSecret(ctx, key).Base().Name).To(Equal("Keyed"))
				},
			)
			It(
				"Should deny a user without update access a replace",
				func(ctx SpecContext) {
					Expect(replace(ctx, u)).Error().To(SatisfyAll(
						MatchError(access.ErrDenied),
						MatchError(ContainSubstring(
							u.ResourceID().String()+" cannot update "+
								existing.ResourceID().String(),
						)),
					))
					Expect(retrieveSecret(ctx, existing.Base().Key).Base().Name).
						To(Equal("Before"))
				},
			)
			It(
				"Should let a user with only update access replace a secret",
				func(ctx SpecContext) {
					editor := createUser(ctx)
					allow(
						ctx,
						editor.ResourceID(),
						[]access.Action{access.ActionUpdate},
						existing.ResourceID(),
					)
					MustSucceed(replace(ctx, editor))
					Expect(retrieveSecret(ctx, existing.Base().Key).Base().Name).
						To(Equal("After"))
				},
			)
			It(
				"Should deny a replace that comes with a new key in the same request",
				func(ctx SpecContext) {
					key := uuid.New()
					Expect(set(AuthedCtx(ctx, u),
						apisecret.SetRequest{Secrets: []apisecret.Input{
							{
								Key:    key,
								Source: secret.SourceStored,
								Name:   "New",
								Value:  "v",
							},
							{
								Key:    existing.Base().Key,
								Source: secret.SourceStored,
								Name:   "After",
								Value:  "b",
							},
						}},
					)).Error().To(SatisfyAll(
						MatchError(access.ErrDenied),
						MatchError(ContainSubstring(
							u.ResourceID().String()+" cannot update "+
								existing.ResourceID().String(),
						)),
					))
					Expect(retrieveSecret(ctx, existing.Base().Key).Base().Name).
						To(Equal("Before"))
					Expect(svc.Secret.NewRetrieve().
						Where(secret.MatchKeys(key)).
						Exec(ctx, db)).To(MatchError(query.ErrNotFound))
				},
			)
			It("Should let an Owner replace a secret", func(ctx SpecContext) {
				owner := createUser(ctx)
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.User.NewWriter(tx).
						SetRole(ctx, svc.RBAC.Builtin.OwnerKey, owner.Key)
				})).To(Succeed())
				MustSucceed(replace(ctx, owner))
				Expect(retrieveSecret(ctx, existing.Base().Key).Base().Name).
					To(Equal("After"))
			})
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve secrets by key", func(ctx SpecContext) {
			u := createUser(ctx)
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
			)
			s := createSecret(ctx, "Password", "hunter2")
			res := MustSucceed(
				retrieve(AuthedCtx(ctx, u), whereKeys(s.Base().Key)),
			)
			Expect(res.Secrets).To(HaveLen(1))
			Expect(res.Secrets[0].Base().Name).To(Equal("Password"))
		})
		It("Should leave out a key that no secret has", func(ctx SpecContext) {
			u := createUser(ctx)
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
			)
			s := createSecret(ctx, "Password", "hunter2")
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, u),
				whereKeys(s.Base().Key, uuid.New()),
			))
			Expect(res.Secrets).To(HaveLen(1))
			Expect(res.Secrets[0].Base().Key).To(Equal(s.Base().Key))
		})
		Describe("IncludeUses", func() {
			var (
				u    user.User
				cert certificate.Certificate
				key  secret.Key
			)
			BeforeEach(func(ctx SpecContext) {
				u = createUser(ctx)
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					secretType,
				)
				cert = createCertificate(ctx)
				key = *cert.Variant.(certificate.StoredCertificate).PrivateKey
			})
			It("Should return the resources that use each secret",
				func(ctx SpecContext) {
					allow(
						ctx,
						u.ResourceID(),
						[]access.Action{access.ActionRetrieve},
						certificateType,
					)
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeysWithUses(key)),
					)
					Expect(res.Secrets).To(HaveLen(1))
					Expect(res.Secrets[0].Base().Uses).
						To(Equal([]resource.ID{cert.ResourceID()}))
				},
			)
			It("Should leave out the uses that the caller cannot read",
				func(ctx SpecContext) {
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeysWithUses(key)),
					)
					Expect(res.Secrets).To(HaveLen(1))
					Expect(res.Secrets[0].Base().Uses).To(Equal([]resource.ID{}))
				},
			)
			It("Should return empty uses for a secret that nothing uses",
				func(ctx SpecContext) {
					s := createSecret(ctx, "Password", "hunter2")
					res := MustSucceed(retrieve(
						AuthedCtx(ctx, u),
						whereKeysWithUses(s.Base().Key),
					))
					Expect(res.Secrets).To(HaveLen(1))
					Expect(res.Secrets[0].Base().Uses).To(Equal([]resource.ID{}))
				},
			)
			It("Should leave uses nil when the request does not ask for them",
				func(ctx SpecContext) {
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeys(key)),
					)
					Expect(res.Secrets).To(HaveLen(1))
					Expect(res.Secrets[0].Base().Uses).To(BeNil())
				},
			)
		})
		It("Should deny a user without retrieve access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "hunter2")
			Expect(retrieve(AuthedCtx(ctx, u), whereKeys(s.Base().Key))).
				Error().To(MatchError(access.ErrDenied))
		})
		It(
			"Should leave out the secrets a user cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				a := createSecret(ctx, "A", "a")
				b := createSecret(ctx, "B", "b")
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					a.ResourceID(),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, u), apisecret.RetrieveRequest{}),
				)
				keys := make([]secret.Key, len(res.Secrets))
				for i, s := range res.Secrets {
					keys[i] = s.Base().Key
				}
				Expect(keys).To(ContainElement(a.Base().Key))
				Expect(keys).NotTo(ContainElement(b.Base().Key))
			},
		)
	})

	Describe("Dispatch", func() {
		setName := func(key secret.Key) apisecret.DispatchRequest {
			return apisecret.DispatchRequest{
				Key: key,
				Actions: []secret.Action{secret.NewSetAction(
					secret.SetPayload{Path: []string{"name"}, Value: "After"},
				)},
			}
		}
		It("Should apply the actions when update is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionUpdate},
				s.ResourceID(),
			)
			MustSucceed(dispatch(AuthedCtx(ctx, u), setName(s.Base().Key)))
			Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("After"))
		})
		It("Should deny a user without update access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			Expect(dispatch(AuthedCtx(ctx, u), setName(s.Base().Key))).
				Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					u.ResourceID().String()+" cannot update "+s.ResourceID().String(),
				)),
			))
			Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("Before"))
		})
	})

	Describe("Delete", func() {
		It("Should delete the secret when delete is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionDelete},
				s.ResourceID(),
			)
			MustSucceed(del(AuthedCtx(ctx, u),
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			))
			Expect(svc.Secret.NewRetrieve().
				Where(secret.MatchKeys(s.Base().Key)).
				Exec(ctx, db)).To(MatchError(query.ErrNotFound))
		})
		It("Should deny a user without delete access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			Expect(del(AuthedCtx(ctx, u),
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Grants and Resolve", func() {
		rk := rack.Key(7)
		allowGrants := func(ctx SpecContext, u user.User) {
			GinkgoHelper()
			allow(ctx, u.ResourceID(), []access.Action{access.ActionCreate}, deviceType)
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
			)
		}
		grant := func(
			ctx freighter.Context,
			keys ...secret.Key,
		) apisecret.GrantCreateResponse {
			GinkgoHelper()
			return MustSucceed(
				createGrant(ctx, apisecret.GrantCreateRequest{
					Subject: rk.ResourceID(),
					Secrets: keys,
				}),
			)
		}

		It("Should let the rack resolve the granted secrets in request order",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allowGrants(ctx, u)
				a := createSecret(ctx, "A", "first")
				b := createSecret(ctx, "B", "second")
				keys := []secret.Key{b.Base().Key, a.Base().Key}
				uCtx := SubjectCtx(ctx, u.ResourceID())
				res := grant(uCtx, keys...)
				values := MustSucceed(resolve(SubjectCtx(ctx, rk.ResourceID()),
					apisecret.ResolveRequest{Keys: keys, Grant: res.Grant},
				))
				Expect(values.Values).To(Equal([]apisecret.Value{
					{Key: b.Base().Key, Value: "second"},
					{Key: a.Base().Key, Value: "first"},
				}))
			},
		)
		It("Should let the rack resolve a grant only once", func(ctx SpecContext) {
			u := createUser(ctx)
			allowGrants(ctx, u)
			s := createSecret(ctx, "A", "v")
			uCtx := SubjectCtx(ctx, u.ResourceID())
			res := grant(uCtx, s.Base().Key)
			req := apisecret.ResolveRequest{
				Keys:  []secret.Key{s.Base().Key},
				Grant: res.Grant,
			}
			rCtx := SubjectCtx(ctx, rk.ResourceID())
			MustSucceed(resolve(rCtx, req))
			Expect(resolve(rCtx, req)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"grant "+res.Grant.String()+" has ended or does not exist",
				)),
			))
		})
		It("Should deny a grant to a user who cannot create devices",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				uCtx := SubjectCtx(ctx, u.ResourceID())
				Expect(createGrant(uCtx, apisecret.GrantCreateRequest{
					Subject: rk.ResourceID(),
					Secrets: []secret.Key{s.Base().Key},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should deny a grant of a secret the user cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionCreate},
					deviceType,
				)
				s := createSecret(ctx, "A", "v")
				uCtx := SubjectCtx(ctx, u.ResourceID())
				Expect(createGrant(uCtx, apisecret.GrantCreateRequest{
					Subject: rk.ResourceID(),
					Secrets: []secret.Key{s.Base().Key},
				})).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should deny a grant to a rack", func(ctx SpecContext) {
			s := createSecret(ctx, "A", "v")
			Expect(createGrant(SubjectCtx(ctx, rk.ResourceID()),
				apisecret.GrantCreateRequest{
					Subject: rk.ResourceID(),
					Secrets: []secret.Key{s.Base().Key},
				},
			)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("only a user can grant secrets")),
			))
		})
		It(
			"Should deny resolve to a subject that is not a rack",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				Expect(resolve(AuthedCtx(ctx, u),
					apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring("only a rack can resolve secrets")),
				))
			},
		)
		It("Should deny resolve without a grant", func(ctx SpecContext) {
			s := createSecret(ctx, "A", "v")
			Expect(resolve(SubjectCtx(ctx, rk.ResourceID()),
				apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("has ended or does not exist")),
			))
		})
	})
})
