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
	base := func() v0.Base {
		return v0.Base{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "T7-0001",
			Serial:     470010001,
			Connection: v0.ConnectionUSB,
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Device]()).To(Equal("labjack_device"))
	})

	It("Should identify itself by the key of its base", func() {
		d := v0.Device{Variant: v0.T7Device{Base: base()}}
		Expect(d.GorpKey()).To(Equal(base().Key))
		Expect(d.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeLabjackDevice,
			Key:  base().Key.String(),
		}))
		Expect(d.Model()).To(Equal(v0.ModelT7))
	})

	It("Should accept a valid device", func() {
		Expect(v0.Device{Variant: v0.T4Device{Base: base()}}.Validate()).To(Succeed())
	})

	DescribeTable("Should reject an invalid device",
		func(change func(*v0.Base), message string) {
			b := base()
			change(&b)
			Expect(v0.Device{Variant: v0.T8Device{Base: b}}.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry("no name", func(b *v0.Base) { b.Name = "" }, "name"),
		Entry("no serial", func(b *v0.Base) { b.Serial = 0 }, "serial"),
		Entry(
			"an unknown connection",
			func(b *v0.Base) { b.Connection = "LJM_ctWIFI" },
			"invalid connection",
		),
	)

	It("Should accept the ports of each model", func() {
		Expect(v0.Device{Variant: v0.T7Device{
			Base:        base(),
			AnalogInput: map[v0.T7AnalogInputPort]channel.Key{"AIN127": 1},
			AnalogOutput: map[v0.AnalogOutputPort]v0.CommandStatePair{
				"DAC1": {Command: 2, State: 3},
			},
			DigitalInput: map[v0.T7DigitalPort]channel.Key{"DIO0": 4},
			DigitalOutput: map[v0.T7DigitalPort]v0.CommandStatePair{
				"DIO22": {Command: 5, State: 6},
			},
		}}.Validate()).To(Succeed())
	})

	DescribeTable("Should reject a port the model does not have",
		func(variant v0.DeviceVariant, message string) {
			Expect(v0.Device{Variant: variant}.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(message)),
			))
		},
		Entry(
			"a T4 analog input",
			v0.T4Device{
				Base:        base(),
				AnalogInput: map[v0.T4AnalogInputPort]channel.Key{"AIN12": 1},
			},
			"invalid analog_input key: AIN12",
		),
		Entry(
			"a T4 digital input",
			v0.T4Device{
				Base:         base(),
				DigitalInput: map[v0.T4DigitalPort]channel.Key{"DIO0": 1},
			},
			"invalid digital_input key: DIO0",
		),
		Entry(
			"a T7 analog input",
			v0.T7Device{
				Base:        base(),
				AnalogInput: map[v0.T7AnalogInputPort]channel.Key{"AIN14": 1},
			},
			"invalid analog_input key: AIN14",
		),
		Entry(
			"a T7 analog output",
			v0.T7Device{
				Base: base(),
				AnalogOutput: map[v0.AnalogOutputPort]v0.CommandStatePair{
					"DAC2": {Command: 1, State: 2},
				},
			},
			"invalid analog_output key: DAC2",
		),
		Entry(
			"a T8 digital output",
			v0.T8Device{
				Base: base(),
				DigitalOutput: map[v0.T8DigitalPort]v0.CommandStatePair{
					"DIO20": {Command: 1, State: 2},
				},
			},
			"invalid digital_output key: DIO20",
		),
	)
})
