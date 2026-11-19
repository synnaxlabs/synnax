// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"context"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var (
	db         *gorp.DB
	channelSvc *channel.Service
	creator    calctestutil.Creator
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		TS:      node.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	creator = calctestutil.Creator{DB: db, Channel: channelSvc}
})

// setExpression applies a set of expr to the expression of ch, then reads ch back.
func setExpression(ctx context.Context, ch *channel.Channel, expr string) {
	GinkgoHelper()
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := channelSvc.NewWriter(tx).Apply(ctx, ch.Key(), channel.NewSetAction(
			channel.SetPayload{Path: []string{"expression"}, Value: expr},
		))
		return err
	})).To(Succeed())
	Expect(channelSvc.NewRetrieve().
		Where(channel.MatchKeys(ch.Key())).
		Entry(ch).
		Exec(ctx, db)).To(Succeed())
}

var _ = Describe("Graph", func() {
	var g *graph.Graph
	BeforeEach(func() {
		g = MustSucceed(graph.New(graph.Config{
			Channel: channelSvc,
			DB:      db,
		}))
	})
	Describe("Add", func() {
		It("Should compile and add a simple channel", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base1", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc1",
				DataType:   telem.Int64T,
				Expression: "return base1 * 2",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})

		It("Should handle nested calculated dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base2", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc2",
				DataType:   telem.Int64T,
				Expression: "return base2 + 1",
			}))
			calc2 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc3",
				DataType:   telem.Int64T,
				Expression: "return calc2 * 2",
			}))
			Expect(g.Add(ctx, calc2)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(grouped[0]).To(HaveLen(2))
		})

		It("Should detect circular dependencies", func(ctx SpecContext) {
			calc1 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "circ1",
				DataType:   telem.Int64T,
				Expression: "return 1",
			}))
			MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "circ2",
				DataType:   telem.Int64T,
				Expression: "return circ1",
			}))
			setExpression(ctx, &calc1, "return circ2")
			Expect(
				g.Add(ctx, calc1),
			).To(MatchError(graph.ErrCycle))
		})

		It("Should not re-add existing channel", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base3", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc4",
				DataType:   telem.Int64T,
				Expression: "return base3",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			for _, mods := range grouped {
				Expect(mods).To(HaveLen(1))
			}
		})
	})

	Describe("CalculateGrouped", func() {
		It("Should group channels by base dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base4", DataType: telem.Int64T, Virtual: true},
				{Name: "base5", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc5",
					DataType:   telem.Int64T,
					Expression: "return base4 + 1",
				},
				{
					Name:       "calc6",
					DataType:   telem.Int64T,
					Expression: "return base4 * 2",
				},
				{
					Name:       "calc7",
					DataType:   telem.Int64T,
					Expression: "return base5 - 1",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(2))
		})

		It("Should group a channel with the calculated channels it reads", func(
			ctx SpecContext,
		) {
			bases := []channel.Channel{
				{Name: "splita1", DataType: telem.Int64T, Virtual: true},
				{Name: "splita2", DataType: telem.Int64T, Virtual: true},
				{Name: "splitb1", DataType: telem.Int64T, Virtual: true},
				{Name: "splitb2", DataType: telem.Int64T, Virtual: true},
				{Name: "splitb3", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "splitsuma",
					DataType:   telem.Int64T,
					Expression: "return splita1 + splita2",
				},
				{
					Name:       "splitsumb",
					DataType:   telem.Int64T,
					Expression: "return splitb1 + splitb2 + splitb3",
				},
				{
					Name:       "splitest",
					DataType:   telem.Int64T,
					Expression: "return splitsuma + splitsumb",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.CalculateGrouped()).To(HaveLen(2))
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			for _, mods := range grouped {
				Expect(mods).To(HaveLen(3))
				Expect(mods[2].Channel.Key()).To(Equal(calcs[2].Key()))
			}
		})

		It("Should reuse groups with superset dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base6", DataType: telem.Int64T, Virtual: true},
				{Name: "base7", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc8",
					DataType:   telem.Int64T,
					Expression: "return base6",
				},
				{
					Name:       "calc9",
					DataType:   telem.Int64T,
					Expression: "return base6 + base7",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})

		It(
			"Should order the independent members of a group by key",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     "base_order",
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &base)
				})).To(Succeed())
				calculations := make([]calctestutil.Calculation, 8)
				for i := range calculations {
					calculations[i] = calctestutil.Calculation{
						Name:       fmt.Sprintf("calc_order_%d", i),
						DataType:   telem.Int64T,
						Expression: "return base_order + 1",
					}
				}
				calcs := MustSucceed(creator.CreateMany(ctx, calculations))
				calcs = append(calcs, MustSucceed(creator.Indexes(ctx, calcs))...)
				for _, calc := range calcs {
					Expect(g.Add(ctx, calc)).To(Succeed())
				}
				keys := channel.KeysFromChannels(calcs)
				slices.Sort(keys)
				for range 10 {
					grouped := g.CalculateGrouped()
					Expect(grouped).To(HaveLen(1))
					for _, mods := range grouped {
						Expect(lo.Map(mods, func(m compiler.Module, _ int) channel.Key {
							return m.Channel.Key()
						})).To(Equal([]channel.Key(keys)))
					}
				}
			},
		)

		It(
			"Should order the dependencies of a nested member by key",
			func(ctx SpecContext) {
				base := channel.Channel{
					Name:     "base_nested_order",
					DataType: telem.Int64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &base)
				})).To(Succeed())
				calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
					{
						Name:       "calc_nested_order_a",
						DataType:   telem.Int64T,
						Expression: "return base_nested_order",
					},
					{
						Name:       "calc_nested_order_b",
						DataType:   telem.Int64T,
						Expression: "return base_nested_order + 1",
					},
					{
						Name:       "calc_nested_order_c",
						DataType:   telem.Int64T,
						Expression: "return base_nested_order + 2",
					},
				}))
				calcs = append(calcs, MustSucceed(creator.Indexes(ctx, calcs))...)
				for _, calc := range calcs {
					Expect(g.Add(ctx, calc)).To(Succeed())
				}
				setExpression(
					ctx,
					&calcs[0],
					"return calc_nested_order_b + calc_nested_order_c",
				)
				Expect(g.Update(ctx, calcs[0])).To(Succeed())
				order := func() []channel.Key {
					GinkgoHelper()
					grouped := g.CalculateGrouped()
					Expect(grouped).To(HaveLen(1))
					for _, mods := range grouped {
						return lo.Map(mods, func(m compiler.Module, _ int) channel.Key {
							return m.Channel.Key()
						})
					}
					return nil
				}
				first := order()
				Expect(slices.Index(first, calcs[1].Key())).
					To(BeNumerically("<", slices.Index(first, calcs[2].Key())))
				Expect(slices.Index(first, calcs[2].Key())).
					To(BeNumerically("<", slices.Index(first, calcs[0].Key())))
				for range 20 {
					Expect(order()).To(Equal(first))
				}
			},
		)
	})

	Describe("CalculateFlat", func() {
		It("Should return all modules in topological order", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "flatbase1", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "flat1",
					DataType:   telem.Int64T,
					Expression: "return flatbase1",
				},
				{
					Name:       "flat2",
					DataType:   telem.Int64T,
					Expression: "return flat1 * 2",
				},
				{
					Name:       "flat3",
					DataType:   telem.Int64T,
					Expression: "return flat2 + 1",
				},
			}))
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			flat := g.CalculateFlat()
			Expect(flat).To(HaveLen(3))
			Expect(flat[0].Channel.Name).To(Equal("flat1"))
			Expect(flat[1].Channel.Name).To(Equal("flat2"))
			Expect(flat[2].Channel.Name).To(Equal("flat3"))
		})

		It("Should return empty list when no channels", func() {
			flat := g.CalculateFlat()
			Expect(flat).To(BeEmpty())
		})
	})

	Describe("Remove", func() {
		It("Should remove a channel from allocator", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base8", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc10",
				DataType:   telem.Int64T,
				Expression: "return base8",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It("Should clean up empty groups", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base9", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc11",
					DataType:   telem.Int64T,
					Expression: "return base9",
				},
				{
					Name:       "calc12",
					DataType:   telem.Int64T,
					Expression: "return base9 * 2",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(MustSucceed(g.Remove(calcs[0].Key()))).To(BeTrue())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(MustSucceed(g.Remove(calcs[1].Key()))).To(BeTrue())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It("Should return false for non-existent channel", func() {
			Expect(MustSucceed(g.Remove(channel.Key(99999)))).To(BeFalse())
		})
	})

	Describe("Reference Counting", func() {
		It("Should increment explicit count on multiple adds", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base12", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc16",
				DataType:   telem.Int64T,
				Expression: "return base12",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
		It(
			"Should cascade remove dependencies when parent removed",
			func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "base13", DataType: telem.Int64T, Virtual: true},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
				})).To(Succeed())
				calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
					{
						Name:       "calc17",
						DataType:   telem.Int64T,
						Expression: "return base13",
					},
					{
						Name:       "calc18",
						DataType:   telem.Int64T,
						Expression: "return calc17 * 2",
					},
				}))
				Expect(g.Add(ctx, calcs[1])).To(Succeed())
				grouped := g.CalculateGrouped()
				Expect(grouped[0]).To(HaveLen(2))
				Expect(MustSucceed(g.Remove(calcs[1].Key()))).To(BeTrue())
				grouped = g.CalculateGrouped()
				Expect(grouped).To(BeEmpty())
			},
		)
		It(
			"Should not remove dep if still referenced by another channel",
			func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "base14", DataType: telem.Int64T, Virtual: true},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
				})).To(Succeed())
				calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
					{
						Name:       "calc19",
						DataType:   telem.Int64T,
						Expression: "return base14",
					},
					{
						Name:       "calc20",
						DataType:   telem.Int64T,
						Expression: "return calc19 + 1",
					},
					{
						Name:       "calc21",
						DataType:   telem.Int64T,
						Expression: "return calc19 * 2",
					},
				}))
				Expect(g.Add(ctx, calcs[1])).To(Succeed())
				Expect(g.Add(ctx, calcs[2])).To(Succeed())
				grouped := g.CalculateGrouped()
				Expect(grouped[0]).To(HaveLen(3))
				Expect(MustSucceed(g.Remove(calcs[1].Key()))).To(BeTrue())
				grouped = g.CalculateGrouped()
				Expect(grouped[0]).To(HaveLen(2))
				Expect(MustSucceed(g.Remove(calcs[2].Key()))).To(BeTrue())
				grouped = g.CalculateGrouped()
				Expect(grouped).To(BeEmpty())
			},
		)
		It("Should handle explicit request on dependency", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base15", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc22",
					DataType:   telem.Int64T,
					Expression: "return base15",
				},
				{
					Name:       "calc23",
					DataType:   telem.Int64T,
					Expression: "return calc22 + 1",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(MustSucceed(g.Remove(calcs[1].Key()))).To(BeTrue())
			grouped := g.CalculateGrouped()
			Expect(grouped[0]).To(HaveLen(1))
			Expect(MustSucceed(g.Remove(calcs[0].Key()))).To(BeTrue())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
	})

	Describe("CalculatedKeys", func() {
		It("Should return all calculated channel keys", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base16", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc24",
					DataType:   telem.Int64T,
					Expression: "return base16",
				},
				{
					Name:       "calc25",
					DataType:   telem.Int64T,
					Expression: "return calc24 * 2",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			keys := g.CalculatedKeys()
			Expect(keys).To(HaveLen(2))
			Expect(keys.Contains(calcs[0].Key())).To(BeTrue())
			Expect(keys.Contains(calcs[1].Key())).To(BeTrue())
			Expect(keys.Contains(bases[0].Key())).To(BeFalse())
		})

		It("Should return empty set when no channels", func() {
			keys := g.CalculatedKeys()
			Expect(keys).To(BeEmpty())
		})

		It("Should update after channel removal", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base17", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "calc26",
				DataType:   telem.Int64T,
				Expression: "return base17",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())

			keys := g.CalculatedKeys()
			Expect(keys).To(HaveLen(1))
			Expect(keys.Contains(calc.Key())).To(BeTrue())

			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			keys = g.CalculatedKeys()
			Expect(keys).To(BeEmpty())
		})
	})

	Describe("Contains", func() {
		It("Should hold an added calculation and its dependencies", func(
			ctx SpecContext,
		) {
			base := channel.Channel{
				Name:     "contains_base",
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &base)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "contains_calc_a",
					DataType:   telem.Int64T,
					Expression: "return contains_base",
				},
				{
					Name:       "contains_calc_b",
					DataType:   telem.Int64T,
					Expression: "return contains_calc_a * 2",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Contains(calcs[0].Key())).To(BeTrue())
			Expect(g.Contains(calcs[1].Key())).To(BeTrue())
			Expect(g.Contains(base.Key())).To(BeFalse())
		})

		It("Should not hold a removed calculation", func(ctx SpecContext) {
			base := channel.Channel{
				Name:     "contains_removed_base",
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &base)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "contains_removed_calc",
				DataType:   telem.Int64T,
				Expression: "return contains_removed_base",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			Expect(g.Contains(calc.Key())).To(BeFalse())
		})
	})

	Describe("BaseKeys", func() {
		It("Should return the concrete channels of one calculation", func(
			ctx SpecContext,
		) {
			bases := []channel.Channel{
				{Name: "basekeys_a", DataType: telem.Int64T, Virtual: true},
				{Name: "basekeys_b", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "basekeys_calc_a",
					DataType:   telem.Int64T,
					Expression: "return basekeys_a",
				},
				{
					Name:       "basekeys_calc_b",
					DataType:   telem.Int64T,
					Expression: "return basekeys_b",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(MustSucceed(g.BaseKeys(calcs[0].Key())).Slice()).
				To(ConsistOf(bases[0].Key()))
			Expect(MustSucceed(g.BaseKeys(calcs[1].Key())).Slice()).
				To(ConsistOf(bases[1].Key()))
		})

		It("Should resolve the channels read through a nested calculation", func(
			ctx SpecContext,
		) {
			bases := []channel.Channel{
				{Name: "basekeys_nested", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "basekeys_inner",
				DataType:   telem.Int64T,
				Expression: "return basekeys_nested",
			}))
			outer := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "basekeys_outer",
				DataType:   telem.Int64T,
				Expression: "return basekeys_inner * 2",
			}))
			Expect(g.Add(ctx, outer)).To(Succeed())
			Expect(MustSucceed(g.BaseKeys(outer.Key())).Slice()).
				To(ConsistOf(bases[0].Key()))
		})

		It("Should follow an update to the expression", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "basekeys_old", DataType: telem.Int64T, Virtual: true},
				{Name: "basekeys_new", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "basekeys_updated",
				DataType:   telem.Int64T,
				Expression: "return basekeys_old",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			setExpression(ctx, &calc, "return basekeys_new")
			Expect(g.Update(ctx, calc)).To(Succeed())
			Expect(MustSucceed(g.BaseKeys(calc.Key())).Slice()).
				To(ConsistOf(bases[1].Key()))
		})

		It("Should return an error for a channel the graph does not hold", func() {
			Expect(g.BaseKeys(channel.Key(12345))).Error().
				To(MatchError(query.ErrNotFound))
		})
	})

	Describe("ConcreteBaseKeys", func() {
		It("Should return all concrete base channel keys", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base18", DataType: telem.Int64T, Virtual: true},
				{Name: "base19", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc27",
					DataType:   telem.Int64T,
					Expression: "return base18",
				},
				{
					Name:       "calc28",
					DataType:   telem.Int64T,
					Expression: "return base18 + base19",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
			Expect(baseKeys.Contains(calcs[0].Key())).To(BeFalse())
			Expect(baseKeys.Contains(calcs[1].Key())).To(BeFalse())
		})

		It("Should handle nested calculated channels", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base20", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc29",
					DataType:   telem.Int64T,
					Expression: "return base20",
				},
				{
					Name:       "calc30",
					DataType:   telem.Int64T,
					Expression: "return calc29 * 2",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(calcs[0].Key())).To(BeFalse())
			Expect(baseKeys.Contains(calcs[1].Key())).To(BeFalse())
		})

		It("Should return empty set when no channels", func() {
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(BeEmpty())
		})

		It("Should update after channel removal", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "base21", DataType: telem.Int64T, Virtual: true},
				{Name: "base22", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "calc31",
					DataType:   telem.Int64T,
					Expression: "return base21",
				},
				{
					Name:       "calc32",
					DataType:   telem.Int64T,
					Expression: "return base22",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))

			Expect(MustSucceed(g.Remove(calcs[0].Key()))).To(BeTrue())
			baseKeys = g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(1))
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[0].Key())).To(BeFalse())
		})
	})

	Describe("Update", func() {
		It(
			"Should update channel expression without changing dependencies",
			func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "upbase1", DataType: telem.Int64T, Virtual: true},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
				})).To(Succeed())
				calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "upcalc1",
					DataType:   telem.Int64T,
					Expression: "return upbase1 * 2",
				}))
				Expect(g.Add(ctx, calc)).To(Succeed())

				setExpression(ctx, &calc, "return upbase1 * 4")
				Expect(g.Update(ctx, calc)).To(Succeed())
				grouped := g.CalculateGrouped()
				Expect(grouped).To(HaveLen(1))
				baseKeys := g.ConcreteBaseKeys()
				Expect(baseKeys).To(HaveLen(1))
				Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			},
		)

		It("Should add new dependencies when updating", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase2", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase3", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "upcalc2",
				DataType:   telem.Int64T,
				Expression: "return upbase2",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())

			setExpression(ctx, &calc, "return upbase2 + upbase3")
			Expect(g.Update(ctx, calc)).To(Succeed())
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
		})

		It("Should remove old dependencies when updating", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase4", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase5", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "upcalc3",
				DataType:   telem.Int64T,
				Expression: "return upbase4 + upbase5",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())

			setExpression(ctx, &calc, "return upbase4")
			Expect(g.Update(ctx, calc)).To(Succeed())
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(1))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeFalse())
		})

		It("Should change group when base dependencies change", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase6", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase7", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "upcalc4",
					DataType:   telem.Int64T,
					Expression: "return upbase6",
				},
				{
					Name:       "upcalc5",
					DataType:   telem.Int64T,
					Expression: "return upbase7",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(2))

			setExpression(ctx, &calcs[0], "return upbase7")
			Expect(g.Update(ctx, calcs[0])).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})

		It("Should move the dependents of a channel that changes group", func(
			ctx SpecContext,
		) {
			bases := []channel.Channel{
				{Name: "dragbase1", DataType: telem.Int64T, Virtual: true},
				{Name: "dragbase2", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "dragsource",
					DataType:   telem.Int64T,
					Expression: "return dragbase1",
				},
				{
					Name:       "dragdependent",
					DataType:   telem.Int64T,
					Expression: "return dragsource * 2",
				},
				{
					Name:       "dragother",
					DataType:   telem.Int64T,
					Expression: "return dragbase2",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			Expect(g.CalculateGrouped()).To(HaveLen(2))
			setExpression(ctx, &calcs[0], "return dragbase2")
			Expect(g.Update(ctx, calcs[0])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			for _, mods := range grouped {
				Expect(mods).To(HaveLen(3))
			}
		})

		It("Should preserve reference counts after update", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase8", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "upcalc6",
				DataType:   telem.Int64T,
				Expression: "return upbase8",
			}))
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			setExpression(ctx, &calc, "return upbase8 * 3")
			Expect(g.Update(ctx, calc)).To(Succeed())
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(MustSucceed(g.Remove(calc.Key()))).To(BeTrue())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It(
			"Should handle updating with calculated dependencies",
			func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "upbase9", DataType: telem.Int64T, Virtual: true},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
				})).To(Succeed())
				calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
					{
						Name:       "upcalc7",
						DataType:   telem.Int64T,
						Expression: "return upbase9 * 2",
					},
					{
						Name:       "upcalc8",
						DataType:   telem.Int64T,
						Expression: "return upbase9",
					},
				}))
				Expect(g.Add(ctx, calcs[1])).To(Succeed())

				setExpression(ctx, &calcs[1], "return upcalc7 + 1")
				Expect(g.Update(ctx, calcs[1])).To(Succeed())
				flat := g.CalculateFlat()
				Expect(flat).To(HaveLen(2))
				Expect(flat[0].Channel.Name).To(Equal("upcalc7"))
				Expect(flat[1].Channel.Name).To(Equal("upcalc8"))
			},
		)

		It("Should clean up orphaned calculated dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase10", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "upcalc9",
					DataType:   telem.Int64T,
					Expression: "return upbase10 * 2",
				},
				{
					Name:       "upcalc10",
					DataType:   telem.Int64T,
					Expression: "return upcalc9 + 1",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			calcKeys := g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(2))

			setExpression(ctx, &calcs[1], "return upbase10")
			Expect(g.Update(ctx, calcs[1])).To(Succeed())
			calcKeys = g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(1))
			Expect(calcKeys.Contains(calcs[1].Key())).To(BeTrue())
			Expect(calcKeys.Contains(calcs[0].Key())).To(BeFalse())
		})

		It("Should fail to update non-existent channel", func(ctx SpecContext) {
			calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "nonexistent",
				DataType:   telem.Int64T,
				Expression: "return 1 + 1",
			}))
			Expect(g.Update(ctx, calc)).To(MatchError(query.ErrNotFound))
		})

		It("Should detect circular dependency during update", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase12", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "upcirc1",
					DataType:   telem.Int64T,
					Expression: "return upbase12",
				},
				{
					Name:       "upcirc2",
					DataType:   telem.Int64T,
					Expression: "return upcirc1",
				},
			}))
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			setExpression(ctx, &calcs[0], "return upcirc2")
			Expect(
				g.Update(ctx, calcs[0]),
			).To(MatchError(ContainSubstring("circular dependency")))
		})

		It("Should not remove shared calculated dependencies", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "upbase13", DataType: telem.Int64T, Virtual: true},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
			})).To(Succeed())
			calcs := MustSucceed(creator.CreateMany(ctx, []calctestutil.Calculation{
				{
					Name:       "upcalc11",
					DataType:   telem.Int64T,
					Expression: "return upbase13 * 2",
				},
				{
					Name:       "upcalc12",
					DataType:   telem.Int64T,
					Expression: "return upcalc11 + 1",
				},
				{
					Name:       "upcalc13",
					DataType:   telem.Int64T,
					Expression: "return upcalc11 * 3",
				},
			}))
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())

			calcKeys := g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(3))

			setExpression(ctx, &calcs[1], "return upbase13")
			Expect(g.Update(ctx, calcs[1])).To(Succeed())
			calcKeys = g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(3))
			Expect(calcKeys.Contains(calcs[0].Key())).To(BeTrue())
		})
		It(
			"Should recompile downstream calcs when upstream DataType changes",
			func(ctx SpecContext) {
				raw := channel.Channel{
					Name:     "cascade_raw",
					DataType: telem.Float32T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())

				calc1 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "cascade_calc1",
					DataType:   telem.Float32T,
					Expression: "return f32(cascade_raw * 1)",
				}))

				calc2 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "cascade_calc2",
					DataType:   telem.Float32T,
					Expression: "return cascade_calc1 * 2",
				}))

				Expect(g.Add(ctx, calc2)).To(Succeed())
				flat := g.CalculateFlat()
				Expect(flat).To(HaveLen(2))

				// Update calc1 to return f64 instead of f32
				setExpression(ctx, &calc1, "return f64(cascade_raw * 1.0)")
				// The update should trigger recompilation of calc2. Since calc2's
				// stored DataType is f32 but calc1 now provides f64, the expression
				// `cascade_calc1 * 2` resolves to f64 which doesn't match calc2's
				// f32 return type. The graph should surface this as an error.
				Expect(g.Update(ctx, calc1)).
					Error().To(MatchError(ContainSubstring("cannot return")))
			},
		)
		It(
			"Should cascade recompilation through a three-level chain",
			func(ctx SpecContext) {
				raw := channel.Channel{
					Name:     "chain3_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())

				calc1 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "chain3_c1",
					DataType:   telem.Float64T,
					Expression: "return chain3_raw + 1.0",
				}))

				MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "chain3_c2",
					DataType:   telem.Float64T,
					Expression: "return chain3_c1 + 2.0",
				}))

				calc3 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "chain3_c3",
					DataType:   telem.Float64T,
					Expression: "return chain3_c2 + 3.0",
				}))

				Expect(g.Add(ctx, calc3)).To(Succeed())
				Expect(g.CalculateFlat()).To(HaveLen(3))

				// Change calc1 to f32 output. This should cascade through
				// calc2 and calc3 and fail because calc2 declares f64 return
				// but calc1 now provides f32.
				setExpression(ctx, &calc1, "return f32(chain3_raw)")
				Expect(g.Update(ctx, calc1)).To(MatchError(ContainSubstring(
					"failed to recompile dependent channel",
				)))
			},
		)
		It(
			"Should cascade recompilation through a diamond dependency",
			func(ctx SpecContext) {
				raw := channel.Channel{
					Name:     "diamond_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())

				left := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "diamond_left",
					DataType:   telem.Float64T,
					Expression: "return diamond_raw * 2.0",
				}))

				right := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "diamond_right",
					DataType:   telem.Float64T,
					Expression: "return diamond_raw * 3.0",
				}))

				bottom := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "diamond_bottom",
					DataType:   telem.Float64T,
					Expression: "return diamond_left + diamond_right",
				}))

				Expect(g.Add(ctx, bottom)).To(Succeed())
				flat := g.CalculateFlat()
				Expect(flat).To(HaveLen(3))

				// Change left to f32 output. This should cascade to bottom
				// but not affect right, since right depends on raw, not left.
				setExpression(ctx, &left, "return f32(diamond_raw)")
				Expect(g.Update(ctx, left)).To(MatchError(ContainSubstring(
					"failed to recompile dependent channel",
				)))

				// right should still be in the graph unchanged
				Expect(g.CalculatedKeys().Contains(right.Key())).To(BeTrue())
			},
		)
		It(
			"Should leave the graph unchanged when a dependent fails to compile",
			func(ctx SpecContext) {
				var (
					failing  channel.Key
					compiled []channel.Key
				)
				g = MustSucceed(graph.New(graph.Config{
					Channel: channelSvc,
					DB:      db,
					Compile: func(
						ctx context.Context,
						db *gorp.DB,
						cfgs ...compiler.Config,
					) (compiler.Module, error) {
						compiled = append(compiled, cfgs[0].Channel.Key())
						if cfgs[0].Channel.Key() == failing {
							return compiler.Module{}, channel.ErrLookupsChanged
						}
						return compiler.Compile(ctx, db, cfgs...)
					},
				}))
				raw := channel.Channel{
					Name:     "rollback_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())
				parent := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "rollback_parent",
					DataType:   telem.Float64T,
					Expression: "return rollback_raw * 2.0",
				}))
				child := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "rollback_child",
					DataType:   telem.Float64T,
					Expression: "return f64(rollback_parent) * 2.0",
				}))
				Expect(g.Add(ctx, child)).To(Succeed())
				dataTypes := func() map[channel.Key]telem.DataType {
					return lo.SliceToMap(
						g.CalculateFlat(),
						func(m compiler.Module) (channel.Key, telem.DataType) {
							return m.Channel.Key(), m.Channel.DataType
						},
					)
				}
				before := dataTypes()

				setExpression(ctx, &parent, "return f32(rollback_raw)")
				failing = child.Key()
				Expect(g.Update(ctx, parent)).
					To(MatchError(channel.ErrLookupsChanged))
				Expect(dataTypes()).To(Equal(before))

				failing, compiled = 0, nil
				Expect(g.Update(ctx, parent)).To(Succeed())
				Expect(compiled).To(Equal([]channel.Key{parent.Key(), child.Key()}))
				Expect(dataTypes()[parent.Key()]).To(Equal(telem.Float32T))
			},
		)
		It(
			"Should change the DataType of a channel with no dependents",
			func(ctx SpecContext) {
				raw := channel.Channel{
					Name:     "lone_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())
				calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "lone_calc",
					DataType:   telem.Float64T,
					Expression: "return lone_raw * 2.0",
				}))
				Expect(g.Add(ctx, calc)).To(Succeed())
				setExpression(ctx, &calc, "return f32(lone_raw)")
				Expect(g.Update(ctx, calc)).To(Succeed())
				flat := g.CalculateFlat()
				Expect(flat).To(HaveLen(1))
				Expect(flat[0].Channel.DataType).To(Equal(telem.Float32T))
			},
		)
		It(
			"Should leave the graph unchanged when an added dependency fails to compile",
			func(ctx SpecContext) {
				var failing channel.Key
				g = MustSucceed(graph.New(graph.Config{
					Channel: channelSvc,
					DB:      db,
					Compile: func(
						ctx context.Context,
						db *gorp.DB,
						cfgs ...compiler.Config,
					) (compiler.Module, error) {
						if cfgs[0].Channel.Key() == failing {
							return compiler.Module{}, channel.ErrLookupsChanged
						}
						return compiler.Compile(ctx, db, cfgs...)
					},
				}))
				raw := channel.Channel{
					Name:     "atomic_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())
				deps := make([]channel.Channel, 2)
				for i, calc := range []calctestutil.Calculation{
					{
						Name:       "atomic_a",
						DataType:   telem.Float64T,
						Expression: "return atomic_raw * 2.0",
					},
					{
						Name:       "atomic_b",
						DataType:   telem.Float64T,
						Expression: "return atomic_raw * 3.0",
					},
				} {
					deps[i] = MustSucceed(creator.Create(ctx, calc))
				}
				sum := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "atomic_sum",
					DataType:   telem.Float64T,
					Expression: "return atomic_a + atomic_b",
				}))

				for _, dep := range deps {
					failing = dep.Key()
					Expect(g.Add(ctx, sum)).To(MatchError(channel.ErrLookupsChanged))
					Expect(g.CalculatedKeys()).To(BeEmpty())
				}
				failing = 0
				Expect(g.Add(ctx, sum)).To(Succeed())
				Expect(g.CalculatedKeys()).To(HaveLen(3))
				Expect(g.Remove(sum.Key())).To(BeTrue())
				Expect(g.CalculatedKeys()).To(BeEmpty())
			},
		)
		It("Should not cascade when DataType stays the same", func(ctx SpecContext) {
			raw := channel.Channel{
				Name:     "nocascade_raw",
				DataType: telem.Float64T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Create(ctx, &raw)
			})).To(Succeed())

			calc1 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "nocascade_c1",
				DataType:   telem.Float64T,
				Expression: "return nocascade_raw + 1.0",
			}))

			calc2 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
				Name:       "nocascade_c2",
				DataType:   telem.Float64T,
				Expression: "return nocascade_c1 + 2.0",
			}))

			Expect(g.Add(ctx, calc2)).To(Succeed())
			Expect(g.CalculateFlat()).To(HaveLen(2))

			// Change calc1's expression but keep the same DataType.
			// This should NOT trigger cascade recompilation.
			setExpression(ctx, &calc1, "return nocascade_raw + 99.0")
			Expect(g.Update(ctx, calc1)).To(Succeed())
			Expect(g.CalculateFlat()).To(HaveLen(2))
		})
		It(
			"Should rollback dependent modules on mid-chain compile failure",
			func(ctx SpecContext) {
				raw := channel.Channel{
					Name:     "rb_raw",
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).Create(ctx, &raw)
				})).To(Succeed())

				calc1 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "rb_c1",
					DataType:   telem.Float64T,
					Expression: "return rb_raw + 1.0",
				}))

				// calc2 uses an explicit cast, so it compiles regardless of
				// calc1's type. Its output is always f64.
				calc2 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "rb_c2",
					DataType:   telem.Float64T,
					Expression: "return f64(rb_c1)",
				}))

				// calc3 references both calc1 and calc2 without casting.
				// When calc1 is f64, this is f64+f64 which works. When calc1
				// changes to i64, this becomes i64+f64 which fails.
				calc3 := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "rb_c3",
					DataType:   telem.Float64T,
					Expression: "return rb_c1 + rb_c2",
				}))

				Expect(g.Add(ctx, calc3)).To(Succeed())
				flat := g.CalculateFlat()
				Expect(flat).To(HaveLen(3))

				// Save calc2's original WASM bytecode before the cascade.
				var origWASM []byte
				for _, mod := range flat {
					if mod.Channel.Key() == calc2.Key() {
						origWASM = make([]byte, len(mod.WASM))
						copy(origWASM, mod.WASM)
					}
				}
				Expect(origWASM).ToNot(BeEmpty())

				// Change calc1 from f64 to i64. This triggers cascade:
				// calc2 recompiles successfully (explicit cast handles i64),
				// but calc3 fails (i64 + f64 is a type mismatch).
				setExpression(ctx, &calc1, "return i64(rb_raw)")
				Expect(g.Update(ctx, calc1)).To(MatchError(ContainSubstring(
					"failed to recompile dependent channel",
				)))

				// calc2's module should be rolled back to its original state
				// since the cascade failed partway through.
				flat = g.CalculateFlat()
				for _, mod := range flat {
					if mod.Channel.Key() == calc2.Key() {
						Expect(mod.WASM).To(Equal(origWASM))
					}
				}
			},
		)
	})

	Describe("Error Messages", func() {
		It(
			"Should include channel name in Add compilation errors",
			func(ctx SpecContext) {
				calc := channel.Channel{
					Name:       "bad_calc_add",
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return invalid_syntax {{",
				}
				Expect(
					g.Add(ctx, calc),
				).Error().
					To(MatchError(ContainSubstring("bad_calc_add")))
			},
		)

		It(
			"Should include channel name in Update compilation errors",
			func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "err_base", DataType: telem.Int64T, Virtual: true},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return channelSvc.NewWriter(tx).CreateMany(ctx, &bases)
				})).To(Succeed())
				calc := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
					Name:       "bad_calc_update",
					DataType:   telem.Int64T,
					Expression: "return err_base * 2",
				}))
				Expect(g.Add(ctx, calc)).To(Succeed())
				calc.Expression = "return invalid_syntax {{"
				Expect(
					g.Update(ctx, calc),
				).Error().
					To(MatchError(ContainSubstring("bad_calc_update")))
			},
		)

		DescribeTable(
			"Should log a compile error that the caller does not retry",
			func(ctx SpecContext, compileErr error, logged int) {
				core, logs := observer.New(zapcore.ErrorLevel)
				logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
					ZapLogger: zap.New(core),
				}))
				g = MustSucceed(graph.New(graph.Config{
					Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
					Channel:         channelSvc,
					DB:              db,
					Compile: func(
						context.Context,
						*gorp.DB,
						...compiler.Config,
					) (compiler.Module, error) {
						return compiler.Module{}, compileErr
					},
				}))
				calc := channel.Channel{
					Name:       "logged_calc",
					DataType:   telem.Float64T,
					Virtual:    true,
					Expression: "return 1.0",
				}
				Expect(g.Add(ctx, calc)).To(MatchError(compileErr))
				failures := logs.FilterMessage("failed to compile calculated channel")
				Expect(failures.Len()).To(Equal(logged))
			},
			Entry("a compile failure", errors.New("bad expression"), 1),
			Entry("changed lookups", channel.ErrLookupsChanged, 0),
		)
	})
})
