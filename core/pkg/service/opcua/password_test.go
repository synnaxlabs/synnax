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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Password", func() {
	var e *env
	BeforeEach(func(ctx SpecContext) { e = openEnv(ctx) })
	createSecret := func(ctx SpecContext) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: "pw"}}}
		Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
			return e.secrets.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))
		})).To(Succeed())
		return s.Base().Key
	}
	migrate := func(ctx SpecContext) error {
		return e.run(ctx, opcua.PasswordMigration(e.secrets))
	}

	Describe("PasswordMigration", func() {
		It("Should move each password into a secret named after its device", func(
			ctx SpecContext,
		) {
			dev := e.create(ctx, opcMake, map[string]any{
				"endpoint": "opc.tcp://localhost:4840",
				"username": "operator",
				"password": "hunter2",
			})
			Expect(migrate(ctx)).To(Succeed())
			conn := e.connection(ctx, dev.Key)
			Expect(conn).To(HaveKeyWithValue("endpoint", "opc.tcp://localhost:4840"))
			Expect(conn).To(HaveKeyWithValue("username", "operator"))
			password, ok := conn["password"].(string)
			Expect(ok).To(BeTrue())
			key := MustSucceed(uuid.Parse(password))
			var s secret.Secret
			Expect(e.secrets.NewRetrieve().
				Where(secret.MatchKeys(key)).
				Entry(&s).
				Exec(ctx, e.db)).To(Succeed())
			Expect(s.Base().Name).To(Equal("Pump PLC password"))
			Expect(e.secrets.Reveal(ctx, e.db, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
			Expect(
				e.secrets.Uses(ctx, e.db, key),
			).To(HaveExactElements(dev.ResourceID()))
		})

		It("Should keep an empty password empty", func(ctx SpecContext) {
			empty := e.create(ctx, opcMake, map[string]any{"password": ""})
			none := e.create(ctx, opcMake, map[string]any{"endpoint": "opc.tcp://a:1"})
			Expect(migrate(ctx)).To(Succeed())
			Expect(e.connection(ctx, empty.Key)).To(HaveKeyWithValue("password", ""))
			Expect(e.connection(ctx, none.Key)).ToNot(HaveKey("password"))
			Expect(e.secrets.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})

		It("Should not change a device of another make", func(ctx SpecContext) {
			dev := e.create(ctx, "modbus", map[string]any{"password": "hunter2"})
			Expect(migrate(ctx)).To(Succeed())
			Expect(
				e.connection(ctx, dev.Key),
			).To(HaveKeyWithValue("password", "hunter2"))
			Expect(e.secrets.NewRetrieve().Count(ctx, e.db)).To(Equal(0))
		})

		It("Should fail for a connection that is not an object", func(ctx SpecContext) {
			dev := device.Device{
				Key:      uuid.New().String(),
				Rack:     e.racks.EmbeddedKey,
				Location: "opc.tcp://localhost:4840",
				Make:     opcMake,
				Model:    "OPC UA",
				Name:     "Broken",
				Properties: msgpack.EncodedJSON{
					"connection": "opc.tcp://localhost:4840",
				},
			}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.devices.NewWriter(tx).Create(ctx, &dev)
			})).To(Succeed())
			Expect(migrate(ctx)).To(MatchError(
				"connection of device " + dev.Key + " is a string, not an object",
			))
		})
	})

	Describe("PasswordReference", func() {
		It("Should list an OPC UA device as a use of its password", func(
			ctx SpecContext,
		) {
			key := createSecret(ctx)
			dev := e.create(ctx, opcMake, map[string]any{"password": key.String()})
			Expect(
				e.secrets.Uses(ctx, e.db, key),
			).To(HaveExactElements(dev.ResourceID()))
		})

		It("Should ignore a device of another make", func(ctx SpecContext) {
			key := createSecret(ctx)
			e.create(ctx, "modbus", map[string]any{"password": key.String()})
			Expect(e.secrets.Uses(ctx, e.db, key)).To(BeEmpty())
		})

		It("Should keep the password from being deleted while the device exists", func(
			ctx SpecContext,
		) {
			key := createSecret(ctx)
			dev := e.create(ctx, opcMake, map[string]any{"password": key.String()})
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.secrets.NewWriter(tx).Delete(ctx, key)
			})).To(MatchError(ContainSubstring(
				"because device " + dev.Key + " references it",
			)))
		})

		It("Should refuse a file secret of another rack", func(ctx SpecContext) {
			other := rack.Rack{Name: "Other Driver"}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.racks.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Pump password"},
				FileName: "pump.txt",
				Rack:     other.Key,
			}}
			Expect(e.db.WithTx(ctx, func(tx gorp.Tx) error {
				return e.secrets.NewWriter(tx).Set(ctx, &s, nil)
			})).To(Succeed())
			_, err := e.write(
				ctx,
				opcMake,
				map[string]any{"password": s.Base().Key.String()},
			)
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(
				`properties.connection.password: secret "Pump password" is a file of`,
			)))
		})
	})
})
