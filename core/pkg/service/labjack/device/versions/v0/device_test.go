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

	Describe("ValidatePorts", func() {
		It("Should accept the ports of the device's model", func() {
			Expect(device().ValidatePorts()).To(Succeed())
		})

		It("Should reject the ports of another model", func() {
			d := device()
			d.Ports = v0.Ports{Variant: v0.T4Ports{}}
			Expect(d.ValidatePorts()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("ports: are for t4, not t7")),
			))
		})
	})

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
