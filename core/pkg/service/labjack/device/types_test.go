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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/labjack/device"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Device", func() {
	valid := device.Device{
		Key:             uuid.New(),
		Rack:            1,
		Name:            "T7",
		Serial:          470010001,
		Connection:      device.ConnectionWifi,
		Identifier:      "lj",
		ReadIndex:       new(channel.Key(1)),
		WriteStateIndex: new(channel.Key(2)),
		Model:           device.ModelT7,
		Ports: device.Ports{Variant: device.T7Ports{
			Mux80Attached: true,
			AnalogInput: map[device.T7AnalogInputPort]channel.Key{
				device.T7AnalogInputPortAin48: 3,
			},
			AnalogOutput: map[device.AnalogOutputPort]device.CommandStatePair{
				device.AnalogOutputPortDac0: {Command: 4, State: new(channel.Key(5))},
			},
			DigitalInput: map[device.T7DigitalPort]channel.Key{
				device.T7DigitalPortDio0: 6,
			},
			DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
				device.T7DigitalPortDio1: {Command: 7, State: new(channel.Key(8))},
			},
		}},
	}
	with := func(change func(*device.Device)) device.Device {
		d := valid
		change(&d)
		return d
	}

	It("Should keep the ports through JSON", func() {
		var got device.Device
		Expect(json.Unmarshal(MustSucceed(json.Marshal(valid)), &got)).To(Succeed())
		Expect(got).To(Equal(valid))
	})

	It("Should reject an unknown ports model in JSON", func() {
		var got device.Ports
		Expect(json.Unmarshal([]byte(`{"model":"t9"}`), &got)).
			To(MatchError(validate.ErrValidation))
	})

	It("Should accept a valid device", func() {
		Expect(valid.Validate()).To(Succeed())
	})

	DescribeTable("Should reject an invalid device",
		func(d device.Device, field string) {
			Expect(d.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
		},
		Entry("with no serial", with(func(d *device.Device) {
			d.Serial = 0
		}), "serial"),
		Entry("with an unknown connection", with(func(d *device.Device) {
			d.Connection = "LJM_ctTCP"
		}), "connection"),
		Entry("with an unknown model", with(func(d *device.Device) {
			d.Model = "t9"
		}), "model"),
		Entry("with no name", with(func(d *device.Device) { d.Name = "" }), "name"),
		Entry("with no ports", with(func(d *device.Device) {
			d.Ports = device.Ports{}
		}), "ports"),
		Entry("with a command channel of zero", with(func(d *device.Device) {
			d.Ports = device.Ports{Variant: device.T7Ports{
				DigitalOutput: map[device.T7DigitalPort]device.CommandStatePair{
					device.T7DigitalPortDio1: {},
				},
			}}
		}), "command"),
	)

	DescribeTable("Should check the identifier",
		func(identifier string, valid bool) {
			err := with(func(d *device.Device) {
				d.Identifier = identifier
			}).Validate()
			if valid {
				Expect(err).ToNot(HaveOccurred())
				return
			}
			Expect(err).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("Identifier must be 2 to 12 characters")),
			))
		},
		Entry("empty", "", true),
		Entry("two characters", "lj", true),
		Entry("twelve characters with digits and underscores", "LabJack_T7_1", true),
		Entry("one character", "l", false),
		Entry("thirteen characters", "LabJack_T7_12", false),
		Entry("a leading digit", "7lj", false),
		Entry("a leading underscore", "_lj", false),
		Entry("a hyphen", "lab-jack", false),
	)
})
