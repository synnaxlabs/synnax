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
	"github.com/synnaxlabs/synnax/pkg/service/rack"
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
			s := storedOf(e.retrieve(ctx, c.Base().Key))
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
			s := storedOf(e.retrieve(ctx, c.Base().Key))
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
			secretKey := storedOf(c).PrivateKey
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
		It("Should match the private key of a certificate whose names Go refuses",
			func(ctx SpecContext) {
				key := newKey()
				spki := MustSucceed(x509.MarshalPKIXPublicKey(&key.PublicKey))
				pemText := string(pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: newRefusedCertificate(plcName(), spki),
				}))
				c := e.create(ctx, stored("PLC", pemText), encodeKey(key))
				Expect(storedOf(c).Details.Subject).To(Equal("CN=PLC_01"))
				Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(key)))
			})

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
			variant := storedOf(c)
			variant.PrivateKey = &planted
			c.Variant = variant
			c = e.create(ctx, c, nil)
			Expect(storedOf(c).PrivateKey).To(BeNil())
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
				"a private key for a certificate whose key Go cannot read",
				func() string {
					return string(pem.EncodeToMemory(&pem.Block{
						Type:  "CERTIFICATE",
						Bytes: newRefusedCertificate(plcName(), unreadableKey()),
					}))
				},
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
		It("Should refuse a file certificate on a rack that does not exist", func(
			ctx SpecContext,
		) {
			c := file("Plant", 9)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Set(ctx, &c, nil)
			})).To(MatchError(ContainSubstring("rack: rack 9 does not exist")))
		})
		It("Should refuse a file certificate with no rack", func(ctx SpecContext) {
			c := file("Plant", 0)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Set(ctx, &c, nil)
			})).To(MatchError(ContainSubstring("rack: must be non-zero")))
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
					Rack:            coreRack,
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
					secretKey := *storedOf(c).PrivateKey
					variant := storedOf(c)
					variant.Pem = newCertificate(replacement, "b", time.Hour)
					c.Variant = variant
					c = e.create(ctx, c, encodeKey(replacement))
					Expect(*storedOf(c).PrivateKey).
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
					variant := storedOf(c)
					variant.Pem = newCertificate(key, "a", 48*time.Hour)
					variant.PrivateKey = nil
					c.Variant = variant
					c = e.create(ctx, c, nil)
					s := storedOf(e.retrieve(ctx, c.Base().Key))
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
					variant := storedOf(c)
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
					variant := storedOf(c)
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
			It("Should refuse to add a private key to a certificate saved without one",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(key, "a", time.Hour)),
						nil,
					)
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &c, encodeKey(key))
					})).To(MatchError(ContainSubstring(
						"private_key_pem: must be empty for a certificate saved " +
							"without a private key",
					)))
					Expect(
						storedOf(e.retrieve(ctx, c.Base().Key)).PrivateKey,
					).To(BeNil())
				},
			)
			It("Should refuse a replace of a stored certificate with a file",
				func(ctx SpecContext) {
					key := newKey()
					c := e.create(
						ctx,
						stored("Plant", newCertificate(key, "a", time.Hour)),
						encodeKey(key),
					)
					next := file("Plant", coreRack)
					next.SetBase(c.Base())
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &next, nil)
					})).To(MatchError(ContainSubstring(
						"source: cannot change from stored to file",
					)))
					Expect(e.resolveKey(ctx, c)).To(Equal(encodeKey(key)))
				},
			)
			It(
				"Should refuse to add a private key file to a certificate saved without one",
				func(ctx SpecContext) {
					c := e.create(ctx, file("Plant", coreRack), nil)
					keyed := fileOf(c)
					keyed.PrivateKeyFile = new("client.pem")
					c.Variant = keyed
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &c, nil)
					})).To(MatchError(ContainSubstring(
						"private_key_file: must be empty for a certificate saved " +
							"without a private key file",
					)))
					Expect(
						fileOf(e.retrieve(ctx, c.Base().Key)).PrivateKeyFile,
					).To(BeNil())
				},
			)
			It("Should let a file certificate change its private key file",
				func(ctx SpecContext) {
					keyed := file("Plant", coreRack)
					variant := fileOf(keyed)
					variant.PrivateKeyFile = new("client.pem")
					keyed.Variant = variant
					c := e.create(ctx, keyed, nil)
					variant = fileOf(c)
					variant.PrivateKeyFile = new("other.pem")
					c.Variant = variant
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &c, nil)
					})).To(Succeed())
					Expect(
						*fileOf(e.retrieve(ctx, c.Base().Key)).PrivateKeyFile,
					).To(Equal("other.pem"))
				},
			)
			It("Should refuse a replace of a file certificate with a stored one",
				func(ctx SpecContext) {
					c := e.create(ctx, file("Plant", coreRack), nil)
					next := stored("Plant", newCertificate(newKey(), "a", time.Hour))
					next.SetBase(c.Base())
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &next, nil)
					})).To(MatchError(ContainSubstring(
						"source: cannot change from file to stored",
					)))
					Expect(e.retrieve(ctx, c.Base().Key)).To(Equal(c))
				},
			)
			It("Should move a file certificate that nothing uses to another rack",
				func(ctx SpecContext) {
					c := e.create(ctx, file("Plant", coreRack), nil)
					moved := file("Plant", otherRack)
					moved.SetBase(c.Base())
					e.create(ctx, moved, nil)
					Expect(
						fileOf(e.retrieve(ctx, c.Base().Key)).Rack,
					).To(Equal(otherRack))
				},
			)
			It(
				"Should refuse to move a file certificate that a record uses to another rack",
				func(ctx SpecContext) {
					c := e.create(ctx, file("Plant", coreRack), nil)
					listeners := MustOpen(gorp.OpenTable(
						ctx,
						gorp.TableConfig[string, listener]{
							DB:   e.db,
							Name: "listener",
							References: []gorp.Reference[string, listener]{
								gorp.NewKeyReference(gorp.KeyReferenceConfig[
									string,
									listener,
									certificate.Key,
								]{
									Name:     "certificate",
									Table:    "certificate",
									OnDelete: gorp.Restrict,
									Kind:     gorp.Use,
									Targets: func(l *listener) []certificate.Key {
										return []certificate.Key{l.Certificate}
									},
								}),
							},
						},
					))
					Expect(listeners.NewCreate().
						Entry(&listener{Key: "l1", Certificate: c.Base().Key}).
						Exec(ctx, e.db)).To(Succeed())
					moved := file("Plant", otherRack)
					moved.SetBase(c.Base())
					Expect(e.write(ctx, func(w certificate.Writer) error {
						return w.Set(ctx, &moved, nil)
					})).To(MatchError(
						"cannot change the rack of certificate " + c.Base().Key.String() +
							" because listener l1 references it through certificate: " +
							"validation error",
					))
					Expect(
						fileOf(e.retrieve(ctx, c.Base().Key)).Rack,
					).To(Equal(coreRack))
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

	Describe("Status", func() {
		// warn writes a warning that c owns, as the OPC UA migration does.
		warn := func(ctx SpecContext, c certificate.Certificate) {
			GinkgoHelper()
			st := status.Status[any]{
				Owner:   new(c.ResourceID()),
				Name:    c.Base().Name,
				Time:    telem.Now(),
				Variant: status.VariantWarning,
				Message: "Private key file renamed",
			}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.status.NewWriter(tx).Set(ctx, &st)
			})).To(Succeed())
		}
		It("Should keep the status of a certificate with no details on rename, under "+
			"the new name", func(ctx SpecContext) {
			c := e.create(ctx, file("Plant", coreRack), nil)
			warn(ctx, c)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, c.Base().Key, "Line")
			})).To(Succeed())
			st := e.expiry(ctx, c.Base().Key)
			Expect(st.Message).To(Equal("Private key file renamed"))
			Expect(st.Name).To(Equal("Line"))
		})
		It("Should rename a certificate with no details and no status", func(
			ctx SpecContext,
		) {
			c := e.create(ctx, file("Plant", coreRack), nil)
			Expect(e.write(ctx, func(w certificate.Writer) error {
				return w.Rename(ctx, c.Base().Key, "Line")
			})).To(Succeed())
			Expect(e.retrieve(ctx, c.Base().Key).Base().Name).To(Equal("Line"))
		})
		It("Should keep the status of a certificate with no details on a replace "+
			"with no details", func(ctx SpecContext) {
			c := e.create(ctx, file("Plant", coreRack), nil)
			warn(ctx, c)
			replaced := file("Plant", coreRack)
			replaced.SetBase(c.Base())
			variant := fileOf(replaced)
			variant.CertificatePath = "/etc/plant/new.pem"
			replaced.Variant = variant
			e.create(ctx, replaced, nil)
			Expect(e.expiry(ctx, c.Base().Key).Message).
				To(Equal("Private key file renamed"))
		})
		It("Should delete the expiry status of a certificate that loses its details",
			func(ctx SpecContext) {
				c := file("Plant", coreRack)
				variant := fileOf(c)
				variant.Details = &certificate.Details{
					NotAfter: telem.NewTimeStamp(start.Add(time.Hour)),
				}
				c.Variant = variant
				c = e.create(ctx, c, nil)
				Expect(e.expiry(ctx, c.Base().Key).Message).
					To(Equal("Certificate expires soon"))
				replaced := file("Plant", coreRack)
				replaced.SetBase(c.Base())
				e.create(ctx, replaced, nil)
				var found []status.Status[any]
				Expect(e.status.NewRetrieve[any]().
					Where(status.MatchOwners[any](c.ResourceID())).
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				Expect(found).To(BeEmpty())
			},
		)
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
				secretKey := *storedOf(c).PrivateKey
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
		It("Should delete the file certificates of a rack that is deleted", func(
			ctx SpecContext,
		) {
			c := file("Plant", otherRack)
			variant := fileOf(c)
			variant.Details = &certificate.Details{
				NotAfter: telem.NewTimeStamp(start.Add(time.Hour)),
			}
			c.Variant = variant
			c = e.create(ctx, c, nil)
			kept := e.create(ctx, file("Core", coreRack), nil)
			Expect(e.racks.NewDelete().
				Where(gorp.MatchKeys[rack.Key, rack.Rack](otherRack)).
				Exec(ctx, e.db)).To(Succeed())
			Expect(e.svc.NewRetrieve().
				Where(certificate.MatchKeys(c.Base().Key)).
				Exec(ctx, e.db)).To(MatchError(query.ErrNotFound))
			var found []status.Status[any]
			Expect(e.status.NewRetrieve[any]().
				Where(status.MatchOwners[any](c.ResourceID())).
				Entries(&found).
				Exec(ctx, e.db)).To(Succeed())
			Expect(found).To(BeEmpty())
			Expect(e.retrieve(ctx, kept.Base().Key).Base().Name).To(Equal("Core"))
		})
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
				secretKey := *storedOf(c).PrivateKey
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
