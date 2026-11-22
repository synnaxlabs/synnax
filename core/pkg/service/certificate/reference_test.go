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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// device is an entry that names one certificate.
type device struct {
	Key         string          `json:"key"         msgpack:"key"`
	Certificate certificate.Key `json:"certificate" msgpack:"certificate"`
}

func (d device) GorpKey() string { return d.Key }

func (d device) EncodeOrc(w *orc.Writer) error {
	w.String(d.Key)
	w.Write(d.Certificate[:])
	return nil
}

func (d *device) DecodeOrc(r *orc.Reader) (err error) {
	if d.Key, err = r.String(); err != nil {
		return err
	}
	_, err = r.Read(d.Certificate[:])
	return err
}

var _ = Describe("Reference", func() {
	var (
		e       *env
		devices *gorp.Table[string, device]
		pemText string
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx, 0)
		pemText = newCertificate(newKey(), "plant", 24*time.Hour)
		// Each device runs on the Core's rack.
		devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, device]{
			DB:   e.db,
			Name: "device",
			References: []gorp.Reference[string, device]{
				certificate.NewReference(
					e.svc,
					certificate.ReferenceConfig[string, device]{
						Name: "certificate",
						Targets: func(d *device) []certificate.Key {
							return []certificate.Key{d.Certificate}
						},
						Rack: func(*device) rack.Key { return coreRack },
					},
				),
			},
		}))
	})
	create := func(ctx SpecContext, c certificate.Certificate) error {
		return devices.NewCreate().
			Entry(&device{Key: "d1", Certificate: c.Base().Key}).
			Exec(ctx, e.db)
	}
	d1 := resource.ID{Type: "device", Key: "d1"}

	Describe("NewReference", func() {
		It("Should allow a stored certificate", func(ctx SpecContext) {
			Expect(create(ctx, e.create(ctx, stored("Server", pemText), nil))).
				To(Succeed())
		})
		It(
			"Should allow a file certificate of the entry's rack",
			func(ctx SpecContext) {
				Expect(create(ctx, e.create(ctx, file("Client", coreRack), nil))).
					To(Succeed())
			},
		)
		It("Should refuse a file certificate of another rack", func(ctx SpecContext) {
			err := create(ctx, e.create(ctx, file("Client", otherRack), nil))
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(
				`certificate: certificate "Client" is a file of rack 2, not of rack 1: ` +
					`validation error`,
			))
		})
		It("Should keep a certificate from being deleted while an entry uses it",
			func(ctx SpecContext) {
				c := e.create(ctx, stored("Server", pemText), nil)
				Expect(create(ctx, c)).To(Succeed())
				Expect(e.write(ctx, func(w certificate.Writer) error {
					return w.Delete(ctx, c.Base().Key)
				})).To(MatchError(ContainSubstring("because device d1 references it")))
			},
		)
	})

	Describe("Uses", func() {
		It("Should return no device for a certificate that nothing references",
			func(ctx SpecContext) {
				c := e.create(ctx, stored("Server", pemText), nil)
				Expect(e.svc.Uses(ctx, e.db, c.Base().Key)).ToNot(ContainElement(d1))
			},
		)
		It("Should return each record that references the certificate",
			func(ctx SpecContext) {
				c := e.create(ctx, stored("Server", pemText), nil)
				Expect(create(ctx, c)).To(Succeed())
				Expect(e.svc.Uses(ctx, e.db, c.Base().Key)).To(ContainElement(d1))
			},
		)
	})

	Describe("HasPrivateKey", func() {
		hasPrivateKey := func(ctx SpecContext, c certificate.Certificate) bool {
			GinkgoHelper()
			return MustSucceed(e.svc.HasPrivateKey(ctx, e.db, c.Base().Key))
		}
		It("Should report a stored certificate with a private key", func(
			ctx SpecContext,
		) {
			key := newKey()
			pemText := newCertificate(key, "client", 24*time.Hour)
			c := e.create(ctx, stored("Client", pemText), encodeSEC1Key(key))
			Expect(hasPrivateKey(ctx, c)).To(BeTrue())
		})
		It("Should report a stored certificate without a private key", func(
			ctx SpecContext,
		) {
			c := e.create(ctx, stored("Server", pemText), nil)
			Expect(hasPrivateKey(ctx, c)).To(BeFalse())
		})
		It("Should report a file certificate with a private key file", func(
			ctx SpecContext,
		) {
			c := file("Client", coreRack)
			f, ok := c.Variant.(certificate.FileCertificate)
			Expect(ok).To(BeTrue())
			f.PrivateKeyFile = new("client.pem")
			c.Variant = f
			Expect(hasPrivateKey(ctx, e.create(ctx, c, nil))).To(BeTrue())
		})
		It("Should report a file certificate without a private key file", func(
			ctx SpecContext,
		) {
			Expect(hasPrivateKey(ctx, e.create(ctx, file("Client", coreRack), nil))).
				To(BeFalse())
		})
		It("Should return query.ErrNotFound for a certificate that does not exist",
			func(ctx SpecContext) {
				Expect(e.svc.HasPrivateKey(ctx, e.db, uuid.New())).
					Error().To(MatchError(query.ErrNotFound))
			},
		)
	})
})
