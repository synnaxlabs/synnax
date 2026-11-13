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
	base := func() v0.Base {
		return v0.Base{
			Key:        uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:       1,
			Name:       "T7-0001",
			Serial:     470010001,
			Connection: v0.ConnectionUsb,
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
})
