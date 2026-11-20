// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete checks", func() {
	It("Should list a record that holds the index the delete of a calculation removes",
		func(ctx SpecContext) {
			node := testutil.NewNode(ctx)
			db := node.DB
			layer := MustOpen(svctestutil.OpenLayer(ctx, node))
			proj := project.Project{Name: "Impact checks"}
			input := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
					return err
				}
				return layer.Channel.NewWriter(tx).Create(ctx, &input)
			})).To(Succeed())
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return " + input.Name + " * 2",
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &calc)
			})).To(Succeed())
			Expect(calc.Index()).ToNot(BeZero())
			// The plot holds the calculation's synthesized index, which the delete of
			// the calculation also removes.
			plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
			plot.Channels.Y1 = []channel.Key{calc.Index()}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.LinePlot.NewWriter(tx).Create(ctx, &plot)
			})).To(Succeed())
			var applied impact.Plan
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				applied, err = layer.Impact.Apply(ctx, tx, impact.Change{
					Variant: impact.DeleteChannelsChange{
						Keys: []channel.Key{calc.Key()},
					},
				}, impact.ApplyOptions{})
				return err
			})).To(Succeed())
			Expect(MustSucceed(layer.Channel.NewRetrieve().
				Where(channel.MatchKeys(calc.Index())).
				Exists(ctx, db))).To(BeFalse())
			records := make([]string, len(applied.Entries))
			for i, e := range applied.Entries {
				records[i] = e.Record.String()
			}
			Expect(records).To(ContainElement(lineplot.ResourceID(plot.Key).String()))
		},
	)
})
