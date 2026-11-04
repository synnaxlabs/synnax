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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apicertificate "github.com/synnaxlabs/synnax/pkg/api/certificate"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	certificateType = ontology.ID{Type: ontology.ResourceTypeCertificate}
	secretType      = ontology.ID{Type: ontology.ResourceTypeSecret}
	create          = []access.Action{access.ActionCreate}
)

// newPair returns a PEM certificate and its PEM private key.
func newPair() (string, string) {
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
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}))
}

func createCertificate(ctx SpecContext, name string) certificate.Certificate {
	GinkgoHelper()
	certPEM, _ := newPair()
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: name},
		Pem:  certPEM,
	}}
	Expect(svc.Certificate.NewWriter(nil).Create(ctx, &c, nil)).To(Succeed())
	return c
}

func retrieveCertificate(ctx SpecContext, key certificate.Key) certificate.Certificate {
	GinkgoHelper()
	var c certificate.Certificate
	Expect(svc.Certificate.NewRetrieve().
		Where(certificate.MatchKeys(key)).
		Entry(&c).
		Exec(ctx, nil)).To(Succeed())
	return c
}

var _ = Describe("ServiceConfig", func() {
	full := func() apicertificate.ServiceConfig {
		return apicertificate.ServiceConfig{
			Certificate: &certificate.Service{},
			RBAC:        &rbac.Service{},
		}
	}
	DescribeTable("Should fail when a service is missing",
		func(clear func(*apicertificate.ServiceConfig), msg string) {
			cfg := full()
			clear(&cfg)
			Expect(cfg.Validate()).To(MatchError(msg))
		},
		Entry("certificate",
			func(c *apicertificate.ServiceConfig) { c.Certificate = nil },
			"certificate: must be non-nil"),
		Entry("rbac", func(c *apicertificate.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apicertificate.ServiceConfig{}.Override(cfg)
		Expect(merged.Certificate).To(BeIdenticalTo(cfg.Certificate))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It(
			"Should create a stored certificate with its private key",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType, secretType)
				certPEM, keyPEM := newPair()
				res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
						Source:     secret.SourceStored,
						Name:       "Plant",
						Pem:        certPEM,
						PrivateKey: keyPEM,
					}}},
				))
				stored := res.Certificates[0].Variant.(certificate.StoredCertificate)
				Expect(stored.Details.Subject).To(Equal("CN=plant"))
				Expect(stored.PrivateKey).ToNot(BeNil())
			},
		)
		It("Should create a file certificate with the details it is given",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				keyPath := "/etc/plant/key.pem"
				res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
						Source:          secret.SourceFile,
						Name:            "Plant",
						CertificatePath: "/etc/plant/cert.pem",
						PrivateKeyPath:  &keyPath,
						Details:         &certificate.Details{Subject: "CN=plant"},
					}}},
				))
				file := retrieveCertificate(ctx, res.Certificates[0].Base().Key).
					Variant.(certificate.FileCertificate)
				Expect(*file.PrivateKeyPath).To(Equal(keyPath))
				Expect(file.Details.Subject).To(Equal("CN=plant"))
			},
		)
		It("Should deny a user without create access", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
				apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
					Source:          secret.SourceFile,
					Name:            "Plant",
					CertificatePath: "/etc/plant/cert.pem",
				}}},
			)).Error().To(MatchError(access.ErrDenied))
		})
		It("Should require create access on secrets for a private key",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				certPEM, keyPEM := newPair()
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
						Source:     secret.SourceStored,
						Name:       "Plant",
						Pem:        certPEM,
						PrivateKey: keyPEM,
					}}},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It("Should create a certificate at an unused key without update access",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				key := uuid.New()
				res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
						Key:             key,
						Source:          secret.SourceFile,
						Name:            "Plant",
						CertificatePath: "/etc/plant/cert.pem",
					}}},
				))
				Expect(res.Certificates[0].Base().Key).To(Equal(key))
				Expect(retrieveCertificate(ctx, key).Base().Name).To(Equal("Plant"))
			},
		)
		It("Should deny replacing a certificate without update access",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				c := createCertificate(ctx, "Plant")
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
						Key:             c.Base().Key,
						Source:          secret.SourceFile,
						Name:            "Replaced",
						CertificatePath: "/etc/plant/cert.pem",
					}}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(u.OntologyID().String()+" cannot update "+
						c.OntologyID().String()+": access denied: auth error"),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should deny a request that also replaces a certificate",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				c := createCertificate(ctx, "Plant")
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apicertificate.CreateRequest{Certificates: []apicertificate.Input{
						{
							Key:             uuid.New(),
							Source:          secret.SourceFile,
							Name:            "New",
							CertificatePath: "/etc/plant/new.pem",
						},
						{
							Key:             c.Base().Key,
							Source:          secret.SourceFile,
							Name:            "Replaced",
							CertificatePath: "/etc/plant/cert.pem",
						},
					}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(u.OntologyID().String()+" cannot update "+
						c.OntologyID().String()+": access denied: auth error"),
				))
				Expect(retrieveCertificate(ctx, c.Base().Key)).To(Equal(c))
			},
		)
		It("Should let an Owner replace a certificate", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(svc.RBAC.Role.NewWriter(nil, true).AssignRole(
				ctx,
				u.OntologyID(),
				svc.RBAC.Builtin.OwnerKey,
			)).To(Succeed())
			c := createCertificate(ctx, "Plant")
			MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
				apicertificate.CreateRequest{Certificates: []apicertificate.Input{{
					Key:             c.Base().Key,
					Source:          secret.SourceFile,
					Name:            "Replaced",
					CertificatePath: "/etc/plant/cert.pem",
				}}},
			))
			Expect(
				retrieveCertificate(ctx, c.Base().Key).Base().Name,
			).To(Equal("Replaced"))
		})
		DescribeTable("Should reject an invalid input",
			func(ctx SpecContext, in apicertificate.Input, msg string) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), create, certificateType)
				Expect(apiSvc.Create(
					AuthedCtx(ctx, u),
					nil,
					apicertificate.CreateRequest{
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
				u.OntologyID(),
				[]access.Action{access.ActionRetrieve},
				certificateType,
			)
			c := createCertificate(ctx, "Plant")
			res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apicertificate.RetrieveRequest{Keys: []certificate.Key{c.Base().Key}},
			))
			Expect(res.Certificates).To(HaveLen(1))
			Expect(res.Certificates[0].Base().Name).To(Equal("Plant"))
		})
		It("Should deny a user without retrieve access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Plant")
			Expect(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apicertificate.RetrieveRequest{Keys: []certificate.Key{c.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Rename", func() {
		It(
			"Should rename the certificate when update is granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificate(ctx, "Before")
				grant(
					ctx,
					u.OntologyID(),
					[]access.Action{access.ActionUpdate},
					c.OntologyID(),
				)
				MustSucceed(apiSvc.Rename(AuthedCtx(ctx, u), nil,
					apicertificate.RenameRequest{Key: c.Base().Key, Name: "After"},
				))
				Expect(
					retrieveCertificate(ctx, c.Base().Key).Base().Name,
				).To(Equal("After"))
			},
		)
		It("Should deny a user without update access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Before")
			Expect(apiSvc.Rename(AuthedCtx(ctx, u), nil,
				apicertificate.RenameRequest{Key: c.Base().Key, Name: "After"},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It(
			"Should delete the certificate when delete is granted",
			func(ctx SpecContext) {
				u := createUser(ctx)
				c := createCertificate(ctx, "Plant")
				grant(
					ctx,
					u.OntologyID(),
					[]access.Action{access.ActionDelete},
					c.OntologyID(),
				)
				MustSucceed(apiSvc.Delete(AuthedCtx(ctx, u), nil,
					apicertificate.DeleteRequest{Keys: []certificate.Key{c.Base().Key}},
				))
				Expect(svc.Certificate.NewRetrieve().
					Where(certificate.MatchKeys(c.Base().Key)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
		It("Should deny a user without delete access", func(ctx SpecContext) {
			u := createUser(ctx)
			c := createCertificate(ctx, "Plant")
			Expect(apiSvc.Delete(AuthedCtx(ctx, u), nil,
				apicertificate.DeleteRequest{Keys: []certificate.Key{c.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
