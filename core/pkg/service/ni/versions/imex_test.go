// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ni/versions"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type decoder func(context.Context, msgpack.EncodedJSON) (any, error)

func decodeV2[T any](ladder imex.Ladder[T]) decoder {
	return func(ctx context.Context, blob msgpack.EncodedJSON) (any, error) {
		return ladder.Decode(ctx, v2.Version, blob)
	}
}

var _ = Describe("Ladders", func() {
	entries := []TableEntry{
		Entry(
			"analog read",
			"ni.AnalogReadConfig",
			decodeV2(versions.AnalogReadConfigLadder),
		),
		Entry(
			"counter read",
			"ni.CounterReadConfig",
			decodeV2(versions.CounterReadConfigLadder),
		),
		Entry(
			"analog write",
			"ni.AnalogWriteConfig",
			decodeV2(versions.AnalogWriteConfigLadder),
		),
		Entry(
			"digital read",
			"ni.DigitalReadConfig",
			decodeV2(versions.DigitalReadConfigLadder),
		),
		Entry(
			"digital write",
			"ni.DigitalWriteConfig",
			decodeV2(versions.DigitalWriteConfigLadder),
		),
		Entry("scan", "ni.ScanConfig", decodeV2(versions.ScanConfigLadder)),
	}

	DescribeTable(
		"Should decode a v2 body",
		func(ctx SpecContext, _ string, decode decoder) {
			MustSucceed(decode(ctx, msgpack.EncodedJSON{"device": "dev"}))
		},
		entries,
	)

	DescribeTable(
		"Should reject a v2 body that does not decode",
		func(ctx SpecContext, resource string, decode decoder) {
			Expect(decode(ctx, msgpack.EncodedJSON{"key": 12})).Error().
				To(MatchError(ContainSubstring("decode v2 " + resource)))
		},
		entries,
	)

	It("Should drop the ignored models from a v2 scan body", func(ctx SpecContext) {
		cfg := MustSucceed(versions.ScanConfigLadder.Decode(
			ctx,
			v2.Version,
			msgpack.EncodedJSON{
				"rate":           5,
				"disabled":       true,
				"ignored_models": []string{"^cRIO.*"},
			},
		))
		Expect(cfg.Rate).To(Equal(telem.Rate(5)))
		Expect(cfg.Disabled).To(BeTrue())
	})

	It("Should clamp and disable a v2 digital channel out of range",
		func(ctx SpecContext) {
			cfg := MustSucceed(versions.DigitalReadConfigLadder.Decode(
				ctx,
				v2.Version,
				msgpack.EncodedJSON{"channels": []any{
					map[string]any{"key": "a", "port": -1, "line": 40},
				}},
			))
			Expect(cfg.Channels).To(HaveLen(1))
			Expect(cfg.Channels[0].Port).To(BeZero())
			Expect(cfg.Channels[0].Line).To(Equal(uint8(31)))
			Expect(cfg.Channels[0].Disabled).To(BeTrue())
		},
	)

	It("Should truncate a v2 fractional initial count", func(ctx SpecContext) {
		cfg := MustSucceed(versions.CounterReadConfigLadder.Decode(
			ctx,
			v2.Version,
			msgpack.EncodedJSON{"channels": []any{
				map[string]any{"type": "ci_edge_count", "initial_count": 3.7},
			}},
		))
		Expect(cfg.Channels[0].Variant).To(BeAssignableToTypeOf(
			versions.CIEdgeCountChannel{},
		))
		Expect(cfg.Channels[0].Variant.(versions.CIEdgeCountChannel).InitialCount).
			To(Equal(uint32(3)))
	})
})
