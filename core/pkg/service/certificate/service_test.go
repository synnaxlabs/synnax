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
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/certificate/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var start = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

// coreRack is the Core's rack in each env, and otherRack is another rack.
const (
	coreRack  rack.Key = 1
	otherRack rack.Key = 2
)

// env is a certificate service and the services it needs, on one in-memory DB, with a
// clock that each spec controls. The DB holds coreRack and otherRack.
type env struct {
	db      *gorp.DB
	racks   *gorp.Table[rack.Key, rack.Rack]
	secrets *secret.Service
	status  *status.Service
	svc     *certificate.Service
	now     atomic.Int64
}

func newEnv(ctx SpecContext, interval telem.TimeSpan) *env {
	GinkgoHelper()
	e := newStore(ctx)
	e.open(ctx, interval)
	return e
}

// newStore returns an env with the services that a certificate service needs, and no
// certificate service.
func newStore(ctx SpecContext) *env {
	GinkgoHelper()
	e := &env{}
	e.now.Store(int64(telem.NewTimeStamp(start)))
	e.db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(e.db.Run)
	e.status = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: e.db}))
	e.racks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[rack.Key, rack.Rack]{
		DB:   e.db,
		Name: "rack",
	}))
	for _, k := range []rack.Key{coreRack, otherRack} {
		Expect(e.racks.NewCreate().
			Entry(&rack.Rack{Key: k, Name: "Rack"}).
			Exec(ctx, e.db)).To(Succeed())
	}
	grants := MustOpen(grant.OpenService(ctx, grant.ServiceConfig{DB: e.db}))
	e.secrets = MustOpen(secret.OpenService(ctx, secret.ServiceConfig{
		DB:            e.db,
		Grant:         grants,
		Now:           e.clock,
		EncryptionKey: secret.NewEncryptionKey(),
		Rack:          coreRack,
	}))
	return e
}

// open opens the certificate service of e, which updates expiry statuses each interval.
func (e *env) open(ctx SpecContext, interval telem.TimeSpan) {
	GinkgoHelper()
	e.svc = MustOpen(certificate.OpenService(ctx, certificate.ServiceConfig{
		DB:                  e.db,
		Secret:              e.secrets,
		Status:              e.status,
		Now:                 e.clock,
		ExpiryCheckInterval: interval,
		Rack:                coreRack,
	}))
}

func (e *env) clock() telem.TimeStamp { return telem.TimeStamp(e.now.Load()) }

func (e *env) advance(span telem.TimeSpan) { e.now.Add(int64(span)) }

func newKey() *ecdsa.PrivateKey {
	GinkgoHelper()
	return MustSucceed(ecdsa.GenerateKey(elliptic.P256(), rand.Reader))
}

// newCertificate returns a PEM certificate for key, signed by key, that expires after
// lifetime from the start of each spec.
func newCertificate(key *ecdsa.PrivateKey, name string, lifetime time.Duration) string {
	GinkgoHelper()
	return newCertificateBetween(key, name, start.Add(-time.Hour), start.Add(lifetime))
}

// newCertificateBetween returns a PEM certificate for key, signed by key, that is valid
// from notBefore to notAfter.
func newCertificateBetween(
	key *ecdsa.PrivateKey,
	name string,
	notBefore, notAfter time.Time,
) string {
	GinkgoHelper()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}
	der := MustSucceed(
		x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key),
	)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

func encodeKey(key *ecdsa.PrivateKey) []byte {
	GinkgoHelper()
	der := MustSucceed(x509.MarshalPKCS8PrivateKey(key))
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func encodeSEC1Key(key *ecdsa.PrivateKey) []byte {
	GinkgoHelper()
	der := MustSucceed(x509.MarshalECPrivateKey(key))
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

// oidP256 names the P-256 curve, as openssl writes it in an EC PARAMETERS block.
var oidP256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}

func stored(name, pemText string) certificate.Certificate {
	return certificate.Certificate{Variant: certificate.StoredCertificate{
		Base: certificate.Base{Name: name},
		Pem:  pemText,
	}}
}

// file returns a file certificate named name, on the rack r.
func file(name string, r rack.Key) certificate.Certificate {
	return certificate.Certificate{Variant: certificate.FileCertificate{
		Base:            certificate.Base{Name: name},
		CertificatePath: "/etc/plant/cert.pem",
		Rack:            r,
	}}
}

// fileOf returns the file variant of c.
func fileOf(c certificate.Certificate) certificate.FileCertificate {
	GinkgoHelper()
	f, ok := c.Variant.(certificate.FileCertificate)
	Expect(ok).To(BeTrue())
	return f
}

// listener stands in for a record that uses a certificate.
type listener struct {
	Key         string          `json:"key"         msgpack:"key"`
	Certificate certificate.Key `json:"certificate" msgpack:"certificate"`
}

func (l listener) GorpKey() string { return l.Key }

func (l listener) EncodeOrc(w *orc.Writer) error {
	w.String(l.Key)
	w.Write(l.Certificate[:])
	return nil
}

func (l *listener) DecodeOrc(r *orc.Reader) (err error) {
	if l.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(l.Certificate[:])
	return err
}

// storedOf returns the stored variant of c.
func storedOf(c certificate.Certificate) certificate.StoredCertificate {
	GinkgoHelper()
	s, ok := c.Variant.(certificate.StoredCertificate)
	Expect(ok).To(BeTrue())
	return s
}

func (e *env) create(
	ctx SpecContext,
	c certificate.Certificate,
	privateKey []byte,
) certificate.Certificate {
	GinkgoHelper()
	Expect(e.write(ctx, func(w certificate.Writer) error {
		return w.Set(ctx, &c, privateKey)
	})).To(Succeed())
	return c
}

// write runs fn with a writer on a new transaction.
func (e *env) write(ctx context.Context, fn func(certificate.Writer) error) error {
	return e.db.WithTx(ctx, func(tx gorp.Tx) error { return fn(e.svc.NewWriter(tx)) })
}

func (e *env) retrieve(ctx SpecContext, key certificate.Key) certificate.Certificate {
	GinkgoHelper()
	var c certificate.Certificate
	Expect(e.svc.NewRetrieve().
		Where(certificate.MatchKeys(key)).
		Entry(&c).
		Exec(ctx, e.db)).To(Succeed())
	return c
}

func (e *env) expiry(ctx SpecContext, key certificate.Key) status.Status[any] {
	GinkgoHelper()
	var s status.Status[any]
	Expect(e.status.NewRetrieve[any]().
		Where(status.MatchOwners[any](certificate.ResourceID(key))).
		Entry(&s).
		Exec(ctx, e.db)).To(Succeed())
	return s
}

// resolveKey reads the private key secret of c.
func (e *env) resolveKey(ctx SpecContext, c certificate.Certificate) []byte {
	GinkgoHelper()
	key := *storedOf(c).PrivateKey
	return MustSucceed(e.secrets.Reveal(ctx, e.db, key))[key]
}

// keyName returns the name of the private key secret of c.
func (e *env) keyName(ctx SpecContext, c certificate.Certificate) string {
	GinkgoHelper()
	var s secret.Secret
	Expect(e.secrets.NewRetrieve().
		Where(secret.MatchKeys(*storedOf(c).PrivateKey)).
		Entry(&s).
		Exec(ctx, e.db)).To(Succeed())
	return s.Base().Name
}

var _ = Describe("Service", func() {
	It("Should require the secret and status services and the rack", func(
		ctx SpecContext,
	) {
		Expect(certificate.OpenService(ctx, certificate.ServiceConfig{})).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("secret: must be non-nil")),
				MatchError(ContainSubstring("status: must be non-nil")),
				MatchError(ContainSubstring("rack: must be non-zero")),
			))
	})
	It("Should move each stored v0 file certificate to the Core's rack", func(
		ctx SpecContext,
	) {
		e := newStore(ctx)
		old := MustSucceed(gorp.OpenTable(
			ctx,
			gorp.TableConfig[v0.Key, v0.Certificate]{DB: e.db, Name: "certificate"},
		))
		base := v0.Base{Key: uuid.New(), Name: "Plant"}
		Expect(gorp.NewCreate[v0.Key, v0.Certificate]().
			Entry(&v0.Certificate{Variant: v0.FileCertificate{
				Base:            base,
				CertificatePath: "/c.pem",
			}}).
			Exec(ctx, e.db)).To(Succeed())
		Expect(old.Close()).To(Succeed())
		e.open(ctx, telem.Hour)
		Expect(e.retrieve(ctx, base.Key)).To(Equal(certificate.Certificate{
			Variant: certificate.FileCertificate{
				Base:            certificate.Base{Key: base.Key, Name: base.Name},
				CertificatePath: "/c.pem",
				Rack:            coreRack,
			},
		}))
	})

	Describe("NewRetrieve", func() {
		It("Should search certificates by name", func(ctx SpecContext) {
			e := newEnv(ctx, telem.Hour)
			key := newKey()
			e.create(
				ctx,
				stored("Plant server", newCertificate(key, "a", 24*time.Hour)),
				nil,
			)
			e.create(
				ctx,
				stored("Broker CA", newCertificate(key, "b", 24*time.Hour)),
				nil,
			)
			var found []certificate.Certificate
			Eventually(func(g Gomega) {
				g.Expect(e.svc.NewRetrieve().
					Search("broker").
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				g.Expect(found).To(HaveLen(1))
			}).Should(Succeed())
			Expect(found[0].Base().Name).To(Equal("Broker CA"))
		})
		It("Should match certificates by source and by the rack of a file", func(
			ctx SpecContext,
		) {
			e := newEnv(ctx, telem.Hour)
			s := e.create(
				ctx,
				stored("Plant", newCertificate(newKey(), "a", time.Hour)),
				nil,
			)
			core := e.create(ctx, file("Core", coreRack), nil)
			other := e.create(ctx, file("Other", otherRack), nil)
			keys := func(w certificate.Where) []certificate.Key {
				GinkgoHelper()
				var found []certificate.Certificate
				Expect(e.svc.NewRetrieve().
					Where(MustSucceed(certificate.MatchWhere(w))).
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				keys := make([]certificate.Key, len(found))
				for i, c := range found {
					keys[i] = c.Base().Key
				}
				return keys
			}
			src := secret.SourceStored
			Expect(
				keys(certificate.Where{Source: &filter.Set[secret.Source]{Eq: &src}}),
			).
				To(Equal([]certificate.Key{s.Base().Key}))
			Expect(keys(certificate.Where{Racks: []rack.Key{otherRack}})).
				To(Equal([]certificate.Key{other.Base().Key}))
			Expect(keys(certificate.Where{Racks: []rack.Key{coreRack, otherRack}})).
				To(ConsistOf(core.Base().Key, other.Base().Key))
		})
	})

	Describe("Resources", func() {
		It("Should name each certificate by its name", func(ctx SpecContext) {
			e := newEnv(ctx, telem.Hour)
			c := e.create(
				ctx,
				stored("Plant", newCertificate(newKey(), "a", time.Hour)),
				nil,
			)
			Expect(
				e.svc.Resources().Retrieve(ctx, e.db, []resource.ID{c.ResourceID()}),
			).
				To(Equal([]resource.Resource{{ID: c.ResourceID(), Name: "Plant"}}))
		})
		It("Should list the certificate as a use of its private key",
			func(ctx SpecContext) {
				e := newEnv(ctx, telem.Hour)
				key := newKey()
				c := e.create(
					ctx,
					stored("Plant", newCertificate(key, "a", time.Hour)),
					encodeKey(key),
				)
				secretKey := *storedOf(c).PrivateKey
				Expect(e.secrets.Uses(ctx, e.db, secretKey)).
					To(Equal([]resource.ID{c.ResourceID()}))
			},
		)
		It("Should list no uses for a certificate that only its status reports on",
			func(ctx SpecContext) {
				e := newEnv(ctx, telem.Hour)
				c := e.create(
					ctx,
					stored("Plant", newCertificate(newKey(), "a", time.Hour)),
					nil,
				)
				Expect(
					e.expiry(ctx, c.Base().Key).Owner,
				).To(HaveValue(Equal(c.ResourceID())))
				Expect(e.svc.Uses(ctx, e.db, c.Base().Key)).To(BeEmpty())
			},
		)
		It("Should convert between keys and resource IDs", func() {
			c := certificate.Certificate{Variant: certificate.FileCertificate{
				Base: certificate.Base{Key: uuid.New()},
			}}
			key := c.Base().Key
			Expect(certificate.ResourceID(key)).To(Equal(c.ResourceID()))
			Expect(certificate.ResourceIDs([]certificate.Key{key})).
				To(Equal([]resource.ID{c.ResourceID()}))
			Expect(
				certificate.ResourceIDsFromCertificates([]certificate.Certificate{c}),
			).
				To(Equal([]resource.ID{c.ResourceID()}))
		})
	})

	Describe("Changes", func() {
		It(
			"Should return the create, rename, and delete that a commit makes",
			func(ctx SpecContext) {
				e := newEnv(ctx, telem.Hour)
				pemText := newCertificate(newKey(), "a", time.Hour)
				renamed := e.create(ctx, stored("Before", pemText), nil)
				deleted := e.create(ctx, stored("Deleted", pemText), nil)
				var changes []gorp.EntryChange[certificate.Key, certificate.Certificate]
				DeferCleanup(
					e.db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
						for ch := range e.svc.Changes(c) {
							changes = append(changes, ch)
						}
					}),
				)
				created := stored("Created", pemText)
				Expect(e.write(ctx, func(w certificate.Writer) error {
					if err := w.Set(ctx, &created, nil); err != nil {
						return err
					}
					if err := w.Rename(ctx, renamed.Base().Key, "After"); err != nil {
						return err
					}
					return w.Delete(ctx, deleted.Base().Key)
				})).To(Succeed())
				kinds := make(map[certificate.Key]gorp.ChangeKind, len(changes))
				names := make(map[certificate.Key]string, len(changes))
				for _, ch := range changes {
					kinds[ch.Key], names[ch.Key] = ch.Kind, ch.Value.Base().Name
				}
				Expect(kinds).To(Equal(map[certificate.Key]gorp.ChangeKind{
					created.Base().Key: gorp.ChangeCreate,
					renamed.Base().Key: gorp.ChangeUpdate,
					deleted.Base().Key: gorp.ChangeDelete,
				}))
				Expect(names[created.Base().Key]).To(Equal("Created"))
				Expect(names[renamed.Base().Key]).To(Equal("After"))
			},
		)
	})

	Describe("Expiry", func() {
		DescribeTable("Should write the expiry status when a certificate is created",
			func(
				ctx SpecContext,
				lifetime time.Duration,
				variant status.Variant,
				message string,
				description string,
			) {
				e := newEnv(ctx, telem.Hour)
				c := e.create(
					ctx,
					stored("Plant", newCertificate(newKey(), "a", lifetime)),
					nil,
				)
				s := e.expiry(ctx, c.Base().Key)
				Expect(s.Name).To(Equal("Plant"))
				Expect(s.Variant).To(Equal(variant))
				Expect(s.Message).To(Equal(message))
				Expect(s.Description).To(Equal(description))
			},
			Entry(
				"valid",
				365*24*time.Hour,
				status.VariantSuccess,
				"Certificate valid",
				"The certificate expires on November 3, 2027.",
			),
			Entry(
				"expiring within 30 days",
				10*24*time.Hour,
				status.VariantWarning,
				"Certificate expires soon",
				"The certificate expires on November 13, 2026. Replace it before then.",
			),
			Entry(
				"expired",
				-24*time.Hour,
				status.VariantError,
				"Certificate expired",
				"The certificate expired on November 2, 2026. "+
					"Replace it with a new certificate.",
			),
		)
		It("Should report a certificate that is not yet valid", func(ctx SpecContext) {
			e := newEnv(ctx, telem.Hour)
			c := e.create(ctx, stored("Plant", newCertificateBetween(
				newKey(),
				"a",
				start.Add(7*24*time.Hour),
				start.Add(365*24*time.Hour),
			)), nil)
			s := e.expiry(ctx, c.Base().Key)
			Expect(s.Variant).To(Equal(status.VariantWarning))
			Expect(s.Message).To(Equal("Certificate not yet valid"))
			Expect(s.Description).To(Equal(
				"The certificate becomes valid on November 10, 2026.",
			))
		})
		It("Should keep a certificate with no expiry valid", func(ctx SpecContext) {
			e := newEnv(ctx, telem.Hour)
			c := e.create(ctx, stored("Plant", newCertificateBetween(
				newKey(),
				"a",
				start.Add(-time.Hour),
				time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC),
			)), nil)
			details := MustBeOk(c.Details())
			Expect(details.NotAfter).To(Equal(telem.TimeStampMax))
			s := e.expiry(ctx, c.Base().Key)
			Expect(s.Variant).To(Equal(status.VariantSuccess))
			Expect(s.Message).To(Equal("Certificate valid"))
			Expect(s.Description).To(Equal("The certificate does not expire."))
		})
		It(
			"Should update the status on each check after it changes",
			func(ctx SpecContext) {
				e := newEnv(ctx, 5*telem.Millisecond)
				c := e.create(
					ctx,
					stored("Plant", newCertificate(newKey(), "a", 40*24*time.Hour)),
					nil,
				)
				written := e.expiry(ctx, c.Base().Key).Time
				e.advance(telem.Day)
				Consistently(func() telem.TimeStamp {
					return e.expiry(ctx, c.Base().Key).Time
				}).WithTimeout(50 * time.Millisecond).Should(Equal(written))
				e.advance(20 * telem.Day)
				Eventually(func() status.Variant {
					return e.expiry(ctx, c.Base().Key).Variant
				}).Should(Equal(status.VariantWarning))
				e.advance(20 * telem.Day)
				Eventually(func() status.Variant {
					return e.expiry(ctx, c.Base().Key).Variant
				}).Should(Equal(status.VariantError))
			},
		)
		It("Should write no status for a file certificate without details",
			func(ctx SpecContext) {
				e := newEnv(ctx, telem.Hour)
				c := e.create(ctx, certificate.Certificate{
					Variant: certificate.FileCertificate{
						Base:            certificate.Base{Name: "Plant"},
						CertificatePath: "/etc/plant/cert.pem",
						Rack:            coreRack,
					},
				}, nil)
				var found []status.Status[any]
				Expect(e.status.NewRetrieve[any]().
					Where(status.MatchOwners[any](c.ResourceID())).
					Entries(&found).
					Exec(ctx, e.db)).To(Succeed())
				Expect(found).To(BeEmpty())
			},
		)
		It("Should report the expiry of a file certificate with details",
			func(ctx SpecContext) {
				e := newEnv(ctx, telem.Hour)
				c := e.create(ctx, certificate.Certificate{
					Variant: certificate.FileCertificate{
						Base:            certificate.Base{Name: "Plant"},
						CertificatePath: "/etc/plant/cert.pem",
						Rack:            coreRack,
						Details: &certificate.Details{
							NotAfter: telem.NewTimeStamp(start.Add(-time.Hour)),
						},
					},
				}, nil)
				Expect(
					e.expiry(ctx, c.Base().Key).Variant,
				).To(Equal(status.VariantError))
			},
		)
	})
})
