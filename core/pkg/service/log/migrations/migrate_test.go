// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	It("Should reject a version greater than the latest supported", func() {
		Expect(migrations.Migrate(migrations.LatestVersion+1, map[string]any{})).Error().
			To(MatchError(ContainSubstring("newer than this Core supports")))
	})

	It("Should parse a well-formed v1 payload at v1.Version", func() {
		data := map[string]any{
			"channels": []any{
				map[string]any{
					"channel":   1,
					"color":     "blue",
					"notation":  "scientific",
					"precision": 2,
				},
			},
			"remote_created":         true,
			"timestamp_precision":    1,
			"show_channel_names":     true,
			"show_receipt_timestamp": false,
		}
		result := MustSucceed(migrations.Migrate(v1.Version, data))
		Expect(result.Channels).To(HaveLen(1))
		Expect(result.Channels[0].Channel).To(Equal(1))
		Expect(result.Channels[0].Color).To(Equal("blue"))
		Expect(result.RemoteCreated).To(BeTrue())
		Expect(result.ShowReceiptTimestamp).To(BeFalse())
	})

	It("Should reject a malformed v1 payload", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"color": "red"}}, // missing required channel
		}
		Expect(migrations.Migrate(v1.Version, data)).Error().
			To(MatchError(ContainSubstring("channel")))
	})

	It("Should parse a v0 payload and lift it forward to the latest", func() {
		data := map[string]any{
			"channels":       []any{1, 2, 3},
			"remote_created": true,
		}
		result := MustSucceed(migrations.Migrate(v0.Version, data))
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(1))
		Expect(result.Channels[0].Notation).To(Equal("standard"))
		Expect(result.Channels[0].Precision).To(Equal(-1))
		Expect(result.RemoteCreated).To(BeTrue())
		Expect(result.ShowChannelNames).To(BeTrue())
		Expect(result.ShowReceiptTimestamp).To(BeTrue())
	})

	It("Should reject a malformed v0 payload", func() {
		data := map[string]any{
			"channels": "not-an-array",
		}
		Expect(migrations.Migrate(v0.Version, data)).Error().
			To(MatchError(ContainSubstring("channels")))
	})
})
