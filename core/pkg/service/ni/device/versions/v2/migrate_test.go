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
	It("Should keep each field, leave unset channels and the hardware absent", func(
		ctx SpecContext,
	) {
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
			CounterInput: map[uint16]v1.CounterInputChannels{1: {Channel: 3}},
			AnalogOutput: map[uint16]v1.AnalogOutputChannels{
				0: {StateIndex: 4, Command: 5},
			},
			DigitalInput: map[uint16]v1.DigitalInputChannels{
				0: {Lines: map[uint8]channel.Key{1: 6}},
			},
			DigitalOutput: map[uint16]v1.DigitalOutputChannels{
				0: {
					StateIndex: 7,
					Lines: map[uint8]v1.CommandStatePair{
						2: {Command: 8},
						3: {},
					},
				},
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
		Expect(got.AnalogInput.Index).To(Equal(new(channel.Key(1))))
		Expect(got.AnalogInput.Temperature).To(BeNil())
		Expect(got.CounterInput[1].Index).To(BeNil())
		Expect(got.CounterInput[1].Channel).To(Equal(new(channel.Key(3))))
		Expect(got.AnalogOutput[0]).To(Equal(v2.AnalogOutputChannels{
			StateIndex: new(channel.Key(4)),
			Command:    new(channel.Key(5)),
		}))
		Expect(got.DigitalInput[0].Index).To(BeNil())
		Expect(got.DigitalInput[0].Lines).To(Equal(map[uint8]channel.Key{1: 6}))
		Expect(got.DigitalOutput[0].StateIndex).To(Equal(new(channel.Key(7))))
		Expect(got.DigitalOutput[0].Lines).To(Equal(map[uint8]v2.CommandStatePair{
			2: {Command: new(channel.Key(8))},
			3: {},
		}))
	})
})
