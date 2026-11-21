// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package certificate_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apicertificate "github.com/synnaxlabs/synnax/pkg/api/certificate"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var (
	certificateType = resource.ID{Type: resource.TypeCertificate}
	secretType      = resource.ID{Type: resource.TypeSecret}
	statusType      = resource.ID{Type: resource.TypeStatus}
	rackType        = resource.ID{Type: resource.TypeRack}
	retrieves       = []access.Action{access.ActionRetrieve}
	creates         = []access.Action{access.ActionCreate}
)

func whereKeys(keys ...certificate.Key) apicertificate.RetrieveRequest {
	return apicertificate.RetrieveRequest{
		Where: &certificate.Where{Key: &filter.Set[certificate.Key]{In: keys}},
	}
}

func whereKeysWithUses(keys ...certificate.Key) apicertificate.RetrieveRequest {
	req := whereKeys(keys...)
	req.IncludeUses = true
	return req
}

func createCertificate(ctx SpecContext, name string) certificate.Certificate {
	GinkgoHelper()
	certPEM, _ := certtestutil.NewPair()
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: name},
		Pem:  certPEM,
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Certificate.NewWriter(tx).Set(ctx, &c, nil)
	})).To(Succeed())
	return c
}

func createCertificateWithKey(ctx SpecContext, name string) certificate.Certificate {
	GinkgoHelper()
	certPEM, keyPEM := certtestutil.NewPair()
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: name},
		Pem:  certPEM,
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Certificate.NewWriter(tx).Set(ctx, &c, []byte(keyPEM))
	})).To(Succeed())
	return c
}

// storedOf returns the stored variant of c.
func storedOf(c certificate.Certificate) certificate.StoredCertificate {
	GinkgoHelper()
	s, ok := c.Variant.(certificate.StoredCertificate)
	Expect(ok).To(BeTrue())
	return s
}

func retrieveCertificate(ctx SpecContext, key certificate.Key) certificate.Certificate {
	GinkgoHelper()
	var c certificate.Certificate
	Expect(svc.Certificate.NewRetrieve().
		Where(certificate.MatchKeys(key)).
		Entry(&c).
		Exec(ctx, db)).To(Succeed())
	return c
}

var _ = Describe("ServiceConfig", func() {
	full := func() apicertificate.ServiceConfig {
		return apicertificate.ServiceConfig{Certificate: &certificate.Service{}}
	}
	DescribeTable("Should fail when a service is missing",
		func(unset func(*apicertificate.ServiceConfig), msg string) {
			cfg := full()
			unset(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("certificate",
			func(c *apicertificate.ServiceConfig) { c.Certificate = nil },
			"certificate: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apicertificate.ServiceConfig{}.Override(cfg)
		Expect(merged.Certificate).To(BeIdenticalTo(cfg.Certificate))
	})
})

var _ = Describe("Service", func() {
	Describe("Set", func() {
		It(
			"Should create a stored certificate with its private key",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(
					ctx,
					u.ResourceID(),
					creates,
					certificateType,
					secretType,
					statusType,
				)
				certPEM, keyPEM := certtestutil.NewPair()
				res := MustSucceed(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Source:        secret.SourceStored,
						Name:          "Plant",
						Pem:           certPEM,
						PrivateKeyPem: keyPEM,
					}}},
				))
				stored := storedOf(res.Certificates[0])
				Expect(stored.Details.Subject).To(Equal("CN=plant"))
				Expect(stored.PrivateKey).ToNot(BeNil())
			},
		)
		It("Should create a file certificate with the details it is given",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType, statusType)
				grant(ctx, u.ResourceID(), retrieves, rackType)
				keyFile := "key.pem"
				res := MustSucceed(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Source:          secret.SourceFile,
						Name:            "Plant",
						CertificatePath: "/etc/plant/cert.pem",
						Rack:            svc.Rack.EmbeddedKey,
						PrivateKeyFile:  &keyFile,
						Details:         &certificate.Details{Subject: "CN=plant"},
					}}},
				))
				file, ok := retrieveCertificate(ctx, res.Certificates[0].Base().Key).
					Variant.(certificate.FileCertificate)
				Expect(ok).To(BeTrue())
				Expect(*file.PrivateKeyFile).To(Equal(keyFile))
				Expect(file.Details.Subject).To(Equal("CN=plant"))
			},
		)
		It("Should deny a user without create access", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(set(AuthedCtx(ctx, u),
				apicertificate.SetRequest{Certificates: []apicertificate.Input{{
					Source:          secret.SourceFile,
					Name:            "Plant",
					CertificatePath: "/etc/plant/cert.pem",
					Rack:            svc.Rack.EmbeddedKey,
				}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should require create access on secrets for a private key",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType, statusType)
				certPEM, keyPEM := certtestutil.NewPair()
				Expect(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Source:        secret.SourceStored,
						Name:          "Plant",
						Pem:           certPEM,
						PrivateKeyPem: keyPEM,
					}}},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should create a certificate at an unused key without update access",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType)
				grant(ctx, u.ResourceID(), retrieves, rackType)
				key := uuid.New()
				res := MustSucceed(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Key:             key,
						Source:          secret.SourceFile,
						Name:            "Plant",
						CertificatePath: "/etc/plant/cert.pem",
						Rack:            svc.Rack.EmbeddedKey,
					}}},
				))
				Expect(res.Certificates[0].Base().Key).To(Equal(key))
				Expect(retrieveCertificate(ctx, key).Base().Name).To(Equal("Plant"))
			},
		)
		It("Should deny replacing a certificate without update access",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType)
				c := createCertificate(ctx, "Plant")
				Expect(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Key:    c.Base().Key,
						Source: secret.SourceStored,
						Name:   "Replaced",
						Pem:    storedOf(c).Pem,
					}}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(u.ResourceID().String()+" cannot update "+
						c.ResourceID().String()+": access denied: auth error"),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should deny a request that also replaces a certificate",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType)
				grant(ctx, u.ResourceID(), retrieves, rackType)
				c := createCertificate(ctx, "Plant")
				Expect(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{
						{
							Key:             uuid.New(),
							Source:          secret.SourceFile,
							Name:            "New",
							CertificatePath: "/etc/plant/new.pem",
							Rack:            svc.Rack.EmbeddedKey,
						},
						{
							Key:    c.Base().Key,
							Source: secret.SourceStored,
							Name:   "Replaced",
							Pem:    storedOf(c).Pem,
						},
					}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(u.ResourceID().String()+" cannot update "+
						c.ResourceID().String()+": access denied: auth error"),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should let an Owner replace a certificate", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.User.NewWriter(tx).
					SetRole(ctx, svc.RBAC.Builtin.OwnerKey, u.Key)
			})).To(Succeed())
			c := createCertificate(ctx, "Plant")
			MustSucceed(set(AuthedCtx(ctx, u),
				apicertificate.SetRequest{Certificates: []apicertificate.Input{{
					Key:    c.Base().Key,
					Source: secret.SourceStored,
					Name:   "Replaced",
					Pem:    storedOf(c).Pem,
				}}},
			))
			Expect(
				retrieveCertificate(ctx, c.Base().Key).Base().Name,
			).To(Equal("Replaced"))
		})
		It("Should deny a replace that renames the key secret without update on it",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificateWithKey(ctx, "Plant")
				stored := storedOf(c)
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionUpdate},
					c.ResourceID(),
					statusType,
				)
				Expect(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Key:    c.Base().Key,
						Source: secret.SourceStored,
						Name:   "Replaced",
						Pem:    stored.Pem,
					}}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						u.ResourceID().String()+" cannot update "+
							secret.ResourceID(*stored.PrivateKey).String(),
					)),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should deny a file certificate on a rack the user cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType)
				Expect(set(AuthedCtx(ctx, u),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Source:          secret.SourceFile,
						Name:            "Plant",
						CertificatePath: "/etc/plant/cert.pem",
						Rack:            svc.Rack.EmbeddedKey,
					}}},
				)).Error().To(MatchError(
					u.ResourceID().String() + " cannot retrieve " +
						svc.Rack.EmbeddedKey.ResourceID().String() +
						": access denied: auth error",
				))
			},
		)
		DescribeTable("Should reject an invalid input",
			func(ctx SpecContext, in apicertificate.Input, msg string) {
				u := createUser(ctx)
				grant(ctx, u.ResourceID(), creates, certificateType)
				Expect(set(
					AuthedCtx(ctx, u),
					apicertificate.SetRequest{
						Certificates: []apicertificate.Input{in},
					},
				)).Error().To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"an unknown source",
				apicertificate.Input{Source: "cloud", Name: "A"},
				`source: must be stored or file, not "cloud"`,
			),
			Entry(
				"a stored certificate with a path",
				apicertificate.Input{
					Source:          secret.SourceStored,
					Name:            "A",
					CertificatePath: "/c.pem",
				},
				"certificate_path: must be empty for a stored certificate",
			),
			Entry(
				"a stored certificate with a rack",
				apicertificate.Input{
					Source: secret.SourceStored,
					Name:   "A",
					Pem:    "pem",
					Rack:   1,
				},
				"rack: must be empty for a stored certificate",
			),
			Entry(
				"a file certificate with PEM",
				apicertificate.Input{
					Source:          secret.SourceFile,
					Name:            "A",
					CertificatePath: "/c.pem",
					Pem:             "pem",
				},
				"pem: must be empty for a file certificate",
			),
		)
	})

	Describe("Retrieve", func() {
		It("Should retrieve certificates by key", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				certificateType,
			)
			c := createCertificate(ctx, "Plant")
			res := MustSucceed(
				retrieve(AuthedCtx(ctx, u), whereKeys(c.Base().Key)),
			)
			Expect(res.Certificates).To(HaveLen(1))
			Expect(res.Certificates[0].Base().Name).To(Equal("Plant"))
		})
		It("Should leave out a key that no certificate has", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionRetrieve},
				certificateType,
			)
			c := createCertificate(ctx, "Plant")
			res := MustSucceed(retrieve(
				AuthedCtx(ctx, u),
				whereKeys(c.Base().Key, uuid.New()),
			))
			Expect(res.Certificates).To(HaveLen(1))
			Expect(res.Certificates[0].Base().Key).To(Equal(c.Base().Key))
		})
		It("Should deny a user without retrieve access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Plant")
			Expect(retrieve(AuthedCtx(ctx, u), whereKeys(c.Base().Key))).
				Error().To(MatchError(access.ErrDenied))
		})
		Describe("IncludeUses", func() {
			var (
				u   user.User
				c   certificate.Certificate
				dev device.Device
			)
			BeforeEach(func(ctx SpecContext) {
				u = createUser(ctx)
				grant(ctx, u.ResourceID(), retrieves, certificateType)
				c = createCertificate(ctx, "Plant")
				dev = device.Device{
					Key:      uuid.New().String(),
					Rack:     svc.Rack.EmbeddedKey,
					Location: "opc.tcp://plc:4840",
					Make:     "opc",
					Model:    "OPC UA",
					Name:     "PLC",
					Properties: msgpack.EncodedJSON{"connection": map[string]any{
						"server_certificate": c.Base().Key.String(),
					}},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Device.NewWriter(tx).Create(ctx, &dev)
				})).To(Succeed())
			})
			It("Should return the resources that use each certificate",
				func(ctx SpecContext) {
					grant(ctx, u.ResourceID(), retrieves, device.ResourceID(dev.Key))
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeysWithUses(c.Base().Key)),
					)
					Expect(res.Certificates).To(HaveLen(1))
					Expect(res.Certificates[0].Base().Uses).
						To(Equal([]resource.ID{device.ResourceID(dev.Key)}))
				},
			)
			It("Should leave out the uses that the caller cannot read",
				func(ctx SpecContext) {
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeysWithUses(c.Base().Key)),
					)
					Expect(res.Certificates).To(HaveLen(1))
					Expect(res.Certificates[0].Base().Uses).To(Equal([]resource.ID{}))
				},
			)
			It("Should leave uses nil when the request does not ask for them",
				func(ctx SpecContext) {
					grant(ctx, u.ResourceID(), retrieves, device.ResourceID(dev.Key))
					res := MustSucceed(
						retrieve(AuthedCtx(ctx, u), whereKeys(c.Base().Key)),
					)
					Expect(res.Certificates).To(HaveLen(1))
					Expect(res.Certificates[0].Base().Uses).To(BeNil())
				},
			)
		})
		It(
			"Should leave out the certificates a user cannot retrieve",
			func(ctx SpecContext) {
				u := createUser(ctx)
				a := createCertificate(ctx, "A")
				b := createCertificate(ctx, "B")
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionRetrieve},
					a.ResourceID(),
				)
				res := MustSucceed(
					retrieve(AuthedCtx(ctx, u), apicertificate.RetrieveRequest{}),
				)
				keys := make([]certificate.Key, len(res.Certificates))
				for i, c := range res.Certificates {
					keys[i] = c.Base().Key
				}
				Expect(keys).To(ContainElement(a.Base().Key))
				Expect(keys).NotTo(ContainElement(b.Base().Key))
			},
		)
	})

	Describe("Dispatch", func() {
		setName := func(key certificate.Key) apicertificate.DispatchRequest {
			return apicertificate.DispatchRequest{
				Key: key,
				Actions: []certificate.Action{certificate.NewSetAction(
					certificate.SetPayload{Path: []string{"name"}, Value: "After"},
				)},
			}
		}
		It("Should apply the actions when update is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Before")
			grant(
				ctx,
				u.ResourceID(),
				[]access.Action{access.ActionUpdate},
				c.ResourceID(),
				statusType,
			)
			MustSucceed(dispatch(AuthedCtx(ctx, u), setName(c.Base().Key)))
			Expect(retrieveCertificate(ctx, c.Base().Key).Base().Name).
				To(Equal("After"))
		})
		It("Should deny a user without update access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Before")
			Expect(dispatch(AuthedCtx(ctx, u), setName(c.Base().Key))).
				Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring(
					u.ResourceID().String()+" cannot update "+c.ResourceID().String(),
				)),
			))
			Expect(retrieveCertificate(ctx, c.Base().Key).Base().Name).
				To(Equal("Before"))
		})
		It("Should deny a name edit without update access on the key secret",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificateWithKey(ctx, "Before")
				key := *storedOf(c).PrivateKey
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionUpdate},
					c.ResourceID(),
					statusType,
				)
				Expect(dispatch(AuthedCtx(ctx, u), setName(c.Base().Key))).
					Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						u.ResourceID().String()+" cannot update "+
							secret.ResourceID(key).String(),
					)),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key).Base().Name).
					To(Equal("Before"))
			},
		)
		It("Should rename the key secret with update access on it",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificateWithKey(ctx, "Before")
				key := *storedOf(c).PrivateKey
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionUpdate},
					c.ResourceID(),
					secret.ResourceID(key),
					statusType,
				)
				MustSucceed(dispatch(AuthedCtx(ctx, u), setName(c.Base().Key)))
				var s secret.Secret
				Expect(svc.Secret.NewRetrieve().
					Where(secret.MatchKeys(key)).
					Entry(&s).
					Exec(ctx, db)).To(Succeed())
				Expect(s.Base().Name).To(Equal("After private key"))
			},
		)
	})

	Describe("Delete", func() {
		It(
			"Should delete the certificate when delete is granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificate(ctx, "Plant")
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionDelete},
					c.ResourceID(),
					statusType,
				)
				MustSucceed(del(AuthedCtx(ctx, u),
					apicertificate.DeleteRequest{Keys: []certificate.Key{c.Base().Key}},
				))
				Expect(svc.Certificate.NewRetrieve().
					Where(certificate.MatchKeys(c.Base().Key)).
					Exec(ctx, db)).To(MatchError(query.ErrNotFound))
			},
		)
		It(
			"Should require delete access on the secret of a private key",
			func(ctx SpecContext) {
				owner := createUser(ctx)
				grant(
					ctx,
					owner.ResourceID(),
					creates,
					certificateType,
					secretType,
					statusType,
				)
				certPEM, keyPEM := certtestutil.NewPair()
				res := MustSucceed(set(AuthedCtx(ctx, owner),
					apicertificate.SetRequest{Certificates: []apicertificate.Input{{
						Source:        secret.SourceStored,
						Name:          "Plant",
						Pem:           certPEM,
						PrivateKeyPem: keyPEM,
					}}},
				))
				c := res.Certificates[0]
				key := *storedOf(c).PrivateKey
				u := createUser(ctx)
				grant(
					ctx,
					u.ResourceID(),
					[]access.Action{access.ActionDelete},
					c.ResourceID(),
					statusType,
				)
				Expect(del(AuthedCtx(ctx, u),
					apicertificate.DeleteRequest{Keys: []certificate.Key{c.Base().Key}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						u.ResourceID().String()+" cannot delete "+
							secret.ResourceID(key).String(),
					)),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should deny a user without delete access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Plant")
			Expect(del(AuthedCtx(ctx, u),
				apicertificate.DeleteRequest{Keys: []certificate.Key{c.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Refresh", func() {
		var (
			rk   rack.Key
			file certificate.Certificate
		)
		BeforeEach(func(ctx SpecContext) {
			rk = svc.Rack.EmbeddedKey
			file = certificate.Certificate{Variant: certificate.FileCertificate{
				Base:            certificate.Base{Name: "Plant"},
				CertificatePath: "/etc/plant/cert.pem",
				Rack:            rk,
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Certificate.NewWriter(tx).Set(ctx, &file, nil)
			})).To(Succeed())
		})
		details := certificate.Details{
			Subject:   "CN=plant",
			Issuer:    "CN=plant",
			NotBefore: telem.TimeStamp(1),
			NotAfter:  telem.TimeStamp(2),
			Sha1:      "aa",
			Sha256:    "bb",
		}
		It("Should save the details that the certificate's rack read",
			func(ctx SpecContext) {
				MustSucceed(refresh(
					SubjectCtx(ctx, rk.ResourceID()),
					apicertificate.RefreshRequest{
						Key:     file.Base().Key,
						Details: details,
					},
				))
				got, ok := retrieveCertificate(ctx, file.Base().Key).
					Variant.(certificate.FileCertificate)
				Expect(ok).To(BeTrue())
				Expect(got.Details).To(Equal(&details))
				Expect(got.CertificatePath).To(Equal("/etc/plant/cert.pem"))
			},
		)
		It("Should deny a rack that the certificate does not name",
			func(ctx SpecContext) {
				other := rack.Rack{Name: "Other"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.Rack.NewWriter(tx).Create(ctx, &other)
				})).To(Succeed())
				Expect(refresh(
					SubjectCtx(ctx, other.Key.ResourceID()),
					apicertificate.RefreshRequest{
						Key:     file.Base().Key,
						Details: details,
					},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring(
						`certificate "Plant" is read by `+rk.ResourceID().String()+
							", not "+other.Key.ResourceID().String(),
					)),
				))
				got, ok := retrieveCertificate(ctx, file.Base().Key).
					Variant.(certificate.FileCertificate)
				Expect(ok).To(BeTrue())
				Expect(got.Details).To(BeNil())
			},
		)
		It("Should deny a subject that is not a rack", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, u.ResourceID(), access.AllActions, certificateType)
			Expect(refresh(
				AuthedCtx(ctx, u),
				apicertificate.RefreshRequest{Key: file.Base().Key, Details: details},
			)).Error().To(SatisfyAll(
				MatchError(access.ErrDenied),
				MatchError(ContainSubstring("only a rack can refresh a certificate")),
			))
		})
		It("Should refuse a stored certificate", func(ctx SpecContext) {
			c := createCertificate(ctx, "Stored")
			Expect(refresh(
				SubjectCtx(ctx, rk.ResourceID()),
				apicertificate.RefreshRequest{Key: c.Base().Key, Details: details},
			)).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(ContainSubstring(
					`key: certificate "Stored" is stored, so the Core reads its details`,
				)),
			))
		})
		It("Should return not found for a missing certificate", func(ctx SpecContext) {
			Expect(refresh(
				SubjectCtx(ctx, rk.ResourceID()),
				apicertificate.RefreshRequest{Key: uuid.New(), Details: details},
			)).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
