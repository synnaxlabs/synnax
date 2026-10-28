// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("autoMigrate", func() {
	It("Should copy an analog read config", func(ctx SpecContext) {
		old := v2.AnalogReadConfig{SampleRate: 1000}
		Expect(MustSucceed(autoMigrateAnalogReadConfig(ctx, old))).To(Equal(old))
	})

	It("Should copy an analog write config", func(ctx SpecContext) {
		old := v2.AnalogWriteConfig{StateRate: 10}
		Expect(MustSucceed(autoMigrateAnalogWriteConfig(ctx, old))).To(Equal(old))
	})

	It("Should copy a counter read config", func(ctx SpecContext) {
		old := v2.CounterReadConfig{SampleRate: 1000}
		Expect(MustSucceed(autoMigrateCounterReadConfig(ctx, old))).To(Equal(old))
	})

	It("Should copy a digital read config", func(ctx SpecContext) {
		old := v2.DigitalReadConfig{Device: "dev", SampleRate: 1000}
		Expect(MustSucceed(autoMigrateDigitalReadConfig(ctx, old))).To(Equal(old))
	})

	It("Should copy a digital write config", func(ctx SpecContext) {
		old := v2.DigitalWriteConfig{StateRate: 10}
		Expect(MustSucceed(autoMigrateDigitalWriteConfig(ctx, old))).To(Equal(old))
	})
})
