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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateReadConfig", func() {
	It("Should keep a stored true as an override and drop a stored false",
		func(ctx SpecContext) {
			cfg := MustSucceed(v2.MigrateReadConfig(ctx, v1.ReadConfig{
				Device: "dev",
				Channels: []v1.ReadChannel{
					{Variant: v1.HoldingRegisterReadChannel{
						BaseReadChannel: v1.BaseReadChannel{Key: "reg", Channel: 7},
						RegisterValue: v1.RegisterValue{
							DataType:     "float32",
							BytesSwapped: true,
						},
						StringLength: 3,
					}},
					{Variant: v1.CoilReadChannel{
						BaseReadChannel: v1.BaseReadChannel{Key: "coil"},
					}},
				},
			}))
			Expect(cfg.Device).To(Equal("dev"))
			Expect(cfg.Channels).To(HaveLen(2))
			reg, ok := cfg.Channels[0].Variant.(v2.HoldingRegisterReadChannel)
			Expect(ok).To(BeTrue())
			Expect(reg.Key).To(Equal("reg"))
			Expect(reg.Channel).To(BeEquivalentTo(7))
			Expect(reg.DataType).To(BeEquivalentTo("float32"))
			Expect(reg.StringLength).To(BeEquivalentTo(3))
			Expect(reg.BytesSwapped).To(HaveValue(BeTrue()))
			Expect(reg.WordsSwapped).To(BeNil())
			coil, ok := cfg.Channels[1].Variant.(v2.CoilReadChannel)
			Expect(ok).To(BeTrue())
			Expect(coil.Key).To(Equal("coil"))
		},
	)
})

var _ = Describe("MigrateWriteConfig", func() {
	It("Should keep a stored true as an override and drop a stored false",
		func(ctx SpecContext) {
			cfg := MustSucceed(v2.MigrateWriteConfig(ctx, v1.WriteConfig{
				Channels: []v1.WriteChannel{
					{Variant: v1.HoldingRegisterWriteChannel{
						BaseWriteChannel: v1.BaseWriteChannel{Key: "reg"},
						RegisterValue: v1.RegisterValue{
							DataType:     "uint32",
							WordsSwapped: true,
						},
					}},
				},
			}))
			Expect(cfg.Channels).To(HaveLen(1))
			reg, ok := cfg.Channels[0].Variant.(v2.HoldingRegisterWriteChannel)
			Expect(ok).To(BeTrue())
			Expect(reg.Key).To(Equal("reg"))
			Expect(reg.DataType).To(BeEquivalentTo("uint32"))
			Expect(reg.BytesSwapped).To(BeNil())
			Expect(reg.WordsSwapped).To(HaveValue(BeTrue()))
		},
	)
})
