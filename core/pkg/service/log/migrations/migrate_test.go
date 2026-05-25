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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/notation"
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
					"color":     "#0000ff",
					"notation":  "scientific",
					"precision": 2,
				},
			},
			"remoteCreated":        true,
			"timestampPrecision":   1,
			"showChannelNames":     true,
			"showReceiptTimestamp": false,
		}
		result := MustSucceed(migrations.Migrate(v1.Version, data))
		Expect(result.Channels).To(HaveLen(1))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(result.Channels[0].Color).To(Equal(color.MustFromHex("#0000ff")))
		Expect(result.RemoteCreated).To(BeTrue())
		Expect(result.ShowReceiptTimestamp).To(BeFalse())
	})

	It("Should reject a v1 payload with a malformed color hex", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 1, "color": "red"}},
		}
		Expect(migrations.Migrate(v1.Version, data)).Error().
			To(MatchError(ContainSubstring("invalid hex color")))
	})

	It("Should reject a v1 payload with a notation outside the closed set", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 1, "notation": "garbage"}},
		}
		Expect(migrations.Migrate(v1.Version, data)).Error().
			To(MatchError(ContainSubstring("invalid value \"garbage\"")))
	})

	It("Should parse a v0 payload and lift it forward to the latest", func() {
		data := map[string]any{
			"channels":      []any{1, 2, 3},
			"remoteCreated": true,
		}
		result := MustSucceed(migrations.Migrate(v0.Version, data))
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(result.Channels[0].Notation).To(Equal(notation.NotationStandard))
		Expect(result.Channels[0].Precision).To(Equal(int32(-1)))
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

	It("Should reject a v1 payload referencing a zero channel key", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 0}},
		}
		Expect(migrations.Migrate(v1.Version, data)).Error().
			To(MatchError(ContainSubstring("channels[0].channel")))
	})

	It("Should reject a v0 payload referencing a zero channel key", func() {
		data := map[string]any{"channels": []any{1, 0}}
		Expect(migrations.Migrate(v0.Version, data)).Error().
			To(MatchError(ContainSubstring("channels[1]")))
	})
})

var _ = Describe("MigrateLenient", func() {
	It("Should reject a version greater than the latest supported", func() {
		Expect(migrations.MigrateLenient(migrations.LatestVersion+1, map[string]any{})).
			Error().To(MatchError(ContainSubstring("newer than this Core supports")))
	})

	It("Should scrub a malformed color hex instead of erroring", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 1, "color": "not-a-hex"}},
		}
		result := MustSucceed(migrations.MigrateLenient(v1.Version, data))
		Expect(result.Channels).To(HaveLen(1))
		Expect(result.Channels[0].Color).To(Equal(color.Color{}))
	})

	It("Should leave an enum outside the closed set unchanged for the lift to default", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 1, "notation": "garbage"}},
		}
		result := MustSucceed(migrations.MigrateLenient(v1.Version, data))
		Expect(result.Channels[0].Notation).To(Equal(notation.Notation("garbage")))
	})

	It("Should not reject a zero channel key (lenient skips validation)", func() {
		data := map[string]any{
			"channels": []any{map[string]any{"channel": 0}},
		}
		result := MustSucceed(migrations.MigrateLenient(v1.Version, data))
		Expect(result.Channels).To(HaveLen(1))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(0)))
	})

	It("Should lift a v0 payload forward to the latest", func() {
		data := map[string]any{
			"channels":      []any{1, 2, 3},
			"remoteCreated": true,
		}
		result := MustSucceed(migrations.MigrateLenient(v0.Version, data))
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(result.RemoteCreated).To(BeTrue())
	})
})
