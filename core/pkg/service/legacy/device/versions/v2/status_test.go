// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	v2 "github.com/synnaxlabs/synnax/pkg/service/legacy/device/versions/v2"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("UnknownStatusMigration", func() {
	DescribeTable("Should migrate the variant of a device status",
		func(
			ctx SpecContext,
			message string,
			variant status.Variant,
			connected bool,
			want status.Variant,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
			statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			d := v2.Device{Key: "pump", Name: "Pump", Make: "http", Model: "server"}
			Expect(gorp.NewCreate[v2.Key, v2.Device]().Entry(&d).Exec(ctx, db)).
				To(Succeed())
			stored := v2.Status{
				Key:     "pump-status",
				Owner:   new(d.ResourceID()),
				Name:    d.Name,
				Time:    telem.Now(),
				Variant: variant,
				Message: message,
				Details: v2.StatusDetails{Connected: connected},
			}
			// The device table is not open during its own migration, so the status
			// skips reference checks.
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statuses.NewMigrationWriter(tx).Set(ctx, stored.ToAny())
			})).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "Device",
				Migrations: []gorp.Migration{v2.NewUnknownStatusMigration(statuses)},
			})).To(Succeed())
			var got v2.Status
			Expect(statuses.NewRetrieve[v2.StatusDetails]().
				Where(status.MatchOwners[v2.StatusDetails](d.ResourceID())).
				Entry(&got).
				Exec(ctx, db)).To(Succeed())
			Expect(got.Variant).To(Equal(want))
			Expect(got.Message).To(Equal(message))
		},
		Entry(
			"of the default",
			"Device status unknown",
			status.VariantWarning,
			false,
			status.VariantDisabled,
		),
		Entry(
			"of an older default that names the device",
			"Pump state unknown",
			status.VariantWarning,
			false,
			status.VariantDisabled,
		),
		Entry(
			"of a connected device with the default message",
			"Device status unknown",
			status.VariantWarning,
			true,
			status.VariantWarning,
		),
		Entry(
			"of a device the Driver reported disconnected",
			"Device disconnected",
			status.VariantWarning,
			false,
			status.VariantWarning,
		),
		Entry(
			"of an error with the default message",
			"Device status unknown",
			status.VariantError,
			false,
			status.VariantError,
		),
	)
})
