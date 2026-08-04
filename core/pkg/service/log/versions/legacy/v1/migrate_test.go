// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	notation "github.com/synnaxlabs/x/notation/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("Migrate", func() {
	It("Should convert bare channel keys to config entries with defaults", func() {
		old := v0.Data{
			Channels:      []channel.Key{1, 2, 3},
			RemoteCreated: false,
		}
		result := v1.Migrate(old)
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(result.Channels[0].Color).To(BeEmpty())
		Expect(result.Channels[0].Notation).To(Equal(notation.NotationStandard))
		Expect(result.Channels[0].Precision).To(Equal(int32(-1)))
		Expect(result.Channels[0].Alias).To(Equal(""))
		Expect(
			result.Channels[0].Timestamp.Format,
		).To(Equal(telem.TimestampFormatPreciseDate))
		Expect(result.Channels[0].Timestamp.Tz).To(Equal(telem.TimeZoneLocal))
		Expect(result.Channels[2].Channel).To(Equal(channel.Key(3)))
	})

	It("Should preserve RemoteCreated", func() {
		old := v0.Data{Channels: []channel.Key{}, RemoteCreated: true}
		result := v1.Migrate(old)
		Expect(result.RemoteCreated).To(BeTrue())
	})

	It("Should set correct v1 defaults", func() {
		old := v0.Data{Channels: []channel.Key{}, RemoteCreated: false}
		result := v1.Migrate(old)
		Expect(result.TimestampPrecision).To(Equal(int32(0)))
		Expect(result.ShowChannelNames).To(BeTrue())
		Expect(result.ShowReceiptTimestamp).To(BeTrue())
	})

	It("Should handle empty channels", func() {
		old := v0.Data{Channels: []channel.Key{}, RemoteCreated: false}
		result := v1.Migrate(old)
		Expect(result.Channels).To(BeEmpty())
	})
})
