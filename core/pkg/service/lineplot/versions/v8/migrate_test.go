// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8_test

import (
	"context"
	"embed"
	"encoding/hex"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var fixtures embed.FS

func decodeV7(ctx context.Context) v7.LinePlot {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/v7_initial.hex"))
	wire := MustSucceed(hex.DecodeString(strings.ReplaceAll(string(raw), "\n", "")))
	var lp v7.LinePlot
	Expect(orc.Codec.Decode(ctx, wire, &lp)).To(Succeed())
	return lp
}

var _ = Describe("MigrateLine", func() {
	DescribeTable("Should keep the stored stroke width",
		func(ctx SpecContext, width float64) {
			out := MustSucceed(
				v8.MigrateLine(ctx, v7.Line{Key: "k", StrokeWidth: width}),
			)
			Expect(out.StrokeWidth).To(HaveValue(Equal(width)))
		},
		Entry("zero", 0.0),
		Entry("the old default", 2.0),
		Entry("a custom width", 3.5),
	)
})

// migrateStored writes stored to an in-memory v7 table, runs the v8 migration, and
// returns the migrated entry.
func migrateStored(ctx context.Context, stored v7.LinePlot) v8.LinePlot {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v7.Key, v7.LinePlot]{DB: db}))
	Expect(gorp.NewCreate[v7.Key, v7.LinePlot]().
		Entry(&stored).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "LinePlot",
		Migrations: []gorp.Migration{v8.Migration},
	})).To(Succeed())
	var got v8.LinePlot
	Expect(gorp.NewRetrieve[v8.Key, v8.LinePlot]().
		Where(gorp.MatchKeys[v8.Key, v8.LinePlot](stored.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got
}

var _ = Describe("Storage migration", func() {
	It("Should lift the pinned v7 payload", func(ctx SpecContext) {
		stored := decodeV7(ctx)
		got := migrateStored(ctx, stored)
		Expect(got).To(Equal(MustSucceed(v8.MigrateLinePlot(ctx, stored))))
		Expect(got.Lines).To(HaveLen(len(stored.Lines)))
		for i, l := range got.Lines {
			Expect(l.StrokeWidth).To(HaveValue(Equal(stored.Lines[i].StrokeWidth)))
		}
	})

	It("Should keep a stored stroke width of 0", func(ctx SpecContext) {
		got := migrateStored(ctx, v7.LinePlot{
			Key:   uuid.New(),
			Name:  "zero",
			Lines: []v7.Line{{Key: "k"}},
		})
		Expect(got.Lines[0].StrokeWidth).To(HaveValue(Equal(0.0)))
	})
})
