// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package modbus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Update", func() {
	coil := func() modbus.ReadChannel {
		return modbus.ReadChannel{Variant: modbus.CoilReadChannel{
			BaseReadChannel: modbus.BaseReadChannel{
				Key:     "c1",
				Name:    "Valve",
				Address: 4,
			},
		}}
	}

	It("Should require a field that has no default to change the variant", func() {
		ch := coil()
		u := modbus.ReadChannelUpdate{
			Variant: modbus.HoldingRegisterStringReadChannelUpdate{Key: "c1"},
		}
		Expect(u.ApplyTo(&ch)).To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("length is required")),
		))
		Expect(ch).To(Equal(coil()))
	})

	It("Should change the variant when the update gives the field", func() {
		ch := coil()
		u := modbus.ReadChannelUpdate{
			Variant: modbus.HoldingRegisterStringReadChannelUpdate{
				Key:    "c1",
				Length: new(uint16(8)),
			},
		}
		Expect(u.ApplyTo(&ch)).To(Succeed())
		next := ch.Variant.(modbus.HoldingRegisterStringReadChannel)
		Expect(next.Length).To(Equal(uint16(8)))
		Expect(next.BaseReadChannel).To(Equal(coil().Base()))
	})
})
