// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v7_test

import (
	"context"
	"embed"
	"encoding/hex"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var fixtures embed.FS

// loadWire returns the frozen Orc payload stored as hex in testdata. The fixtures are
// real stored entries captured from a running Core.
func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(
		strings.ReplaceAll(string(raw), "\n", ""),
	))
}

// staticCustomKey is the key a fixed v6 custom window migrates to.
var staticCustomKey = uuid.MustParse("cb056cea-fb6b-42aa-9991-d91bdf6db8ab")

func migrateLinePlot(ctx context.Context, old v6.LinePlot) v7.LinePlot {
	GinkgoHelper()
	return MustSucceed(v7.MigrateLinePlot(ctx, old))
}

func styledLine(key string) v6.Line {
	return v6.Line{Key: key, StrokeWidth: 3, Downsample: 2}
}

func linesOf(keys ...string) []v6.Line {
	lines := make([]v6.Line, len(keys))
	for i, k := range keys {
		lines[i] = styledLine(k)
	}
	return lines
}

func lineKeysOf(lines []v7.Line) []string {
	keys := make([]string, len(lines))
	for i, l := range lines {
		keys[i] = l.Key
	}
	return keys
}

func persisted(key uuid.UUID) v7.Range {
	return v7.Range{Variant: v7.PersistedRange{BaseRange: v7.BaseRange{Key: key}}}
}

var _ = Describe("MigrateLinePlot", func() {
	DescribeTable(
		"Should lift each built-in range key to its rolling span",
		func(ctx SpecContext, key string, span telem.TimeSpan) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{key}},
				Lines:  linesOf("y1---x1---" + key + "---1---2"),
			})
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{
				Rolling: &span,
				Ranges:  []v7.Range{},
			}))
			Expect(out.Lines).To(Equal(linesOf("y1---x1---rolling---1---2")))
		},
		Entry("recent", "recent", 30*telem.Second),
		Entry("rolling1m", "rolling1m", telem.Minute),
		Entry("rolling5m", "rolling5m", 5*telem.Minute),
		Entry("rolling15m", "rolling15m", 15*telem.Minute),
		Entry("rolling30m", "rolling30m", 30*telem.Minute),
		Entry("rolling1h", "rolling1h", telem.Hour),
		Entry("rolling6h", "rolling6h", 6*telem.Hour),
		Entry("rolling12h", "rolling12h", 12*telem.Hour),
		Entry("rolling1d", "rolling1d", telem.Day),
		Entry("rolling7d", "rolling7d", 7*telem.Day),
		Entry("rolling30d", "rolling30d", 30*telem.Day),
	)

	It("Should keep the first rolling key and drop the rest with their lines",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{"rolling5m", "recent", "rolling1h"}},
				Lines: linesOf(
					"y1---x1---rolling5m---1---2",
					"y1---x1---recent---1---2",
					"y1---x1---rolling1h---1---2",
				),
			})
			Expect(out.Ranges.X1.Rolling).To(Equal(new(5 * telem.Minute)))
			Expect(out.Ranges.X1.Ranges).To(BeEmpty())
			Expect(
				lineKeysOf(out.Lines),
			).To(Equal([]string{"y1---x1---rolling---1---2"}))
		})

	It("Should lift a dynamic custom window to the rolling window",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{
					X1: []string{"custom"},
					Custom: &v6.CustomRange{
						Variant: v6.DynamicCustomRange{Span: 45 * telem.Minute},
					},
				},
				Lines: linesOf("y1---x1---custom---1---2"),
			})
			Expect(out.Ranges.X1.Rolling).To(Equal(new(45 * telem.Minute)))
			Expect(out.Lines).To(Equal(linesOf("y1---x1---rolling---1---2")))
		})

	It("Should drop a dynamic custom window when a built-in key came first",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{
					X1: []string{"rolling1m", "custom"},
					Custom: &v6.CustomRange{
						Variant: v6.DynamicCustomRange{Span: 45 * telem.Minute},
					},
				},
				Lines: linesOf(
					"y1---x1---rolling1m---1---2",
					"y1---x1---custom---1---2",
				),
			})
			Expect(out.Ranges.X1.Rolling).To(Equal(new(telem.Minute)))
			Expect(
				lineKeysOf(out.Lines),
			).To(Equal([]string{"y1---x1---rolling---1---2"}))
		})

	It("Should lift a static custom window to a static range with a fixed key",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{
					X1: []string{"custom"},
					Custom: &v6.CustomRange{
						Variant: v6.StaticCustomRange{Start: 10, End: 20},
					},
				},
				Lines: linesOf("y1---x1---custom---1---2"),
			})
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{
				Ranges: []v7.Range{{Variant: v7.StaticRange{
					BaseRange: v7.BaseRange{Key: staticCustomKey},
					Start:     10,
					End:       20,
				}}},
			}))
			Expect(out.Lines).To(Equal(linesOf(
				"y1---x1---" + staticCustomKey.String() + "---1---2",
			)))
		})

	DescribeTable(
		"Should drop a custom key that names no window, along with its lines",
		func(ctx SpecContext, custom *v6.CustomRange) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{"custom"}, Custom: custom},
				Lines:  linesOf("y1---x1---custom---1---2"),
			})
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{Ranges: []v7.Range{}}))
			Expect(out.Lines).To(BeEmpty())
		},
		Entry("nil custom", nil),
		Entry("custom with no variant", &v6.CustomRange{}),
	)

	It("Should lift a UUID key to a persisted range with a canonical key",
		func(ctx SpecContext) {
			key := uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e6f")
			upper := strings.ToUpper(key.String())
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{upper}},
				Lines:  linesOf("y1---x1---" + upper + "---1---2"),
			})
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{
				Ranges: []v7.Range{persisted(key)},
			}))
			Expect(out.Lines).To(Equal(linesOf(
				"y1---x1---" + key.String() + "---1---2",
			)))
		})

	It("Should drop a key that is neither built in nor a UUID, along with its lines",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{"local-range"}},
				Lines:  linesOf("y1---x1---local-range---1---2"),
			})
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{Ranges: []v7.Range{}}))
			Expect(out.Lines).To(BeEmpty())
		})

	It("Should collapse a key the axis holds twice", func(ctx SpecContext) {
		key := uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e6f")
		out := migrateLinePlot(ctx, v6.LinePlot{
			Ranges: v6.Ranges{X1: []string{key.String(), key.String()}},
		})
		Expect(out.Ranges.X1.Ranges).To(Equal([]v7.Range{persisted(key)}))
	})

	It("Should drop lines whose key does not name a surviving range",
		func(ctx SpecContext) {
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{X1: []string{"recent"}},
				Lines: linesOf(
					"l1",
					"y1---x1---rolling1h---1---2",
					"y1---y2---recent---1---2",
					"y1---x1---recent---1---2",
				),
			})
			Expect(
				lineKeysOf(out.Lines),
			).To(Equal([]string{"y1---x1---rolling---1---2"}))
		})

	It("Should rename each axis's lines through that axis's own ranges",
		func(ctx SpecContext) {
			key := uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e6f")
			out := migrateLinePlot(ctx, v6.LinePlot{
				Ranges: v6.Ranges{
					X1: []string{"rolling1m"},
					X2: []string{key.String()},
				},
				Lines: linesOf(
					"y1---x1---rolling1m---1---2",
					"y1---x2---rolling1m---3---2",
					"y1---x2---"+key.String()+"---3---2",
				),
			})
			Expect(out.Ranges.X1.Rolling).To(Equal(new(telem.Minute)))
			Expect(out.Ranges.X2).To(Equal(v7.XAxisRanges{
				Ranges: []v7.Range{persisted(key)},
			}))
			Expect(lineKeysOf(out.Lines)).To(Equal([]string{
				"y1---x1---rolling---1---2",
				"y1---x2---" + key.String() + "---3---2",
			}))
		})

	It("Should carry every field the migration does not reshape",
		func(ctx SpecContext) {
			in := v6.LinePlot{
				Key:      uuid.MustParse("251ed216-d959-4684-ac64-a8475c146697"),
				Name:     "plot",
				Title:    v5.Title{Visible: true, Level: "h2"},
				Channels: v5.Channels{X1: 4, Y1: []channel.Key{1, 2}},
				Rules:    []v5.Rule{{Key: "r1", Label: "max", Position: 42}},
			}
			out := migrateLinePlot(ctx, in)
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.Name).To(Equal(in.Name))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Rules).To(Equal(in.Rules))
		})
})

var _ = Describe("MigrateRanges", func() {
	It(
		"Should split both axes into a rolling window and ranges",
		func(ctx SpecContext) {
			key := uuid.MustParse("5f6c1d3e-2a4b-4c8d-9e0f-1a2b3c4d5e6f")
			out := MustSucceed(v7.MigrateRanges(ctx, v6.Ranges{
				X1: []string{"rolling1h", key.String()},
				X2: []string{"recent"},
			}))
			Expect(out).To(Equal(v7.Ranges{
				X1: v7.XAxisRanges{
					Rolling: new(telem.Hour),
					Ranges:  []v7.Range{persisted(key)},
				},
				X2: v7.XAxisRanges{
					Rolling: new(30 * telem.Second),
					Ranges:  []v7.Range{},
				},
			}))
		},
	)
})

var _ = Describe("Released payloads", func() {
	It("Should lift a released v5 payload through v6 to v7", func(ctx SpecContext) {
		var lp5 v5.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v5_released.hex"), &lp5)).To(Succeed())
		lp6 := MustSucceed(v6.MigrateLinePlot(ctx, lp5))
		out := migrateLinePlot(ctx, lp6)
		Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{
			Rolling: new(telem.Hour),
			Ranges:  []v7.Range{},
		}))
		Expect(out.Lines).To(HaveLen(len(lp5.Lines)))
		for i, l := range out.Lines {
			Expect(l.Key).To(Equal(
				strings.Replace(
					lp5.Lines[i].Key,
					"---rolling1h---",
					"---rolling---",
					1,
				),
			))
			Expect(l.Key).To(ContainSubstring("---rolling---"))
		}
	})

	It("Should lift a stored v6 payload with a dynamic custom window",
		func(ctx SpecContext) {
			var lp6 v6.LinePlot
			Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &lp6)).
				To(Succeed())
			out := migrateLinePlot(ctx, lp6)
			Expect(out.Ranges.X1).To(Equal(v7.XAxisRanges{
				Rolling: new(telem.TimeSpan(108000000000000)),
				Ranges:  []v7.Range{},
			}))
			Expect(lineKeysOf(out.Lines)).To(Equal([]string{
				"y1---x1---rolling---0---1048581",
				"y1---x1---rolling---0---1048580",
			}))
		})

	It("Should round-trip a migrated plot through the v7 codec", func(ctx SpecContext) {
		var lp6 v6.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &lp6)).To(Succeed())
		out := migrateLinePlot(ctx, lp6)
		encoded := MustSucceed(orc.Codec.Encode(ctx, out))
		var decoded v7.LinePlot
		Expect(orc.Codec.Decode(ctx, encoded, &decoded)).To(Succeed())
		Expect(decoded).To(Equal(out))
	})
})

var _ = Describe("Storage migration", func() {
	It("Should lift a stored v6 entry to v7 through the gorp migration",
		func(ctx SpecContext) {
			var stored v6.LinePlot
			Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &stored)).
				To(Succeed())
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[v6.Key, v6.LinePlot]{DB: db},
			))
			Expect(gorp.NewCreate[v6.Key, v6.LinePlot]().
				Entry(&stored).Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: []migrate.Migration{v7.Migration},
			})).To(Succeed())
			var got v7.LinePlot
			Expect(gorp.NewRetrieve[v7.Key, v7.LinePlot]().
				Where(gorp.MatchKeys[v7.Key, v7.LinePlot](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(migrateLinePlot(ctx, stored)))
		})
})
