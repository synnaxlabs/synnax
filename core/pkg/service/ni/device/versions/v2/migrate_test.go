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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/device/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep each field and leave the hardware absent", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Device]{DB: db}))
		old := v1.Device{
			Key:          uuid.New(),
			Rack:         1,
			Name:         "NI 9205",
			Serial:       "01A",
			ResourceName: "{GUID}",
			Model:        "9205",
			Identifier:   "dev1",
			AnalogInput: v1.AnalogInputPorts{
				Index:    1,
				Channels: map[uint16]channel.Key{0: 2},
			},
		}
		Expect(gorp.NewCreate[v1.Key, v1.Device]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "ni_device",
			Migrations: []gorp.Migration{v2.Migration},
		})).To(Succeed())
		var got v2.Device
		Expect(gorp.NewRetrieve[v2.Key, v2.Device]().
			Where(gorp.MatchKeys[v2.Key, v2.Device](old.Key)).
			Entry(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got.Hardware).To(BeNil())
		Expect(got.Name).To(Equal("NI 9205"))
		Expect(got.ResourceName).To(Equal("{GUID}"))
		Expect(got.AnalogInput.Channels).To(Equal(map[uint16]channel.Key{0: 2}))
	})
})
