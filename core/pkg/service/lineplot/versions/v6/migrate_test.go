// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"context"
	"embed"
	"encoding/hex"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v5 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v5"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
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
// real stored entries captured from a running Core, so they pin the wire format a
// released build wrote rather than whatever the current codec produces.
func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(
		strings.ReplaceAll(string(raw), "\n", ""),
	))
}

func decodeReleasedV5(ctx context.Context) v5.LinePlot {
	GinkgoHelper()
	var lp v5.LinePlot
	Expect(orc.Codec.Decode(ctx, loadWire("v5_released.hex"), &lp)).To(Succeed())
	return lp
}

var _ = Describe("Released v5 wire format", func() {
	It("Should decode a payload written by a released build", func(ctx SpecContext) {
		lp := decodeReleasedV5(ctx)
		Expect(lp.Key).To(Equal(uuid.MustParse("251ed216-d959-4684-ac64-a8475c146697")))
		Expect(lp.Name).To(Equal("UT Disk Free"))
		Expect(lp.Ranges.X1).To(Equal([]string{"rolling1h"}))
		Expect(lp.Ranges.X2).To(BeEmpty())
		Expect(lp.Channels.Y1).To(HaveLen(3))
		Expect(lp.Lines).To(HaveLen(3))
		for _, l := range lp.Lines {
			Expect(l.DownsampleMode).To(Equal(v5.DownsampleModeDecimate))
		}
	})

	It("Should re-encode the payload byte for byte", func(ctx SpecContext) {
		raw := loadWire("v5_released.hex")
		lp := decodeReleasedV5(ctx)
		Expect(orc.Codec.Encode(ctx, lp)).To(Equal(raw))
	})
})

var _ = Describe("MigrateLinePlot", func() {
	It("Should lift a released v5 payload with a nil custom range",
		func(ctx SpecContext) {
			in := decodeReleasedV5(ctx)
			out := MustSucceed(v6.MigrateLinePlot(ctx, in))
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.Name).To(Equal(in.Name))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges.X1).To(Equal(in.Ranges.X1))
			Expect(out.Ranges.X2).To(Equal(in.Ranges.X2))
			Expect(out.Ranges.Custom).To(BeNil())
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Lines).To(Equal(in.Lines))
			Expect(out.Rules).To(Equal(in.Rules))
		})

	It("Should round-trip the migrated plot through the v6 codec",
		func(ctx SpecContext) {
			out := MustSucceed(v6.MigrateLinePlot(ctx, decodeReleasedV5(ctx)))
			encoded := MustSucceed(orc.Codec.Encode(ctx, out))
			var decoded v6.LinePlot
			Expect(orc.Codec.Decode(ctx, encoded, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(out))
		})
})

var _ = Describe("MigrateRanges", func() {
	It("Should carry both axes and leave custom nil", func(ctx SpecContext) {
		out := MustSucceed(v6.MigrateRanges(
			ctx, v5.Ranges{X1: []string{"a", "b"}, X2: []string{"c"}},
		))
		Expect(out.X1).To(Equal([]string{"a", "b"}))
		Expect(out.X2).To(Equal([]string{"c"}))
		Expect(out.Custom).To(BeNil())
	})
})

var _ = Describe("Storage migration", func() {
	It("Should lift a stored v5 entry to v6 through the gorp migration",
		func(ctx SpecContext) {
			seed := decodeReleasedV5(ctx)
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[v5.Key, v5.LinePlot]{DB: db},
			))
			Expect(gorp.NewCreate[v5.Key, v5.LinePlot]().
				Entry(&seed).Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: []migrate.Migration{v6.Migration},
			})).To(Succeed())
			var got v6.LinePlot
			Expect(gorp.NewRetrieve[v6.Key, v6.LinePlot]().
				Where(gorp.MatchKeys[v6.Key, v6.LinePlot](seed.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got).To(Equal(MustSucceed(v6.MigrateLinePlot(ctx, seed))))
		})
})

var _ = Describe("v6 wire format", func() {
	It("Should decode a stored v6 payload with a custom range", func(ctx SpecContext) {
		var lp v6.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &lp)).To(Succeed())
		Expect(lp.Key).To(Equal(uuid.MustParse("f86c4bec-0bdd-4125-afac-f25d3e3a8711")))
		Expect(lp.Ranges.X1).To(Equal([]string{"custom"}))
		Expect(lp.Ranges.Custom).ToNot(BeNil())
		Expect(lp.Ranges.Custom.Variant).To(BeAssignableToTypeOf(
			v6.DynamicCustomRange{},
		))
		Expect(lp.Ranges.Custom.Variant.(v6.DynamicCustomRange).Span).
			To(BeNumerically(">", telem.TimeSpan(0)))
		Expect(lp.Lines).To(HaveLen(2))
	})

	It("Should re-encode the payload byte for byte", func(ctx SpecContext) {
		raw := loadWire("v6_initial.hex")
		var lp v6.LinePlot
		Expect(orc.Codec.Decode(ctx, raw, &lp)).To(Succeed())
		Expect(orc.Codec.Encode(ctx, lp)).To(Equal(raw))
	})

	It("Should reject a v6 payload under the v5 codec", func(ctx SpecContext) {
		var lp v5.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v6_initial.hex"), &lp)).
			To(truncatedErr)
	})
})
