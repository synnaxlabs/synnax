// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	migrate := func(ctx context.Context, old v0.Device) v1.Device {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Device]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Device]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "ni_device",
			Migrations: []gorp.Migration{v1.Migration},
		})).To(Succeed())
		var got v1.Device
		Expect(gorp.NewRetrieve[v1.Key, v1.Device]().
			Where(gorp.MatchKeys[v1.Key, v1.Device](old.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should keep the serial of a physical device", func(ctx SpecContext) {
		old := v0.Device{
			Key:        uuid.New(),
			Rack:       1,
			Name:       "PXI-6255",
			Serial:     "01A",
			Model:      "PXI-6255",
			Alias:      "Mod1",
			Identifier: "dev1",
			AnalogInput: v0.InputPorts{
				Index:    1,
				Channels: map[string]channel.Key{"0": 2},
			},
		}
		got := migrate(ctx, old)
		Expect(got.Serial).To(Equal("01A"))
		Expect(got.ResourceName).To(BeEmpty())
		Expect(got.Name).To(Equal("PXI-6255"))
		Expect(got.Model).To(Equal("PXI-6255"))
		Expect(got.Identifier).To(Equal("dev1"))
		Expect(got.AnalogInput).To(Equal(old.AnalogInput))
	})

	It("Should move the serial of a simulated device to its resource name", func(
		ctx SpecContext,
	) {
		got := migrate(ctx, v0.Device{
			Key:       uuid.New(),
			Rack:      1,
			Name:      "Simulated",
			Serial:    "7A1F0C2E",
			Model:     "PXI-6255",
			Simulated: true,
		})
		Expect(got.Serial).To(BeEmpty())
		Expect(got.ResourceName).To(Equal("7A1F0C2E"))
	})
})
