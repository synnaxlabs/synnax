// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/log/migrations/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/notation"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateData", func() {
	It("Should error on an unknown declared version", func() {
		Expect(legacy.MigrateData(msgpack.EncodedJSON{"version": "99.0.0"})).
			Error().To(MatchError(ContainSubstring("unknown log data version")))
	})

	It("Should error on an explicit empty version string", func() {
		Expect(legacy.MigrateData(msgpack.EncodedJSON{"version": ""})).
			Error().To(MatchError(ContainSubstring("invalid version")))
	})

	It("Should leave an enum outside the closed set unchanged for the lift to default", func() {
		blob := msgpack.EncodedJSON{
			"version":  "1.0.0",
			"channels": []any{map[string]any{"channel": 1, "notation": "garbage"}},
		}
		result := MustSucceed(legacy.MigrateData(blob))
		Expect(result.Channels[0].Notation).To(Equal(notation.Notation("garbage")))
	})

	It("Should not reject a zero channel key (the chain skips validation)", func() {
		blob := msgpack.EncodedJSON{
			"version":  "1.0.0",
			"channels": []any{map[string]any{"channel": 0}},
		}
		result := MustSucceed(legacy.MigrateData(blob))
		Expect(result.Channels).To(HaveLen(1))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(0)))
	})

	It("Should lift a v0 payload forward to the latest", func() {
		blob := msgpack.EncodedJSON{
			"version":       "0.0.0",
			"channels":      []any{1, 2, 3},
			"remoteCreated": true,
		}
		result := MustSucceed(legacy.MigrateData(blob))
		Expect(result.Channels).To(HaveLen(3))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
		Expect(result.RemoteCreated).To(BeTrue())
	})

	It("Should fall back to v0 when the blob carries no version field", func() {
		blob := msgpack.EncodedJSON{"channels": []any{1, 2}}
		result := MustSucceed(legacy.MigrateData(blob))
		Expect(result.Channels).To(HaveLen(2))
		Expect(result.Channels[0].Channel).To(Equal(channel.Key(1)))
	})

	It("Should produce a zero v1.Data for a nil blob", func() {
		result := MustSucceed(legacy.MigrateData(nil))
		Expect(result.Channels).To(BeEmpty())
	})
})
