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
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Update", func() {
	identity := device.Update{Variant: device.IdentityUpdate{
		Identity: device.Identity{
			Rack:       1,
			Serial:     470010001,
			Connection: device.ConnectionWifi,
			Model:      device.ModelT7,
		},
	}}
	configuration := device.Update{Variant: device.ConfigurationUpdate{
		Configuration: device.Configuration{
			Name:            "T7",
			Identifier:      "lj",
			ReadIndex:       new(channel.Key(1)),
			WriteStateIndex: new(channel.Key(2)),
			Ports: device.Ports{Variant: device.T7Ports{
				Mux80Attached: true,
				AnalogInput: map[device.T7AnalogInputPort]channel.Key{
					device.T7AnalogInputPortAin48: 3,
				},
				AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{
					device.AnalogOutputPortDac0: {
						Command: 4,
						State:   new(channel.Key(5)),
					},
				},
				DigitalInput: map[device.T7DigitalPort]channel.Key{
					device.T7DigitalPortDio0: 6,
				},
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio1: {Command: 7, State: new(channel.Key(8))},
				},
			}},
		},
	}}

	DescribeTable("Should keep the group through msgpack",
		func(update device.Update) {
			var got device.Update
			Expect(msgpack.Unmarshal(
				MustSucceed(msgpack.Marshal(update)),
				&got,
			)).To(Succeed())
			Expect(got).To(Equal(update))
		},
		Entry("identity", identity),
		Entry("configuration", configuration),
	)

	It("Should match the JSON shape in msgpack", func() {
		var wire map[string]any
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(configuration)),
			&wire,
		)).To(Succeed())
		var fromJSON map[string]any
		Expect(json.Unmarshal(
			MustSucceed(json.Marshal(configuration)),
			&fromJSON,
		)).To(Succeed())
		Expect(wire["group"]).To(Equal(fromJSON["group"]))
		Expect(wire["ports"]).To(HaveKeyWithValue("model", "t7"))
		Expect(wire["ports"]).To(HaveKeyWithValue("mux80_attached", true))
		Expect(fromJSON["ports"]).To(HaveKeyWithValue("model", "t7"))
	})

	It("Should encode an empty update as nil", func() {
		var got device.Update
		Expect(msgpack.Unmarshal(
			MustSucceed(msgpack.Marshal(device.Update{})),
			&got,
		)).To(Succeed())
		Expect(got.Variant).To(BeNil())
	})

	DescribeTable("Should reject an unknown discriminator in msgpack",
		func(raw map[string]any) {
			var got device.Update
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(raw)), &got)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("group", map[string]any{"group": "other"}),
		Entry("model", map[string]any{
			"group": "configuration",
			"ports": map[string]any{"model": "t9"},
		}),
	)
})
