// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/versions"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Stored v7 line plots", func() {
	It(
		"Should drop key references to ranges the Core does not hold",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			// Mark the chain through v7 as applied, so only the v8 steps run on open.
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: versions.Migrations[:len(versions.Migrations)-2],
			})).To(Succeed())
			saved := ranger.Range{
				Key:       uuid.New(),
				Name:      "saved",
				TimeRange: telem.TimeRange{Start: 1, End: 2},
			}
			Expect(gorp.NewCreate[ranger.Key, ranger.Range]().
				Entry(&saved).Exec(ctx, db)).To(Succeed())
			local := uuid.New().String()
			old := v7.LinePlot{
				Key:      uuid.New(),
				Name:     "plot",
				Channels: v7.Channels{Y1: []channel.Key{5}},
				Ranges: v7.Ranges{X1: []string{
					saved.Key.String(), local, "not-a-uuid", "rolling1m",
				}},
				Lines: []v7.Line{
					{
						Key:         "y1---x1---" + saved.Key.String() + "---0---5",
						StrokeWidth: 3,
					},
					{Key: "y1---x1---" + local + "---0---5", StrokeWidth: 4},
					{Key: "y1---x1---rolling1m---0---5", StrokeWidth: 5},
				},
			}
			Expect(gorp.NewCreate[v7.Key, v7.LinePlot]().
				Entry(&old).Exec(ctx, db)).To(Succeed())
			otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
				DB:       db,
				Ontology: otg,
				ImEx:     imex.NewService(),
			}))
			var got lineplot.LinePlot
			Expect(gorp.NewRetrieve[lineplot.Key, lineplot.LinePlot]().
				Where(gorp.MatchKeys[lineplot.Key, lineplot.LinePlot](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Ranges).To(Equal([]ranger.Ref{
				keyRef(saved.Key.String()),
				{Variant: ranger.WindowRef{
					Window: telem.Window{
						Variant: telem.DynamicWindow{TimeSpan: telem.Minute},
					},
				}},
			}))
			// The three v7 lines style one channel, so they merge into one line.
			Expect(got.Lines).To(HaveLen(1))
		},
	)
})
