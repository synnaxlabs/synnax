// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package opcua_test

import (
	"fmt"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	certtestutil "github.com/synnaxlabs/synnax/pkg/service/certificate/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Certificate", func() {
	var e *env
	BeforeEach(func(ctx SpecContext) { e = openEnv(ctx) })
	retrieve := func(ctx SpecContext, value any) certificate.Certificate {
		GinkgoHelper()
		s, ok := value.(string)
		Expect(ok).To(BeTrue())
		var c certificate.Certificate
		Expect(e.certs.NewRetrieve().
			Where(certificate.MatchKeys(MustSucceed(uuid.Parse(s)))).
			Entry(&c).
			Exec(ctx, e.db)).To(Succeed())
		return c
	}
	set := func(ctx SpecContext, c *certificate.Certificate) {
		GinkgoHelper()
		Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
			return e.certs.NewWriter(tx).Set(ctx, c, nil)
		})).To(Succeed())
	}
	// devicesUsing returns the devices that use the certificate with key.
	devicesUsing := func(ctx SpecContext, key certificate.Key) []resource.ID {
		GinkgoHelper()
		uses := MustSucceed(e.certs.Uses(ctx, e.db, key))
		return lo.Filter(uses, func(id resource.ID, _ int) bool {
			return id.Type == resource.TypeDevice
		})
	}
	migrate := func(ctx SpecContext) error {
		return e.run(ctx, opcua.CertificateMigration(e.certs, e.statuses))
	}

	Describe("CertificateMigration", func() {
		It("Should move a pinned server certificate into a stored certificate",
			func(ctx SpecContext) {
				pem, _ := certtestutil.NewPair()
				dev := e.create(ctx, opcMake, map[string]any{
					"endpoint":           "opc.tcp://localhost:4840",
					"server_certificate": pem,
				})
				Expect(migrate(ctx)).To(Succeed())
				conn := e.connection(ctx, dev.Key)
				Expect(
					conn,
				).To(HaveKeyWithValue("endpoint", "opc.tcp://localhost:4840"))
				c := retrieve(ctx, conn["server_certificate"])
				Expect(c.Base().Name).To(Equal("Pump PLC server certificate"))
				stored, ok := c.Variant.(certificate.StoredCertificate)
				Expect(ok).To(BeTrue())
				Expect(stored.Pem).To(Equal(pem))
				Expect(stored.PrivateKey).To(BeNil())
				Expect(devicesUsing(ctx, c.Base().Key)).
					To(HaveExactElements(dev.ResourceID()))
			},
		)

		It("Should clear a server certificate that is not PEM", func(ctx SpecContext) {
			dev := e.create(ctx, opcMake, map[string]any{
				"server_certificate": "/pki/trusted/server.der",
			})
			Expect(migrate(ctx)).To(Succeed())
			Expect(e.connection(ctx, dev.Key)).
				To(HaveKeyWithValue("server_certificate", ""))
			Expect(e.certs.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})

		It("Should clear a server certificate that does not parse", func(
			ctx SpecContext,
		) {
			dev := e.create(ctx, opcMake, map[string]any{
				"server_certificate": "-----BEGIN CERTIFICATE-----\nAAAA\n" +
					"-----END CERTIFICATE-----\n",
			})
			Expect(migrate(ctx)).To(Succeed())
			Expect(e.connection(ctx, dev.Key)).
				To(HaveKeyWithValue("server_certificate", ""))
			Expect(e.certs.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})

		It("Should move a client certificate and key path into a file certificate",
			func(ctx SpecContext) {
				dev := e.create(ctx, opcMake, map[string]any{
					"client_certificate": "/pki/own/certs/client.der",
					"client_private_key": "/pki/own/private/private.pem",
				})
				Expect(migrate(ctx)).To(Succeed())
				conn := e.connection(ctx, dev.Key)
				Expect(conn).ToNot(HaveKey("client_private_key"))
				c := retrieve(ctx, conn["client_certificate"])
				key := c.Base().Key
				Expect(c.Base().Name).To(Equal("Pump PLC client certificate"))
				Expect(c.Variant).To(Equal(certificate.FileCertificate{
					Base:            c.Base(),
					CertificatePath: "/pki/own/certs/client.der",
					PrivateKeyFile:  new(key.String() + ".pem"),
					Rack:            e.racks.EmbeddedKey,
				}))
				var st status.Status[any]
				Expect(e.statuses.NewRetrieve[any]().
					Where(status.MatchOwners[any](c.ResourceID())).
					Entry(&st).
					Exec(ctx, e.db)).To(Succeed())
				Expect(st.Variant).To(Equal(status.VariantWarning))
				Expect(st.Name).To(Equal("Pump PLC client certificate"))
				Expect(st.Message).To(Equal("Private key file renamed"))
				Expect(st.Description).To(Equal(
					"The Driver reads the private key of this certificate from its " +
						"secrets folder now. Copy /pki/own/private/private.pem into " +
						"the secrets folder of the Driver as " + key.String() + ".pem.",
				))
			},
		)

		It("Should give each device its own key file", func(ctx SpecContext) {
			conn := map[string]any{
				"client_certificate": "/pki/own/certs/client.der",
				"client_private_key": "/pki/own/private/private.pem",
			}
			a := e.create(ctx, opcMake, conn)
			b := e.create(ctx, opcMake, conn)
			Expect(migrate(ctx)).To(Succeed())
			keyFile := func(key string) string {
				file, ok := retrieve(
					ctx,
					e.connection(ctx, key)["client_certificate"],
				).Variant.(certificate.FileCertificate)
				Expect(ok).To(BeTrue())
				return *file.PrivateKeyFile
			}
			Expect(keyFile(a.Key)).ToNot(Equal(keyFile(b.Key)))
		})

		It("Should keep empty certificates empty", func(ctx SpecContext) {
			empty := e.create(ctx, opcMake, map[string]any{
				"client_certificate": "",
				"client_private_key": "",
				"server_certificate": "",
			})
			none := e.create(ctx, opcMake, map[string]any{"endpoint": "opc.tcp://a:1"})
			Expect(migrate(ctx)).To(Succeed())
			Expect(e.connection(ctx, empty.Key)).
				To(HaveKeyWithValue("client_certificate", ""))
			Expect(e.connection(ctx, none.Key)).ToNot(HaveKey("client_certificate"))
			Expect(e.certs.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})

		It("Should not change a device of another make", func(ctx SpecContext) {
			pem, _ := certtestutil.NewPair()
			dev := e.create(ctx, "modbus", map[string]any{"server_certificate": pem})
			Expect(migrate(ctx)).To(Succeed())
			Expect(e.connection(ctx, dev.Key)).
				To(HaveKeyWithValue("server_certificate", pem))
			Expect(e.certs.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})
	})

	Describe("CertificateReference", func() {
		It("Should list an OPC UA device as a use of its certificates", func(
			ctx SpecContext,
		) {
			pem, _ := certtestutil.NewPair()
			server := certificate.Certificate{Variant: certificate.StoredCertificate{
				Base: certificate.Base{Name: "Server"},
				Pem:  pem,
			}}
			set(ctx, &server)
			client := certificate.Certificate{Variant: certificate.FileCertificate{
				Base:            certificate.Base{Name: "Client"},
				CertificatePath: "/pki/client.der",
				Rack:            e.racks.EmbeddedKey,
			}}
			set(ctx, &client)
			dev := e.create(ctx, opcMake, map[string]any{
				"server_certificate": server.Base().Key.String(),
				"client_certificate": client.Base().Key.String(),
			})
			for _, c := range []certificate.Certificate{server, client} {
				Expect(devicesUsing(ctx, c.Base().Key)).
					To(HaveExactElements(dev.ResourceID()))
			}
		})

		It("Should refuse a file certificate of another rack", func(ctx SpecContext) {
			other := rack.Rack{Name: "Other"}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.racks.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			client := certificate.Certificate{Variant: certificate.FileCertificate{
				Base:            certificate.Base{Name: "Client"},
				CertificatePath: "/pki/client.der",
				Rack:            other.Key,
			}}
			set(ctx, &client)
			dev := device.Device{
				Key:      uuid.New().String(),
				Rack:     e.racks.EmbeddedKey,
				Location: "opc.tcp://localhost:4840",
				Make:     opcMake,
				Model:    "OPC UA",
				Name:     "Pump PLC",
				Properties: msgpack.EncodedJSON{"connection": map[string]any{
					"client_certificate": client.Base().Key.String(),
				}},
			}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.devices.NewWriter(tx).Create(ctx, &dev)
			})).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(fmt.Sprintf(
					"certificate \"Client\" is a file of rack %d, not of rack %d",
					other.Key,
					e.racks.EmbeddedKey,
				))),
			))
		})

		It("Should ignore a device of another make", func(ctx SpecContext) {
			client := certificate.Certificate{Variant: certificate.FileCertificate{
				Base:            certificate.Base{Name: "Client"},
				CertificatePath: "/pki/client.der",
				Rack:            e.racks.EmbeddedKey,
			}}
			set(ctx, &client)
			e.create(ctx, "modbus", map[string]any{
				"client_certificate": client.Base().Key.String(),
			})
			Expect(devicesUsing(ctx, client.Base().Key)).To(BeEmpty())
		})
	})
})
