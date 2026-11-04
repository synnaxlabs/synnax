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
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	secretType = ontology.ID{Type: ontology.ResourceTypeSecret}
	deviceType = ontology.ID{Type: ontology.ResourceTypeDevice}
	all        = access.AllActions
)

func createSecret(ctx SpecContext, name, value string) secret.Secret {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: name}}}
	Expect(svc.Secret.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
	return s
}

func retrieveSecret(ctx SpecContext, key secret.Key) secret.Secret {
	GinkgoHelper()
	var s secret.Secret
	Expect(svc.Secret.NewRetrieve().
		Where(secret.MatchKeys(key)).
		Entry(&s).
		Exec(ctx, nil)).To(Succeed())
	return s
}

// createCertificate creates a stored certificate with a private key.
func createCertificate(ctx SpecContext) certificate.Certificate {
	GinkgoHelper()
	key := MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "plant"},
		NotAfter:     time.Now().Add(time.Hour),
	}
	der := MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
	)
	keyDER := MustSucceed(x509.MarshalPKCS8PrivateKey(key))
	c := certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: "Plant"},
		Pem:  string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
	}}
	Expect(svc.Certificate.NewWriter(nil).Create(
		ctx,
		&c,
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
	)).To(Succeed())
	return c
}

var _ = Describe("ServiceConfig", func() {
	full := func() apisecret.ServiceConfig {
		return apisecret.ServiceConfig{
			Secret:      &secret.Service{},
			Certificate: &certificate.Service{},
			RBAC:        &rbac.Service{},
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
		Entry("certificate", func(c *apisecret.ServiceConfig) { c.Certificate = nil },
			"certificate: must be non-nil"),
		Entry("rbac", func(c *apisecret.ServiceConfig) { c.RBAC = nil },
			"rbac: must be non-nil"),
	)
	It("Should take the other config's services when set", func() {
		cfg := full()
		Expect(cfg.Validate()).To(Succeed())
		merged := apisecret.ServiceConfig{}.Override(cfg)
		Expect(merged.Secret).To(BeIdenticalTo(cfg.Secret))
		Expect(merged.Certificate).To(BeIdenticalTo(cfg.Certificate))
		Expect(merged.RBAC).To(BeIdenticalTo(cfg.RBAC))
	})
})

var _ = Describe("Service", func() {
	Describe("Create", func() {
		It("Should create stored and file secrets and never return a value",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), all, secretType)
				res := MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{
						{
							Source: secret.SourceStored,
							Name:   "Password",
							Value:  "hunter2",
						},
						{Source: secret.SourceFile, Name: "Key", Path: "/k.pem"},
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
				grant(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "Before", "a")
				MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{{
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
				grant(ctx, u.OntologyID(), all, secretType)
				Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
					apisecret.CreateRequest{Secrets: []apisecret.Input{in}},
				)).Error().To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"an unknown source",
				apisecret.Input{Source: "cloud", Name: "A", Value: "v"},
				`source: must be stored or file, not "cloud"`,
			),
			Entry(
				"a stored secret with a path",
				apisecret.Input{
					Source: secret.SourceStored,
					Name:   "A",
					Value:  "v",
					Path:   "/a",
				},
				"path: must be empty for a stored secret",
			),
			Entry(
				"a file secret with a value",
				apisecret.Input{
					Source: secret.SourceFile,
					Name:   "A",
					Value:  "v",
					Path:   "/a",
				},
				"value: must be empty for a file secret",
			),
		)
		It("Should deny a user without create access", func(ctx SpecContext) {
			u := createUser(ctx)
			Expect(apiSvc.Create(AuthedCtx(ctx, u), nil,
				apisecret.CreateRequest{Secrets: []apisecret.Input{
					{Source: secret.SourceStored, Name: "A", Value: "v"},
				}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Retrieve", func() {
		It("Should retrieve secrets by key", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionRetrieve},
				secretType,
			)
			s := createSecret(ctx, "Password", "hunter2")
			res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apisecret.RetrieveRequest{Keys: []secret.Key{s.Base().Key}},
			))
			Expect(res.Secrets).To(HaveLen(1))
			Expect(res.Secrets[0].Base().Name).To(Equal("Password"))
		})
		It("Should deny a user without retrieve access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "hunter2")
			Expect(apiSvc.Retrieve(AuthedCtx(ctx, u),
				apisecret.RetrieveRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Rename", func() {
		It("Should rename the secret when update is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			grant(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionUpdate},
				s.OntologyID(),
			)
			MustSucceed(apiSvc.Rename(AuthedCtx(ctx, u), nil,
				apisecret.RenameRequest{Key: s.Base().Key, Name: "After"},
			))
			Expect(retrieveSecret(ctx, s.Base().Key).Base().Name).To(Equal("After"))
		})
		It("Should deny a user without update access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Before", "a")
			Expect(apiSvc.Rename(AuthedCtx(ctx, u), nil,
				apisecret.RenameRequest{Key: s.Base().Key, Name: "After"},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Delete", func() {
		It("Should delete the secret when delete is granted", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			grant(
				ctx,
				u.OntologyID(),
				[]access.Action{access.ActionDelete},
				s.OntologyID(),
			)
			MustSucceed(apiSvc.Delete(AuthedCtx(ctx, u), nil,
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			))
			Expect(svc.Secret.NewRetrieve().
				Where(secret.MatchKeys(s.Base().Key)).
				Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
		})
		It("Should deny a user without delete access", func(ctx SpecContext) {
			u := createUser(ctx)
			s := createSecret(ctx, "Password", "a")
			Expect(apiSvc.Delete(AuthedCtx(ctx, u), nil,
				apisecret.DeleteRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})

	Describe("Grant and Resolve", func() {
		rk := rack.Key(7)

		It("Should let the rack resolve the granted secrets in request order",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(
					ctx,
					u.OntologyID(),
					[]access.Action{access.ActionCreate},
					deviceType,
				)
				a := createSecret(ctx, "A", "first")
				b := createSecret(ctx, "B", "second")
				keys := []secret.Key{b.Base().Key, a.Base().Key}
				res := MustSucceed(apiSvc.Grant(AuthedCtx(ctx, u), nil,
					apisecret.GrantRequest{Rack: rk, Secrets: keys},
				))
				values := MustSucceed(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
					apisecret.ResolveRequest{Keys: keys, Grant: res.Grant},
				))
				Expect(values.Values).To(Equal([]apisecret.Value{
					{Key: b.Base().Key, Value: "second"},
					{Key: a.Base().Key, Value: "first"},
				}))
			},
		)
		It("Should grant the private keys of certificates", func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, u.OntologyID(), []access.Action{access.ActionCreate}, deviceType)
			c := createCertificate(ctx)
			res := MustSucceed(apiSvc.Grant(AuthedCtx(ctx, u), nil,
				apisecret.GrantRequest{
					Rack:         rk,
					Certificates: []certificate.Key{c.Base().Key},
				},
			))
			key := *c.Variant.(certificate.StoredCertificate).PrivateKey
			values := MustSucceed(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
				apisecret.ResolveRequest{Keys: []secret.Key{key}, Grant: res.Grant},
			))
			Expect(values.Values[0].Value).To(ContainSubstring("PRIVATE KEY"))
		})
		It("Should deny a grant to a user who cannot create devices",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				Expect(apiSvc.Grant(
					AuthedCtx(ctx, u),
					nil,
					apisecret.GrantRequest{
						Rack:    rk,
						Secrets: []secret.Key{s.Base().Key},
					},
				)).Error().To(MatchError(access.ErrDenied))
			},
		)
		It(
			"Should deny resolve to a subject that is not a rack",
			func(ctx SpecContext) {
				u := createUser(ctx)
				grant(ctx, u.OntologyID(), all, secretType)
				s := createSecret(ctx, "A", "v")
				Expect(apiSvc.Resolve(AuthedCtx(ctx, u),
					apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
				)).Error().To(SatisfyAll(
					MatchError(access.ErrDenied),
					MatchError(ContainSubstring("only a rack can resolve secrets")),
				))
			},
		)
		It("Should deny resolve without a grant", func(ctx SpecContext) {
			s := createSecret(ctx, "A", "v")
			Expect(apiSvc.Resolve(SubjectCtx(ctx, rk.OntologyID()),
				apisecret.ResolveRequest{Keys: []secret.Key{s.Base().Key}},
			)).Error().To(MatchError(access.ErrDenied))
		})
	})
})
