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
