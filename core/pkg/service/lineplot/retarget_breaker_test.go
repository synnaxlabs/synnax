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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel retarget breaker", func() {
	It("Should leave an empty x axis empty when the map names the zero key", func(
		ctx SpecContext,
	) {
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, 3))
		a, b, c := keys[0], keys[1], keys[2]
		plot := defaults.Fill(lineplot.LinePlot{
			Project:  &proj.Key,
			Name:     "breaker",
			Channels: lineplot.Channels{Y1: []channel.Key{a}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &plot)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, plot.Key)
			})).To(Succeed())
		})
		id := lineplot.ResourceID(plot.Key)
		holders := []gorp.Dependent{{
			Holder:    gorp.Target{Table: string(id.Type), Key: id.Key},
			Reference: "channels",
		}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Retarget(
				ctx,
				map[channel.Key]channel.Key{a: b, 0: c},
				holders,
			)
		})).To(Succeed())
		var res lineplot.LinePlot
		Expect(svc.NewRetrieve().
			Where(lineplot.MatchKeys(plot.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res.Channels.Y1).To(Equal([]channel.Key{b}))
		Expect(res.Channels.X1).To(BeZero())
		Expect(res.Channels.X2).To(BeZero())
	})
})
