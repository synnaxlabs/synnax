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
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Writer", func() {
	var e *env
	BeforeEach(func(ctx SpecContext) { e = newEnv(ctx, telem.Hour) })

	Describe("Create", func() {
		It("Should read the details of a stored certificate", func(ctx SpecContext) {
			pemText := newCertificate(newKey(), "plant", 24*time.Hour)
			c := e.create(ctx, stored("Plant", pemText), nil)
			block, _ := pem.Decode([]byte(pemText))
			sum1, sum256 := sha1.Sum(block.Bytes), sha256.Sum256(block.Bytes)
			s := e.retrieve(ctx, c.Base().Key).Variant.(certificate.StoredCertificate)
			Expect(s.Details).To(Equal(certificate.Details{
				Subject:   "CN=plant",
				Issuer:    "CN=plant",
				NotBefore: telem.NewTimeStamp(start.Add(-time.Hour)),
				NotAfter:  telem.NewTimeStamp(start.Add(24 * time.Hour)),
				Sha1:      hex.EncodeToString(sum1[:]),
				Sha256:    hex.EncodeToString(sum256[:]),
			}))
			Expect(s.PrivateKey).To(BeNil())
		})
		It("Should keep only the certificate blocks of the PEM", func(ctx SpecContext) {
			leaf := newCertificate(newKey(), "leaf", time.Hour)
			ca := newCertificate(newKey(), "ca", time.Hour)
			c := e.create(ctx, stored("Plant", "Bag Attributes\n"+leaf+"\n"+ca), nil)
			s := e.retrieve(ctx, c.Base().Key).Variant.(certificate.StoredCertificate)
			Expect(s.Pem).To(Equal(leaf + ca))
			Expect(s.Details.Subject).To(Equal("CN=leaf"))
		})
		It("Should save a matching private key as a secret", func(ctx SpecContext) {
			key := newKey()
			c := e.create(
				ctx,
				stored("Plant", newCertificate(key, "a", time.Hour)),
				encodeKey(key),
			)
			secretKey := c.Variant.(certificate.StoredCertificate).PrivateKey
			Expect(secretKey).ToNot(BeNil())
			var s secret.Secret
			Expect(e.secrets.NewRetrieve().
				Where(secret.MatchKeys(*secretKey)).
				Entry(&s).
				Exec(ctx, nil)).To(Succeed())
			Expect(s.Base().Name).To(Equal("Plant private key"))
			Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(key)))
		})
		It("Should ignore a private key secret the caller sets", func(ctx SpecContext) {
			planted := uuid.New()
			c := stored("Plant", newCertificate(newKey(), "a", time.Hour))
			variant := c.Variant.(certificate.StoredCertificate)
			variant.PrivateKey = &planted
			c.Variant = variant
			c = e.create(ctx, c, nil)
			Expect(c.Variant.(certificate.StoredCertificate).PrivateKey).To(BeNil())
		})
		DescribeTable(
			"Should reject an invalid stored certificate",
			func(
				ctx SpecContext,
				pemText func() string,
				privateKey func() []byte,
				msg string,
			) {
				c := stored("Plant", pemText())
				Expect(e.svc.NewWriter(nil).Create(ctx, &c, privateKey())).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry(
				"text with no certificate",
				func() string { return "not a certificate" },
				func() []byte { return nil },
				"pem: holds no certificate",
			),
			Entry(
				"a private key block",
				func() string {
					key := newKey()
					return newCertificate(key, "a", time.Hour) + string(encodeKey(key))
				},
				func() []byte { return nil },
				"pem: holds a PRIVATE KEY block. Send a private key as private_key",
			),
			Entry(
				"a block that does not parse",
				func() string {
					return string(pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: []byte("garbage"),
					}))
				},
				func() []byte { return nil },
				"pem: x509: malformed certificate",
			),
			Entry(
				"a private key for another certificate",
				func() string { return newCertificate(newKey(), "a", time.Hour) },
				func() []byte { return encodeKey(newKey()) },
				"private_key: private key does not match public key",
			),
			Entry(
				"a private key that is not PEM",
				func() string { return newCertificate(newKey(), "a", time.Hour) },
				func() []byte { return []byte("not a key") },
				"private_key: failed to find any PEM data in key input",
			),
		)
		It("Should reject a certificate with no name", func(ctx SpecContext) {
			c := stored("", newCertificate(newKey(), "a", time.Hour))
			Expect(e.svc.NewWriter(nil).Create(ctx, &c, nil)).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should reject a certificate with no source", func(ctx SpecContext) {
			c := certificate.Certificate{}
			Expect(e.svc.NewWriter(nil).Create(ctx, &c, nil)).
				To(MatchError(ContainSubstring("source: required")))
		})
		DescribeTable("Should reject an invalid file certificate",
			func(ctx SpecContext, path string, privateKey []byte, msg string) {
				c := certificate.Certificate{Variant: certificate.FileCertificate{
					Base:            certificate.Base{Name: "Plant"},
					CertificatePath: path,
				}}
				Expect(e.svc.NewWriter(nil).Create(ctx, &c, privateKey)).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("no path", "", nil, "certificate_path: required"),
			Entry(
				"a private key",
				"/c.pem",
				[]byte("key"),
				"private_key: must be empty for a file certificate",
			),
		)

		Describe("Replace", func() {
			It(
				"Should replace the value of the private key secret",
				func(ctx SpecContext) {
					oldKey, replacement := newKey(), newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(oldKey, "a", time.Hour)),
						encodeKey(oldKey),
					)
					secretKey := *c.Variant.(certificate.StoredCertificate).PrivateKey
					variant := c.Variant.(certificate.StoredCertificate)
					variant.Pem = newCertificate(replacement, "b", time.Hour)
					c.Variant = variant
					c = e.create(ctx, c, encodeKey(replacement))
					Expect(*c.Variant.(certificate.StoredCertificate).PrivateKey).
						To(Equal(secretKey))
					Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(replacement)))
				},
			)
			It("Should keep the private key for a renewal with the same key",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(key, "a", time.Hour)),
						encodeKey(key),
					)
					variant := c.Variant.(certificate.StoredCertificate)
					variant.Pem = newCertificate(key, "a", 48*time.Hour)
					variant.PrivateKey = nil
					c.Variant = variant
					c = e.create(ctx, c, nil)
					s := e.retrieve(ctx, c.Base().Key).
						Variant.(certificate.StoredCertificate)
					Expect(s.Details.NotAfter).
						To(Equal(telem.NewTimeStamp(start.Add(48 * time.Hour))))
					Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(key)))
				},
			)
			It("Should require a private key when the new certificate does not match",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(key, "a", time.Hour)),
						encodeKey(key),
					)
					variant := c.Variant.(certificate.StoredCertificate)
					variant.Pem = newCertificate(newKey(), "b", time.Hour)
					c.Variant = variant
					Expect(e.svc.NewWriter(nil).Create(ctx, &c, nil)).To(MatchError(
						ContainSubstring(
							"private_key: required, because the new certificate " +
								"does not match the saved key",
						),
					))
				},
			)
			It("Should delete the private key secret when a file replaces it",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(key, "a", time.Hour)),
						encodeKey(key),
					)
					secretKey := *c.Variant.(certificate.StoredCertificate).PrivateKey
					e.create(ctx, certificate.Certificate{
						Variant: certificate.FileCertificate{
							Base:            c.Base(),
							CertificatePath: "/c.pem",
						},
					}, nil)
					Expect(e.secrets.NewRetrieve().
						Where(secret.MatchKeys(secretKey)).
						Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
				},
			)
		})
	})

	Describe("Rename", func() {
		It("Should rename the certificate", func(ctx SpecContext) {
			c := e.create(
				ctx,
				stored("Before", newCertificate(newKey(), "a", time.Hour)),
				nil,
			)
			Expect(
				e.svc.NewWriter(nil).Rename(ctx, c.Base().Key, "After"),
			).To(Succeed())
			Expect(e.retrieve(ctx, c.Base().Key).Base().Name).To(Equal("After"))
		})
		It("Should reject an empty name", func(ctx SpecContext) {
			c := e.create(
				ctx,
				stored("Before", newCertificate(newKey(), "a", time.Hour)),
				nil,
			)
			Expect(e.svc.NewWriter(nil).Rename(ctx, c.Base().Key, "")).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return not found for a missing certificate", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(nil).Rename(ctx, uuid.New(), "A")).
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Delete", func() {
		It(
			"Should delete the certificate, its key, and its status",
			func(ctx SpecContext) {
				key := newKey()
				c := e.create(
					ctx,
					stored("Plant", newCertificate(key, "a", time.Hour)),
					encodeKey(key),
				)
				secretKey := *c.Variant.(certificate.StoredCertificate).PrivateKey
				Expect(e.svc.NewWriter(nil).Delete(ctx, c.Base().Key)).To(Succeed())
				Expect(e.svc.NewRetrieve().
					Where(certificate.MatchKeys(c.Base().Key)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
				Expect(e.secrets.NewRetrieve().
					Where(secret.MatchKeys(secretKey)).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
				Expect(e.otg.NewRetrieve().
					WhereIDs(c.OntologyID()).
					Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
			},
		)
		It("Should succeed for a missing certificate", func(ctx SpecContext) {
			Expect(e.svc.NewWriter(nil).Delete(ctx, uuid.New())).To(Succeed())
		})
		It("Should keep the secret service from deleting a key in use",
			func(ctx SpecContext) {
				key := newKey()
				c := e.create(
					ctx,
					stored("Plant", newCertificate(key, "a", time.Hour)),
					encodeKey(key),
				)
				secretKey := *c.Variant.(certificate.StoredCertificate).PrivateKey
				Expect(e.secrets.NewWriter(nil).Delete(ctx, secretKey)).To(MatchError(
					ContainSubstring(
						`cannot delete secret "Plant private key" because ` +
							`certificate "Plant" uses it`,
					),
				))
			},
		)
	})
})
