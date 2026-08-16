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
	"github.com/synnaxlabs/synnax/pkg/service/labjack/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Write", func() {
	DescribeTable("legacy shapes",
		func(in, want msgpack.EncodedJSON) {
			Expect(legacy.Write.Apply(in)).To(Equal(want))
		},
		Entry("renames the v0 cmdKey and stateKey",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmdKey":   float64(7),
					"stateKey": float64(8),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_channel":   float64(7),
					"state_channel": float64(8),
				}},
			},
		),
		Entry("renames the stored snake spellings",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_key":   float64(7),
					"state_key": float64(8),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_channel":   float64(7),
					"state_channel": float64(8),
				}},
			},
		),
		Entry("keeps a canonical config unchanged",
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_channel":   float64(7),
					"state_channel": float64(8),
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"cmd_channel":   float64(7),
					"state_channel": float64(8),
				}},
			},
		),
	)
})
