// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v2 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/log/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateLog", func() {
	It(
		"Should replace a precision of -1 with an absent precision",
		func(ctx SpecContext) {
			out := MustSucceed(v3.MigrateLog(ctx, v2.Log{
				Channels: []v2.ChannelEntry{{Channel: 1, Precision: -1}},
			}))
			Expect(out.Channels[0].Precision).To(BeNil())
		},
	)

	It("Should keep an explicit precision", func(ctx SpecContext) {
		out := MustSucceed(v3.MigrateLog(ctx, v2.Log{
			Channels: []v2.ChannelEntry{{Channel: 1, Precision: 3}},
		}))
		Expect(*out.Channels[0].Precision).To(Equal(uint8(3)))
	})

	It("Should keep a precision of zero", func(ctx SpecContext) {
		out := MustSucceed(v3.MigrateLog(ctx, v2.Log{
			Channels: []v2.ChannelEntry{{Channel: 1, Precision: 0}},
		}))
		Expect(*out.Channels[0].Precision).To(Equal(uint8(0)))
	})

	It("Should keep the timestamp precision", func(ctx SpecContext) {
		out := MustSucceed(v3.MigrateLog(ctx, v2.Log{TimestampPrecision: 2}))
		Expect(out.TimestampPrecision).To(Equal(uint8(2)))
	})
})
