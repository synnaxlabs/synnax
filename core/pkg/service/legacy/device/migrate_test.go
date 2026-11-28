// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// markReleased marks each device migration before the HTTP blank header migration as
// applied, as a v0.62.0 Core leaves them.
func markReleased(ctx SpecContext, db *gorp.DB) {
	GinkgoHelper()
	var applied []gorp.Migration
	for _, m := range versions.NewMigrations(versions.MigrationsConfig{}) {
		if m.Key() == v2.HTTPBlankHeaderMigration.Key() {
			break
		}
		applied = append(applied, migrate.New(
			m.Key(),
			func(context.Context, gorp.Tx, alamos.Instrumentation) error { return nil },
		))
	}
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Device",
		Migrations: applied,
	})).To(Succeed())
}

// openDevices opens the device service and the services that it needs, which runs the
// device migrations.
func openDevices(ctx SpecContext, db *gorp.DB, stat *status.Service) *device.Service {
	GinkgoHelper()
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:      db,
		HostKey: 1,
		Status:  stat,
		DriverRole: MustSucceed(
			racktestutil.CreateDriverRole(ctx, db, roleSvc),
		),
	}))
	return MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:     db,
		Status: stat,
		Rack:   rackSvc,
	}))
}

var _ = Describe("Migrations", func() {
	It("Should drop each blank health check header of an HTTP server", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		markReleased(ctx, db)
		StartRun(db.Run)
		server := device.Device{
			Key:      "server",
			Location: "localhost:8080",
			Name:     "Weather station",
			Make:     "http",
			Model:    "HTTP server",
			Properties: msgpack.EncodedJSON{"health_check": map[string]any{
				"method": "GET",
				"path":   "/health",
				"headers": []any{
					map[string]any{"name": "X-Key", "value": ""},
					map[string]any{"name": "", "value": "abc"},
					map[string]any{"name": "X-Trace", "value": "on"},
				},
			}},
		}
		Expect(gorp.NewCreate[device.Key, device.Device]().
			Entry(&server).Exec(ctx, db)).To(Succeed())
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		svc := openDevices(ctx, db, stat)
		var got device.Device
		Expect(svc.NewRetrieve().
			Where(device.MatchKeys(server.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got.Properties["health_check"]).To(HaveKeyWithValue(
			"headers",
			[]any{map[string]any{"name": "X-Trace", "value": "on"}},
		))
	})

	It("Should give the default status of a device the disabled variant", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		markReleased(ctx, db)
		StartRun(db.Run)
		server := device.Device{
			Key:      "server",
			Location: "localhost:8080",
			Name:     "Weather station",
			Make:     "http",
			Model:    "HTTP server",
		}
		Expect(gorp.NewCreate[device.Key, device.Device]().
			Entry(&server).Exec(ctx, db)).To(Succeed())
		MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		stored := device.Status{
			Key:     "server-status",
			Owner:   new(server.ResourceID()),
			Name:    server.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Device status unknown",
		}
		// The device table is not open yet, so the status skips reference checks.
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return stat.NewMigrationWriter(tx).Set(ctx, stored.ToAny())
		})).To(Succeed())
		openDevices(ctx, db, stat)
		var got device.Status
		Expect(stat.NewRetrieve[device.StatusDetails]().
			Where(status.MatchOwners[device.StatusDetails](server.ResourceID())).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got.Variant).To(Equal(status.VariantDisabled))
	})
})
