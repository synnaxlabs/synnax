// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/labjack/device/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Device", func() {
	device := func() v0.Device {
		return v0.Device{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "T7-0001",
			Serial:     470010001,
			Connection: v0.ConnectionUSB,
			Model:      v0.ModelT7,
			Ports:      v0.Ports{Variant: v0.T7Ports{}},
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Device]()).To(Equal("labjack_device"))
	})

	It("Should identify itself by its key", func() {
		d := device()
		Expect(d.GorpKey()).To(Equal(d.Key))
		Expect(d.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeLabjackDevice,
			Key:  d.Key.String(),
		}))
	})

	It("Should accept a valid device", func() {
		Expect(device().Validate()).To(Succeed())
	})

	It("Should accept the ports of each model", func() {
		d := device()
		d.Ports = v0.Ports{Variant: v0.T7Ports{
			AnalogInput: map[v0.T7AnalogInputPort]channel.Key{"AIN127": 1},
			AnalogOutput: map[v0.AnalogOutputPort]v0.CommandStatePair{
				"DAC1": {Command: 2, State: 3},
			},
			DigitalInput: map[v0.T7DigitalPort]channel.Key{"DIO0": 4},
			DigitalOutput: map[v0.T7DigitalPort]v0.CommandStatePair{
				"DIO22": {Command: 5, State: 6},
			},
		}}
		Expect(d.Validate()).To(Succeed())
	})

	DescribeTable("Should reject a port the model does not have",
		func(ports v0.PortsVariant, message string) {
			d := device()
			d.Ports = v0.Ports{Variant: ports}
			Expect(d.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry(
			"a T4 analog input",
			v0.T4Ports{AnalogInput: map[v0.T4AnalogInputPort]channel.Key{"AIN12": 1}},
			"invalid analog_input key: AIN12",
		),
		Entry(
			"a T4 digital input",
			v0.T4Ports{DigitalInput: map[v0.T4DigitalPort]channel.Key{"DIO0": 1}},
			"invalid digital_input key: DIO0",
		),
		Entry(
			"a T7 analog input",
			v0.T7Ports{AnalogInput: map[v0.T7AnalogInputPort]channel.Key{"AIN14": 1}},
			"invalid analog_input key: AIN14",
		),
		Entry(
			"a T7 analog output",
			v0.T7Ports{AnalogOutput: map[v0.AnalogOutputPort]v0.CommandStatePair{
				"DAC2": {Command: 1, State: 2},
			}},
			"invalid analog_output key: DAC2",
		),
		Entry(
			"a T8 digital output",
			v0.T8Ports{DigitalOutput: map[v0.T8DigitalPort]v0.CommandStatePair{
				"DIO20": {Command: 1, State: 2},
			}},
			"invalid digital_output key: DIO20",
		),
	)

	DescribeTable("Should reject an invalid device",
		func(change func(*v0.Device), message string) {
			d := device()
			change(&d)
			Expect(d.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("no name", func(d *v0.Device) { d.Name = "" }, "name"),
		Entry("no serial", func(d *v0.Device) { d.Serial = 0 }, "serial"),
		Entry("no model", func(d *v0.Device) { d.Model = "" }, "model"),
		Entry("no ports", func(d *v0.Device) { d.Ports = v0.Ports{} }, "ports"),
		Entry(
			"an unknown connection",
			func(d *v0.Device) { d.Connection = "LJM_ctTCP" },
			"invalid connection",
		),
		Entry(
			"an identifier that starts with a digit",
			func(d *v0.Device) { d.Identifier = "1lj" },
			"Identifier must be 2 to 12 characters",
		),
	)
})
