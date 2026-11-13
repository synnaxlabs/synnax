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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ni/versions"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v4"
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

type portDecoder func(context.Context, msgpack.EncodedJSON) (uint16, bool)

// decodePort decodes a v3 body through ladder and returns the port and disabled state
// of the first channel.
func decodePort[T any](
	ladder imex.Ladder[T],
	first func(T) (uint16, bool),
) portDecoder {
	return func(ctx context.Context, blob msgpack.EncodedJSON) (uint16, bool) {
		GinkgoHelper()
		return first(MustSucceed(ladder.Decode(ctx, v3.Version, blob)))
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

	It("Should keep the port and line of a v2 body", func(ctx SpecContext) {
		cfg := MustSucceed(versions.DigitalReadConfigLadder.Decode(
			ctx,
			v2.Version,
			msgpack.EncodedJSON{"channels": []any{
				map[string]any{"key": "a", "port": 2, "line": 5},
			}},
		))
		Expect(cfg.Channels).To(HaveLen(1))
		Expect(cfg.Channels[0].Port).To(Equal(uint16(2)))
		Expect(cfg.Channels[0].Line).To(Equal(uint8(5)))
	})

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

	DescribeTable("Should clamp an imported v3 port above 65535",
		func(ctx SpecContext, channel map[string]any, decode portDecoder) {
			channel["key"] = "a"
			channel["port"] = 70000
			port, disabled := decode(
				ctx,
				msgpack.EncodedJSON{"channels": []any{channel}},
			)
			Expect(port).To(Equal(uint16(math.MaxUint16)))
			Expect(disabled).To(BeTrue())
		},
		Entry(
			"digital read",
			map[string]any{"line": 2},
			decodePort(versions.DigitalReadConfigLadder, func(
				c v4.DigitalReadConfig,
			) (uint16, bool) {
				return c.Channels[0].Port, c.Channels[0].Disabled
			}),
		),
		Entry(
			"digital write",
			map[string]any{"line": 2},
			decodePort(versions.DigitalWriteConfigLadder, func(
				c v4.DigitalWriteConfig,
			) (uint16, bool) {
				return c.Channels[0].Port, c.Channels[0].Disabled
			}),
		),
		Entry(
			"analog read",
			map[string]any{"type": "ai_voltage"},
			decodePort(versions.AnalogReadConfigLadder, func(
				c v4.AnalogReadConfig,
			) (uint16, bool) {
				ch := c.Channels[0].Variant.(v4.AIVoltageChannel)
				return ch.Port, ch.Disabled
			}),
		),
		Entry(
			"counter read",
			map[string]any{"type": "ci_frequency"},
			decodePort(versions.CounterReadConfigLadder, func(
				c v4.CounterReadConfig,
			) (uint16, bool) {
				base := c.Channels[0].Base()
				return base.Port, base.Disabled
			}),
		),
		Entry(
			"analog write",
			map[string]any{"type": "ao_voltage"},
			decodePort(versions.AnalogWriteConfigLadder, func(
				c v4.AnalogWriteConfig,
			) (uint16, bool) {
				base := c.Channels[0].Base()
				return base.Port, base.Disabled
			}),
		),
	)

	DescribeTable("Should make an imported function generator frequency positive",
		func(ctx SpecContext, version imex.Version, frequency any) {
			channel := map[string]any{"key": "gen", "type": "ao_func_gen"}
			if frequency != nil {
				channel["frequency"] = frequency
			}
			cfg := MustSucceed(versions.AnalogWriteConfigLadder.Decode(
				ctx,
				version,
				msgpack.EncodedJSON{"channels": []any{channel}},
			))
			Expect(cfg.Channels).To(HaveLen(1))
			Expect(cfg.Channels[0].Variant.(v4.AOFuncGenChannel).Frequency).
				To(Equal(1.0))
			Expect(cfg.Validate()).To(Succeed())
		},
		Entry("v3 at 0 Hz", v3.Version, 0.0),
		Entry("v3 with no frequency", v3.Version, nil),
		Entry("v3 below 0 Hz", v3.Version, -5.0),
		Entry("v2 at 0 Hz", v2.Version, 0.0),
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
