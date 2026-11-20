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
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
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
)

var (
	node testutil.Node
	db   *gorp.DB
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	db = node.DB
})

func TestOPCUA(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service OPC UA Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// opcMake is the make of an OPC UA device.
const opcMake = "opc"

// env holds the services that the specs of OPC UA devices write through, with the OPC
// UA references on the device table.
type env struct {
	db       *gorp.DB
	racks    *rack.Service
	statuses *status.Service
	secrets  *secret.Service
	certs    *certificate.Service
	devices  *device.Service
}

func openEnv(ctx SpecContext) *env {
	GinkgoHelper()
	e := &env{db: DeferClose(gorp.Wrap(memkv.New()))}
	StartRun(e.db.Run)
	e.statuses = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: e.db}))
	roles := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: e.db}))
	e.racks = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         e.db,
		HostKey:    1,
		Status:     e.statuses,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, e.db, roles)),
		// No Driver reports for the rack, so a check would overwrite statuses.
		HealthCheckInterval: telem.Hour,
	}))
	e.secrets = secrettestutil.OpenService(ctx, e.db)
	e.certs = MustOpen(certificate.OpenService(ctx, certificate.ServiceConfig{
		DB:      e.db,
		Secret:  e.secrets,
		Status:  e.statuses,
		Rack:    e.racks.EmbeddedKey,
		HostKey: 1,
	}))
	e.devices = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:     e.db,
		Status: e.statuses,
		Rack:   e.racks,
		References: []gorp.Reference[device.Key, device.Device]{
			opcua.PasswordReference(e.secrets),
			opcua.CertificateReference(e.certs),
		},
	}))
	return e
}

// create creates a device named "Pump PLC" of make mk with the given connection
// properties, on the embedded rack.
// write creates a device of make mk with the given connection properties.
func (e *env) write(
	ctx SpecContext,
	mk string,
	conn map[string]any,
) (device.Device, error) {
	dev := device.Device{
		Key:        uuid.New().String(),
		Rack:       e.racks.EmbeddedKey,
		Location:   "opc.tcp://localhost:4840",
		Make:       mk,
		Model:      "OPC UA",
		Name:       "Pump PLC",
		Properties: msgpack.EncodedJSON{"connection": conn},
	}
	return dev, e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return e.devices.NewWriter(tx).Create(ctx, &dev)
	})
}

func (e *env) create(ctx SpecContext, mk string, conn map[string]any) device.Device {
	GinkgoHelper()
	return MustSucceed(e.write(ctx, mk, conn))
}

// connection returns the stored connection of the device with key.
func (e *env) connection(ctx SpecContext, key device.Key) map[string]any {
	GinkgoHelper()
	var dev device.Device
	Expect(e.devices.NewRetrieve().
		Where(device.MatchKeys(key)).
		Entry(&dev).
		Exec(ctx, e.db)).To(Succeed())
	conn, ok := dev.Properties["connection"].(map[string]any)
	Expect(ok).To(BeTrue())
	return conn
}

// run runs m in one transaction.
func (e *env) run(ctx SpecContext, m gorp.Migration) error {
	return e.db.WithTx(ctx, func(tx gorp.Tx) error {
		return m.Run(ctx, tx, alamos.Instrumentation{})
	})
}
