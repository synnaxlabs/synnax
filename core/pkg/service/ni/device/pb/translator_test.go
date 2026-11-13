// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device"
	"github.com/synnaxlabs/synnax/pkg/service/ni/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Translator", func() {
	It("Should round trip a device with its port maps and status", func() {
		key := uuid.New()
		chassisKey := chassis.Key(uuid.New())
		d := device.Device{
			Key:          key,
			Rack:         rack.NewKey(1, 2),
			Name:         "Module",
			Serial:       "01A",
			ResourceName: "{GUID}",
			Model:        "NI 9205",
			Chassis:      &chassisKey,
			Identifier:   "mod1",
			AnalogInput: device.AnalogInputPorts{
				Index:       1,
				Temperature: 2,
				Channels:    map[uint16]channel.Key{3: 4},
			},
			AnalogOutput: map[uint16]device.AnalogOutputChannels{
				1: {StateIndex: 5, Command: 6, State: 7},
			},
			CounterInput: map[uint16]device.CounterInputChannels{
				2: {Index: 8, Channel: 9},
			},
			DigitalInput: map[uint16]device.DigitalInputChannels{
				0: {Index: 10, Lines: map[uint8]channel.Key{7: 11}},
			},
			DigitalOutput: map[uint16]device.DigitalOutputChannels{
				1: {
					StateIndex: 12,
					Lines: map[uint8]device.CommandStatePair{
						255: {Command: 13, State: 14},
					},
				},
			},
			Status: &device.Status{
				Key:     "status",
				Owner:   new(device.ResourceID(key)),
				Variant: status.VariantSuccess,
				Message: "Device present",
				Labels:  []label.Key{label.Key(uuid.New())},
				Details: device.StatusDetails{Rack: rack.NewKey(1, 2), Device: key},
			},
		}
		translated := MustSucceed(pb.DeviceToPB(d))
		Expect(pb.DeviceFromPB(translated)).To(Equal(d))
	})

	It("Should reject a line number that does not fit in a byte", func() {
		translated := MustSucceed(
			pb.DigitalInputChannelsToPB(device.DigitalInputChannels{}),
		)
		translated.Lines = map[uint32]uint32{256: 1}
		Expect(pb.DigitalInputChannelsFromPB(translated)).Error().
			To(MatchError(validate.ErrValidation))
	})

	It("Should reject a port number that does not fit in 16 bits", func() {
		translated := MustSucceed(pb.DeviceToPB(device.Device{Key: uuid.New()}))
		translated.CounterInput = map[uint32]*pb.CounterInputChannels{65536: {}}
		Expect(pb.DeviceFromPB(translated)).Error().
			To(MatchError(validate.ErrValidation))
	})

	DescribeTable("Should round trip an update", func(u device.Update) {
		translated := MustSucceed(pb.UpdateToPB(u))
		Expect(pb.UpdateFromPB(translated)).To(Equal(u))
	},
		Entry("of the identity group", device.Update{Variant: device.IdentityUpdate{
			Identity: device.Identity{
				Rack:         rack.NewKey(1, 2),
				Serial:       "01A",
				ResourceName: "{GUID}",
				Model:        "NI 9205",
			},
		}}),
		Entry("of the configuration group", device.Update{
			Variant: device.ConfigurationUpdate{Configuration: device.Configuration{
				Name:       "Module",
				Identifier: "mod1",
				AnalogInput: device.AnalogInputPorts{
					Channels: map[uint16]channel.Key{3: 4},
				},
				AnalogOutput:  map[uint16]device.AnalogOutputChannels{},
				CounterInput:  map[uint16]device.CounterInputChannels{},
				DigitalInput:  map[uint16]device.DigitalInputChannels{},
				DigitalOutput: map[uint16]device.DigitalOutputChannels{},
			}},
		}),
	)
})
