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
	"encoding/json/v2"
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/sift"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	secretType      = resource.ID{Type: resource.TypeSecret}
	deviceType      = resource.ID{Type: resource.TypeDevice}
	certificateType = resource.ID{Type: resource.TypeCertificate}
	rackType        = resource.ID{Type: resource.TypeRack}
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
	certPEM, keyPEM := certtestutil.NewPair()
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: "Plant"},
		Pem:  certPEM,
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Certificate.NewWriter(tx).Set(ctx, &c, []byte(keyPEM))
	})).To(Succeed())
	return c
}

var _ = Describe("ServiceConfig", func() {
	full := func() apisecret.ServiceConfig {
		return apisecret.ServiceConfig{
			Secret:      &secret.Service{},
			RBAC:        &rbac.Service{},
			Task:        &task.Service{},
			Rack:        &rack.Service{},
			Certificate: &certificate.Service{},
			DB:          &gorp.DB{},
		}
	}
	DescribeTable("Should fail when a service is missing",
		func(unset func(*apisecret.ServiceConfig), msg string) {
			cfg := full()
			unset(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("secret", func(c *apisecret.ServiceConfig) { c.Secret = nil },
			"secret: must be non-nil"),
		Entry("rbac", func(c *apisecret.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil"),
		Entry("task", func(c *apisecret.ServiceConfig) { c.Task = nil },
			"task: must be non-nil"),
		Entry("rack", func(c *apisecret.ServiceConfig) { c.Rack = nil },
			"rack: must be non-nil"),
		Entry("certificate", func(c *apisecret.ServiceConfig) { c.Certificate = nil },
			"certificate: must be non-nil"),
		Entry("db", func(c *apisecret.ServiceConfig) { c.DB = nil },
			"db: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apisecret.ServiceConfig{}.Override(cfg)
		Expect(merged.Secret).To(BeIdenticalTo(cfg.Secret))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
		Expect(merged.Task).To(BeIdenticalTo(cfg.Task))
		Expect(merged.Rack).To(BeIdenticalTo(cfg.Rack))
		Expect(merged.Certificate).To(BeIdenticalTo(cfg.Certificate))
		Expect(merged.DB).To(BeIdenticalTo(cfg.DB))
	})
})

var _ = Describe("Service", func() {
	Describe("Set", func() {
		It("Should create stored and file secrets and never return a value",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					rackType,
				)
				res := MustSucceed(set(AuthedCtx(ctx, u),
					apisecret.SetRequest{Secrets: []apisecret.Input{
						{
							Source: secret.SourceStored,
							Name:   "Password",
							Value:  "hunter2",
						},
						{
							Source:   secret.SourceFile,
							Name:     "Key",
							FileName: "k.pem",
							Rack:     svc.Rack.EmbeddedKey,
						},
					}},
				))
				Expect(res.Secrets).To(HaveLen(2))
				Expect(res.Secrets[0].Base().Key).ToNot(Equal(uuid.Nil()))
				file, ok := retrieveSecret(ctx, res.Secrets[1].Base().Key).
					Variant.(secret.FileSecret)
				Expect(ok).To(BeTrue())
				Expect(file.Rack).To(Equal(svc.Rack.EmbeddedKey))
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
				"a stored secret with a rack",
				apisecret.Input{
					Source: secret.SourceStored,
					Name:   "A",
					Value:  "v",
					Rack:   1,
				},
				"rack: must be empty for a stored secret",
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
		It("Should deny a file secret on a rack the user cannot retrieve", func(
			ctx SpecContext,
		) {
			u := createUser(ctx)
			allow(ctx, u.ResourceID(), all, secretType)
			Expect(set(AuthedCtx(ctx, u),
				apisecret.SetRequest{Secrets: []apisecret.Input{{
					Source:   secret.SourceFile,
					Name:     "Key",
					FileName: "k.pem",
					Rack:     svc.Rack.EmbeddedKey,
				}}},
			)).Error().To(MatchError(
				u.ResourceID().String() + " cannot retrieve " +
					svc.Rack.EmbeddedKey.ResourceID().String() +
					": access denied: auth error",
			))
		})
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
					allow(
						ctx,
						u.ResourceID(),
						[]access.Action{access.ActionRetrieve},
						rackType,
					)
					key := uuid.New()
					res := MustSucceed(set(AuthedCtx(ctx, u),
						apisecret.SetRequest{Secrets: []apisecret.Input{
							{Source: secret.SourceStored, Name: "New", Value: "v"},
							{
								Key:      key,
								Source:   secret.SourceFile,
								Name:     "Keyed",
								FileName: "k",
								Rack:     svc.Rack.EmbeddedKey,
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
				stored, ok := cert.Variant.(certificate.StoredCertificate)
				Expect(ok).To(BeTrue())
				key = *stored.PrivateKey
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
			It("Should return the task of a config that uses a secret",
				func(ctx SpecContext) {
					s := createSecret(ctx, "Routing key", "rk")
					t := task.Task{
						Type: pagerduty.AlertTaskType,
						Rack: svc.Rack.EmbeddedKey,
						Name: "Alerts",
						Config: msgpack.EncodedJSON{
							"routing_key": s.Base().Key.String(),
						},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.Task.NewWriter(tx).Create(ctx, &t)
					})).To(Succeed())
					allow(
						ctx,
						u.ResourceID(),
						[]access.Action{access.ActionRetrieve},
						t.ResourceID(),
					)
					res := MustSucceed(retrieve(
						AuthedCtx(ctx, u),
						whereKeysWithUses(s.Base().Key),
					))
					Expect(res.Secrets).To(HaveLen(1))
					Expect(res.Secrets[0].Base().Uses).
						To(Equal([]resource.ID{t.ResourceID()}))
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
		var (
			rk  rack.Key
			t   task.Task
			cmd task.Command
		)
		createTask := func(ctx SpecContext, rk rack.Key) task.Task {
			GinkgoHelper()
			t := task.Task{
				Type:   sift.ScanTaskType,
				Rack:   rk,
				Name:   "Scanner",
				Config: msgpack.EncodedJSON{"rate": 1},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Task.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
			return t
		}
		BeforeEach(func(ctx SpecContext) {
			rk = svc.Rack.EmbeddedKey
			t = createTask(ctx, rk)
			cmd = task.Command{
				Task: t.Key,
				Type: "test_connection",
				Key:  "cmd",
				Args: msgpack.EncodedJSON{"host": "plc"},
			}
		})
		allowGrants := func(ctx SpecContext, u user.User) {
			GinkgoHelper()
			allow(ctx, u.ResourceID(), []access.Action{access.ActionCreate}, deviceType)
			allow(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
				certificateType,
			)
		}
		createFileCertificate := func(ctx SpecContext, r rack.Key) certificate.Key {
			GinkgoHelper()
			c := certificate.Certificate{Variant: certificate.FileCertificate{
				Base:            certificate.Base{Name: "Client"},
				CertificatePath: "/pki/client.pem",
				PrivateKeyFile:  new("client.key"),
				Rack:            r,
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Certificate.NewWriter(tx).Set(ctx, &c, nil)
			})).To(Succeed())
			return c.Base().Key
		}
		grantCertificates := func(
			ctx SpecContext,
			keys ...certificate.Key,
		) (grant.Key, error) {
			GinkgoHelper()
			u := createUser(ctx)
			allowGrants(ctx, u)
			res, err := createGrant(
				SubjectCtx(ctx, u.ResourceID()),
				apisecret.GrantCreateRequest{
					Subject:      rk.ResourceID(),
					Certificates: keys,
					Command:      cmd,
				},
			)
			return res.Grant, err
		}
		grantReq := func(keys ...secret.Key) apisecret.GrantCreateRequest {
			return apisecret.GrantCreateRequest{
				Subject: rk.ResourceID(),
				Secrets: keys,
				Command: cmd,
			}
		}
		grantSecrets := func(ctx SpecContext, keys ...secret.Key) grant.Key {
			GinkgoHelper()
			u := createUser(ctx)
			allowGrants(ctx, u)
			uCtx := SubjectCtx(ctx, u.ResourceID())
			return MustSucceed(createGrant(uCtx, grantReq(keys...))).Grant
		}

		It("Should return the stored command and the values of the granted secrets",
			func(ctx SpecContext) {
				a := createSecret(ctx, "A", "first")
				b := createSecret(ctx, "B", "second")
				g := grantSecrets(ctx, a.Base().Key, b.Base().Key)
				res := MustSucceed(resolve(SubjectCtx(ctx, rk.ResourceID()),
					apisecret.ResolveRequest{Grant: g, Task: t.Key},
				))
				Expect(res.Command).To(Equal(cmd))
				Expect(res.Secrets).To(Equal([]secret.Key{a.Base().Key, b.Base().Key}))
				Expect(res.Values).To(ConsistOf(
					apisecret.Value{Key: a.Base().Key, Value: "first"},
					apisecret.Value{Key: b.Base().Key, Value: "second"},
				))
			},
		)
		It("Should return the file certificates that a grant names, with no values",
			func(ctx SpecContext) {
				c := createFileCertificate(ctx, rk)
				g := MustSucceed(grantCertificates(ctx, c))
				res := MustSucceed(resolve(SubjectCtx(ctx, rk.ResourceID()),
					apisecret.ResolveRequest{Grant: g, Task: t.Key},
				))
				Expect(res.Certificates).To(Equal([]certificate.Key{c}))
				Expect(res.Secrets).To(BeEmpty())
				Expect(res.Values).To(BeEmpty())
			},
		)
		It("Should refuse a grant of a file certificate of another rack",
			func(ctx SpecContext) {
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				c := createFileCertificate(ctx, other.Key)
				Expect(grantCertificates(ctx, c)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(fmt.Sprintf(
						"certificate \"Client\" is a file of rack %d, not of %s",
						other.Key,
						rk.ResourceID(),
					))),
				))
			},
		)
		It("Should refuse a grant that names a stored certificate", func(
			ctx SpecContext,
		) {
			c := createCertificate(ctx)
			Expect(grantCertificates(ctx, c.Base().Key)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"certificate \"Plant\" is stored: name its private key in secrets",
				)),
			))
		})
		It("Should return not found for a grant of a missing certificate", func(
			ctx SpecContext,
		) {
			Expect(grantCertificates(ctx, uuid.New())).Error().
				To(MatchError(query.ErrNotFound))
		})
		It("Should deny a grant of a certificate the user cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionCreate},
					deviceType,
				)
				c := createFileCertificate(ctx, rk)
				Expect(createGrant(
					SubjectCtx(ctx, u.ResourceID()),
					apisecret.GrantCreateRequest{
						Subject:      rk.ResourceID(),
						Certificates: []certificate.Key{c},
						Command:      cmd,
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should let the rack resolve a grant only once", func(ctx SpecContext) {
			g := grantSecrets(ctx, createSecret(ctx, "A", "v").Base().Key)
			req := apisecret.ResolveRequest{Grant: g, Task: t.Key}
			rCtx := SubjectCtx(ctx, rk.ResourceID())
			MustSucceed(resolve(rCtx, req))
			Expect(resolve(rCtx, req)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					"grant "+g.String()+" has ended or does not exist",
				)),
			))
		})
		It("Should deny a resolve for another task and spend the grant",
			func(ctx SpecContext) {
				g := grantSecrets(ctx, createSecret(ctx, "A", "v").Base().Key)
				rCtx := SubjectCtx(ctx, rk.ResourceID())
				other := createTask(ctx, rk)
				Expect(resolve(rCtx, apisecret.ResolveRequest{
					Grant: g,
					Task:  other.Key,
				})).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"grant is not for task "+other.Key.String(),
					)),
				))
				Expect(resolve(rCtx, apisecret.ResolveRequest{
					Grant: g,
					Task:  t.Key,
				})).Error().To(MatchError(ContainSubstring(
					"grant " + g.String() + " has ended or does not exist",
				)))
			},
		)
		It("Should refuse a grant whose task is on another rack",
			func(ctx SpecContext) {
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				cmd.Task = createTask(ctx, other.Key).Key
				u := createUser(ctx)
				allowGrants(ctx, u)
				s := createSecret(ctx, "A", "v")
				Expect(createGrant(
					SubjectCtx(ctx, u.ResourceID()),
					grantReq(s.Base().Key),
				)).Error().To(SatisfyAll(
					BeAssignableToTypeOf(validate.PathError{}),
					MatchError(ContainSubstring("command.task: ")),
					MatchError(ContainSubstring(
						"is not on "+rk.ResourceID().String(),
					)),
				))
			},
		)
		It("Should return not found for a grant to a missing task",
			func(ctx SpecContext) {
				cmd.Task = uuid.New()
				u := createUser(ctx)
				allowGrants(ctx, u)
				s := createSecret(ctx, "A", "v")
				Expect(createGrant(
					SubjectCtx(ctx, u.ResourceID()),
					grantReq(s.Base().Key),
				)).Error().To(MatchError(query.ErrNotFound))
			},
		)
		It("Should deny a grant to a user who cannot create devices",
			func(ctx SpecContext) {
				u := createUser(ctx)
				allow(ctx, u.ResourceID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				Expect(createGrant(
					SubjectCtx(ctx, u.ResourceID()),
					grantReq(s.Base().Key),
				)).Error().To(MatchError(access.ErrDenied))
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
				Expect(createGrant(
					SubjectCtx(ctx, u.ResourceID()),
					grantReq(s.Base().Key),
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should deny a grant to a rack", func(ctx SpecContext) {
			s := createSecret(ctx, "A", "v")
			Expect(createGrant(
				SubjectCtx(ctx, rk.ResourceID()),
				grantReq(s.Base().Key),
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
				Expect(resolve(AuthedCtx(ctx, u),
					apisecret.ResolveRequest{Grant: uuid.New(), Task: t.Key},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring("only a rack can resolve secrets")),
				))
			},
		)
		DescribeTable("Should refuse a wrong mix of grant, task, and secrets",
			func(ctx SpecContext, req apisecret.ResolveRequest, msg string) {
				Expect(resolve(SubjectCtx(ctx, rk.ResourceID()), req)).Error().
					To(SatisfyAll(
						BeAssignableToTypeOf(validate.PathError{}),
						MatchError(ContainSubstring(msg)),
					))
			},
			Entry("a grant without a task",
				apisecret.ResolveRequest{Grant: uuid.New()},
				"task: required with a grant",
			),
			Entry("a grant with secrets",
				apisecret.ResolveRequest{
					Grant:   uuid.New(),
					Task:    uuid.New(),
					Secrets: []secret.Key{uuid.New()},
				},
				"secrets: must be empty with a grant",
			),
			Entry("no grant and no secrets",
				apisecret.ResolveRequest{},
				"secrets: required without a grant",
			),
			Entry("a task without a grant",
				apisecret.ResolveRequest{
					Task:    uuid.New(),
					Secrets: []secret.Key{uuid.New()},
				},
				"task: must be empty without a grant",
			),
		)
	})

	Describe("Resolve without a grant", func() {
		var routingKey secret.Secret
		BeforeEach(func(ctx SpecContext) {
			routingKey = createSecret(ctx, "Routing key", "rk")
			t := task.Task{
				Type: pagerduty.AlertTaskType,
				Rack: svc.Rack.EmbeddedKey,
				Name: "Alerts",
				Config: msgpack.EncodedJSON{
					"routing_key": routingKey.Base().Key.String(),
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Task.NewWriter(tx).Create(ctx, &t)
			})).To(Succeed())
		})
		It("Should return the value of a secret that a task on the rack uses",
			func(ctx SpecContext) {
				rk := svc.Rack.EmbeddedKey
				res := MustSucceed(resolve(
					SubjectCtx(ctx, rk.ResourceID()),
					apisecret.ResolveRequest{
						Secrets: []secret.Key{routingKey.Base().Key},
					},
				))
				Expect(res.Command).To(Equal(task.Command{}))
				Expect(res.Values).To(Equal([]apisecret.Value{
					{Key: routingKey.Base().Key, Value: "rk"},
				}))
			},
		)
		It("Should deny a secret that only a task on another rack uses",
			func(ctx SpecContext) {
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(resolve(
					SubjectCtx(ctx, other.Key.ResourceID()),
					apisecret.ResolveRequest{
						Secrets: []secret.Key{routingKey.Base().Key},
					},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"no device or task on "+other.Key.ResourceID().String()+
							" uses secret "+routingKey.Base().Key.String(),
					)),
				))
			},
		)
		Describe("Through a certificate", func() {
			var (
				cert certificate.Certificate
				key  secret.Key
			)
			BeforeEach(func(ctx SpecContext) {
				cert = createCertificate(ctx)
				stored, ok := cert.Variant.(certificate.StoredCertificate)
				Expect(ok).To(BeTrue())
				key = *stored.PrivateKey
			})
			It("Should return the private key of a certificate a device uses",
				func(ctx SpecContext) {
					rk := svc.Rack.EmbeddedKey
					dev := device.Device{
						Key:      uuid.New().String(),
						Rack:     rk,
						Location: "opc.tcp://plc:4840",
						Make:     "opc",
						Model:    "OPC UA",
						Name:     "PLC",
						Properties: msgpack.EncodedJSON{"connection": map[string]any{
							"client_certificate": cert.Base().Key.String(),
						}},
					}
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.Device.NewWriter(tx).Create(ctx, &dev)
					})).To(Succeed())
					res := MustSucceed(resolve(
						SubjectCtx(ctx, rk.ResourceID()),
						apisecret.ResolveRequest{Secrets: []secret.Key{key}},
					))
					Expect(res.Values).To(HaveLen(1))
					Expect(res.Values[0].Key).To(Equal(key))
				},
			)
			It("Should deny the private key of a certificate that nothing uses",
				func(ctx SpecContext) {
					rk := svc.Rack.EmbeddedKey
					Expect(resolve(
						SubjectCtx(ctx, rk.ResourceID()),
						apisecret.ResolveRequest{Secrets: []secret.Key{key}},
					)).Error().To(SatisfyAll(
						MatchError(access.ErrDenied),
						MatchError(ContainSubstring("uses secret "+key.String())),
					))
				},
			)
		})
		It("Should deny a secret that nothing uses, with the used one",
			func(ctx SpecContext) {
				rk := svc.Rack.EmbeddedKey
				unused := createSecret(ctx, "Unused", "u")
				Expect(resolve(
					SubjectCtx(ctx, rk.ResourceID()),
					apisecret.ResolveRequest{Secrets: []secret.Key{
						routingKey.Base().Key,
						unused.Base().Key,
					}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						"uses secret "+unused.Base().Key.String(),
					)),
				))
			},
		)
	})
})
