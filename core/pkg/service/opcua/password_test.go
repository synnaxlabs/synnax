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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/opcua"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

// opcMake is the make of an OPC UA device.
const opcMake = "opc"

var _ = Describe("Password", func() {
	var (
		db      *gorp.DB
		racks   *rack.Service
		secrets *secret.Service
		devices *device.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		racks = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     statuses,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roles)),
			// No Driver reports for the rack, so a check would overwrite statuses.
			HealthCheckInterval: telem.Hour,
		}))
		secrets = secrettestutil.OpenService(ctx, db)
		devices = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:     db,
			Status: statuses,
			Rack:   racks,
			References: []gorp.Reference[device.Key, device.Device]{
				opcua.PasswordReference(secrets),
			},
		}))
	})

	// write creates a device of make mk with the given connection properties.
	write := func(ctx SpecContext, mk string, conn map[string]any) (device.Device, error) {
		dev := device.Device{
			Key:        uuid.New().String(),
			Rack:       racks.EmbeddedKey,
			Location:   "opc.tcp://localhost:4840",
			Make:       mk,
			Model:      "OPC UA",
			Name:       "Pump PLC",
			Properties: msgpack.EncodedJSON{"connection": conn},
		}
		return dev, db.WithTx(ctx, func(tx gorp.Tx) error {
			return devices.NewWriter(tx).Create(ctx, &dev)
		})
	}
	create := func(ctx SpecContext, mk string, conn map[string]any) device.Device {
		GinkgoHelper()
		return MustSucceed(write(ctx, mk, conn))
	}
	// connection returns the stored connection of the device with key.
	connection := func(ctx SpecContext, key device.Key) map[string]any {
		GinkgoHelper()
		var dev device.Device
		Expect(devices.NewRetrieve().
			Where(device.MatchKeys(key)).
			Entry(&dev).
			Exec(ctx, db)).To(Succeed())
		conn, ok := dev.Properties["connection"].(map[string]any)
		Expect(ok).To(BeTrue())
		return conn
	}
	createSecret := func(ctx SpecContext) secret.Key {
		GinkgoHelper()
		s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: "pw"}}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return secrets.NewWriter(tx).Set(ctx, &s, []byte("hunter2"))
		})).To(Succeed())
		return s.Base().Key
	}
	migrate := func(ctx SpecContext) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return opcua.PasswordMigration(secrets).
				Run(ctx, tx, alamos.Instrumentation{})
		})
	}

	Describe("PasswordMigration", func() {
		It("Should move each password into a secret named after its device", func(
			ctx SpecContext,
		) {
			dev := create(ctx, opcMake, map[string]any{
				"endpoint": "opc.tcp://localhost:4840",
				"username": "operator",
				"password": "hunter2",
			})
			Expect(migrate(ctx)).To(Succeed())
			conn := connection(ctx, dev.Key)
			Expect(conn).To(HaveKeyWithValue("endpoint", "opc.tcp://localhost:4840"))
			Expect(conn).To(HaveKeyWithValue("username", "operator"))
			password, ok := conn["password"].(string)
			Expect(ok).To(BeTrue())
			key := MustSucceed(uuid.Parse(password))
			var s secret.Secret
			Expect(secrets.NewRetrieve().
				Where(secret.MatchKeys(key)).
				Entry(&s).
				Exec(ctx, db)).To(Succeed())
			Expect(s.Base().Name).To(Equal("Pump PLC password"))
			Expect(secrets.Reveal(ctx, db, key)).
				To(Equal(map[secret.Key][]byte{key: []byte("hunter2")}))
			Expect(secrets.Uses(ctx, db, key)).To(HaveExactElements(dev.ResourceID()))
		})

		It("Should keep an empty password empty", func(ctx SpecContext) {
			empty := create(ctx, opcMake, map[string]any{"password": ""})
			none := create(ctx, opcMake, map[string]any{"endpoint": "opc.tcp://a:1"})
			Expect(migrate(ctx)).To(Succeed())
			Expect(connection(ctx, empty.Key)).To(HaveKeyWithValue("password", ""))
			Expect(connection(ctx, none.Key)).ToNot(HaveKey("password"))
			Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(0))
		})

		It("Should not change a device of another make", func(ctx SpecContext) {
			dev := create(ctx, "modbus", map[string]any{"password": "hunter2"})
			Expect(migrate(ctx)).To(Succeed())
			Expect(connection(ctx, dev.Key)).To(HaveKeyWithValue("password", "hunter2"))
			Expect(secrets.NewRetrieve().Count(ctx, db)).To(Equal(0))
		})

		It("Should fail for a connection that is not an object", func(ctx SpecContext) {
			dev := device.Device{
				Key:      uuid.New().String(),
				Rack:     racks.EmbeddedKey,
				Location: "opc.tcp://localhost:4840",
				Make:     opcMake,
				Model:    "OPC UA",
				Name:     "Broken",
				Properties: msgpack.EncodedJSON{
					"connection": "opc.tcp://localhost:4840",
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return devices.NewWriter(tx).Create(ctx, &dev)
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
			dev := create(ctx, opcMake, map[string]any{"password": key.String()})
			Expect(secrets.Uses(ctx, db, key)).To(HaveExactElements(dev.ResourceID()))
		})

		It("Should ignore a device of another make", func(ctx SpecContext) {
			key := createSecret(ctx)
			create(ctx, "modbus", map[string]any{"password": key.String()})
			Expect(secrets.Uses(ctx, db, key)).To(BeEmpty())
		})

		It("Should keep the password from being deleted while the device exists", func(
			ctx SpecContext,
		) {
			key := createSecret(ctx)
			dev := create(ctx, opcMake, map[string]any{"password": key.String()})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return secrets.NewWriter(tx).Delete(ctx, key)
			})).To(MatchError(ContainSubstring(
				"because device " + dev.Key + " references it",
			)))
		})

		It("Should refuse a file secret of another rack", func(ctx SpecContext) {
			other := rack.Rack{Name: "Other Driver"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return racks.NewWriter(tx).Create(ctx, &other)
			})).To(Succeed())
			s := secret.Secret{Variant: secret.FileSecret{
				Base:     secret.Base{Name: "Pump password"},
				FileName: "pump.txt",
				Rack:     other.Key,
			}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return secrets.NewWriter(tx).Set(ctx, &s, nil)
			})).To(Succeed())
			_, err := write(
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
