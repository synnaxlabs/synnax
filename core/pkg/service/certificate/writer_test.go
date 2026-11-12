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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"math/big"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Writer", func() {
	var e *env
	BeforeEach(func(ctx SpecContext) { e = newEnv(ctx, telem.Hour) })

	Describe("Set", func() {
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
				Exec(ctx, e.db)).To(Succeed())
			Expect(s.Base().Name).To(Equal("Plant private key"))
			Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(key)))
		})
		DescribeTable("Should accept each PEM private key form",
			func(ctx SpecContext, encode func(*ecdsa.PrivateKey) []byte) {
				key := newKey()
				c := e.create(
					ctx,
					stored("Plant", newCertificate(key, "a", time.Hour)),
					encode(key),
				)
				Expect(e.resolveKey(ctx, c)).To(Equal(encode(key)))
			},
			Entry("PKCS #8", encodeKey),
			Entry("SEC 1", encodeSEC1Key),
			Entry("SEC 1 after EC PARAMETERS", func(key *ecdsa.PrivateKey) []byte {
				GinkgoHelper()
				params := pem.EncodeToMemory(&pem.Block{
					Type:  "EC PARAMETERS",
					Bytes: MustSucceed(asn1.Marshal(oidP256)),
				})
				return append(params, encodeSEC1Key(key)...)
			}),
		)
		It("Should accept a PKCS #1 RSA private key", func(ctx SpecContext) {
			key := MustSucceed(rsa.GenerateKey(rand.Reader, 2048))
			tmpl := &x509.Certificate{
				SerialNumber: big.NewInt(1),
				Subject:      pkix.Name{CommonName: "rsa"},
				NotBefore:    time.Now().Add(-time.Hour),
				NotAfter:     time.Now().Add(time.Hour),
			}
			der := MustSucceed(
				x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
			)
			certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
			keyPEM := pem.EncodeToMemory(&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(key),
			})
			c := e.create(ctx, stored("RSA", string(certPEM)), keyPEM)
			Expect(e.resolveKey(ctx, c)).To(Equal(keyPEM))
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
				Expect(e.write(ctx, func(w certificate.Writer) error {
					return w.Set(ctx, &c, privateKey())
				})).
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
				"pem: holds a PRIVATE KEY block. Send a private key as private_key_pem",
			),
			Entry(
				"a block that is not a certificate",
				func() string {
					return newCertificate(newKey(), "a", time.Hour) +
						string(pem.EncodeToMemory(&pem.Block{Type: "X509 CRL"}))
				},
				func() []byte { return nil },
				"pem: holds a X509 CRL block, not a certificate",
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
				"private_key_pem: does not match the certificate",
			),
			Entry(
				"a private key that is not PEM",
				func() string { return newCertificate(newKey(), "a", time.Hour) },
				func() []byte { return []byte("not a key") },
				"private_key_pem: is not a PEM private key",
			),
			Entry(
				"a certificate in place of a private key",
				func() string { return newCertificate(newKey(), "a", time.Hour) },
				func() []byte {
					return []byte(newCertificate(newKey(), "b", time.Hour))
				},
				"private_key_pem: is not a PEM private key",
			),
			Entry(
				"a private key of a type Synnax cannot read",
				func() string { return newCertificate(newKey(), "a", time.Hour) },
				func() []byte {
					return pem.EncodeToMemory(&pem.Block{
						Type:  "OPENSSH PRIVATE KEY",
						Bytes: []byte("key"),
					})
				},
				"private_key_pem: is a key type Synnax cannot read",
			),
		)
		It("Should reject a certificate with no name", func(ctx SpecContext) {
			c := stored("", newCertificate(newKey(), "a", time.Hour))
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Set(ctx, &c, nil)
			})).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should reject a certificate with no source", func(ctx SpecContext) {
			c := certificate.Certificate{}
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Set(ctx, &c, nil)
			})).
				To(MatchError(ContainSubstring("source: required")))
		})
		DescribeTable("Should reject an invalid file certificate",
			func(
				ctx SpecContext,
				path string,
				keyFile *string,
				privateKey []byte,
				msg string,
			) {
				c := certificate.Certificate{Variant: certificate.FileCertificate{
					Base:            certificate.Base{Name: "Plant"},
					CertificatePath: path,
					PrivateKeyFile:  keyFile,
				}}
				Expect(e.write(ctx, func(w certificate.Writer) error {
					return w.Set(ctx, &c, privateKey)
				})).
					To(MatchError(ContainSubstring(msg)))
			},
			Entry("no path", "", nil, nil, "certificate_path: required"),
			Entry(
				"a private key",
				"/c.pem",
				nil,
				[]byte("key"),
				"private_key_pem: must be empty for a file certificate",
			),
			Entry(
				"an empty private key file",
				"/c.pem",
				new(""),
				nil,
				"private_key_file: required",
			),
			Entry(
				"a private key file that is a path",
				"/c.pem",
				new("/etc/ssl/private/other.key"),
				nil,
				"private_key_file: must be a file name, not a path",
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
			It("Should rename the private key secret with the certificate",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Before", newCertificate(key, "a", time.Hour)),
						encodeKey(key),
					)
					variant := c.Variant.(certificate.StoredCertificate)
					variant.Name = "After"
					c.Variant = variant
					c = e.create(ctx, c, nil)
					Expect(e.keyName(ctx, c)).To(Equal("After private key"))
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
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &c, nil)
					})).To(MatchError(
						ContainSubstring(
							"private_key_pem: required, because the new certificate " +
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
						Exec(ctx, e.db)).To(MatchError(query.ErrNotFound))
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
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, c.Base().Key, "After")
			})).To(Succeed())
			Expect(e.retrieve(ctx, c.Base().Key).Base().Name).To(Equal("After"))
			Expect(e.expiry(ctx, c.Base().Key).Name).To(Equal("After"))
		})
		It("Should rename the private key secret", func(ctx SpecContext) {
			key := newKey()
			c := e.create(
				ctx,
				stored("Before", newCertificate(key, "a", time.Hour)),
				encodeKey(key),
			)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, c.Base().Key, "After")
			})).To(Succeed())
			Expect(e.keyName(ctx, c)).To(Equal("After private key"))
		})
		It("Should reject an empty name", func(ctx SpecContext) {
			c := e.create(
				ctx,
				stored("Before", newCertificate(newKey(), "a", time.Hour)),
				nil,
			)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, c.Base().Key, "")
			})).
				To(MatchError(ContainSubstring("name: required")))
		})
		It("Should return not found for a missing certificate", func(ctx SpecContext) {
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, uuid.New(), "A")
			})).
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
				Expect(e.write(ctx, func(w certificate.Writer) error {
					return w.Delete(ctx, c.Base().Key)
				})).To(Succeed())
				Expect(e.svc.NewRetrieve().
					Where(certificate.MatchKeys(c.Base().Key)).
					Exec(ctx, e.db)).To(MatchError(query.ErrNotFound))
				Expect(e.secrets.NewRetrieve().
					Where(secret.MatchKeys(secretKey)).
					Exec(ctx, e.db)).To(MatchError(query.ErrNotFound))
				var found []status.Status[any]
				Expect(e.status.NewRetrieve[any]().
					Where(status.MatchOwners[any](c.ResourceID())).
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				Expect(found).To(BeEmpty())
			},
		)
		It("Should succeed for a missing certificate", func(ctx SpecContext) {
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Delete(ctx, uuid.New())
			})).To(Succeed())
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
				Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
					return e.secrets.NewWriter(tx).Delete(ctx, secretKey)
				})).To(
					MatchError(gorp.RestrictError{
						Target:    gorp.NewTarget("secret", secretKey),
						Holder:    gorp.NewTarget("certificate", c.Base().Key),
						Reference: "private_key",
					}),
				)
			},
		)
	})
})
