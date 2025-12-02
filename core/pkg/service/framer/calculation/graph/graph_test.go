// Copyright 2025 Synnax Labs, Inc.
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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx    context.Context
	arcSvc *arc.Service
	dist   mock.Node
)

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
	labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Framer:   dist.Framer,
		Status:   statusSvc,
		Signals:  dist.Signals,
	}))
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})

var _ = Describe("Graph", func() {
	var g *graph.Graph
	BeforeEach(func() {
		g = MustSucceed(graph.New(graph.Config{
			Channel:        dist.Channel,
			SymbolResolver: arcSvc.SymbolResolver(),
		}))
	})
	Describe("Add", func() {
		It("Should compile and add a simple channel", func() {
			bases := []channel.Channel{{Name: "base1", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "calc1", DataType: telem.Int64T, Virtual: true, Expression: "return base1 * 2"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})

		It("Should handle nested calculated dependencies", func() {
			bases := []channel.Channel{{Name: "base2", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc1 := channel.Channel{Name: "calc2", DataType: telem.Int64T, Virtual: true, Expression: "return base2 + 1"}
			Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
			calc2 := channel.Channel{Name: "calc3", DataType: telem.Int64T, Virtual: true, Expression: "return calc2 * 2"}
			Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
			Expect(g.Add(ctx, calc2)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(grouped[0]).To(HaveLen(2))
		})

		It("Should detect circular dependencies", func() {
			calc1 := channel.Channel{Name: "circ1", DataType: telem.Int64T, Virtual: true, Expression: "return circ2"}
			Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
			calc2 := channel.Channel{Name: "circ2", DataType: telem.Int64T, Virtual: true, Expression: "return circ1"}
			Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
			Expect(g.Add(ctx, calc1)).To(MatchError(ContainSubstring("circular dependency")))
		})

		It("Should not re-add existing channel", func() {
			bases := []channel.Channel{{Name: "base3", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "calc4", DataType: telem.Int64T, Virtual: true, Expression: "return base3"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
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
		It("Should group channels by base dependencies", func() {
			bases := []channel.Channel{
				{Name: "base4", DataType: telem.Int64T, Virtual: true},
				{Name: "base5", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc5", DataType: telem.Int64T, Virtual: true, Expression: "return base4 + 1"},
				{Name: "calc6", DataType: telem.Int64T, Virtual: true, Expression: "return base4 * 2"},
				{Name: "calc7", DataType: telem.Int64T, Virtual: true, Expression: "return base5 - 1"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(2))
		})

		It("Should reuse groups with superset dependencies", func() {
			bases := []channel.Channel{
				{Name: "base6", DataType: telem.Int64T, Virtual: true},
				{Name: "base7", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc8", DataType: telem.Int64T, Virtual: true, Expression: "return base6"},
				{Name: "calc9", DataType: telem.Int64T, Virtual: true, Expression: "return base6 + base7"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})
	})

	Describe("CalculateFlat", func() {
		It("Should return all modules in topological order", func() {
			bases := []channel.Channel{{Name: "flatbase1", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "flat1", DataType: telem.Int64T, Virtual: true, Expression: "return flatbase1"},
				{Name: "flat2", DataType: telem.Int64T, Virtual: true, Expression: "return flat1 * 2"},
				{Name: "flat3", DataType: telem.Int64T, Virtual: true, Expression: "return flat2 + 1"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
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
		It("Should remove a channel from allocator", func() {
			bases := []channel.Channel{{Name: "base8", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "calc10", DataType: telem.Int64T, Virtual: true, Expression: "return base8"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Remove(calc.Key())).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It("Should clean up empty groups", func() {
			bases := []channel.Channel{{Name: "base9", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc11", DataType: telem.Int64T, Virtual: true, Expression: "return base9"},
				{Name: "calc12", DataType: telem.Int64T, Virtual: true, Expression: "return base9 * 2"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Remove(calcs[0].Key())).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(g.Remove(calcs[1].Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It("Should not fail to remove non-existent channel", func() {
			Expect(g.Remove(channel.Key(99999))).To(Succeed())
		})
	})

	Describe("Reference Counting", func() {
		It("Should increment explicit count on multiple adds", func() {
			bases := []channel.Channel{{Name: "base12", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "calc16", DataType: telem.Int64T, Virtual: true, Expression: "return base12"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Remove(calc.Key())).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(g.Remove(calc.Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
		It("Should cascade remove dependencies when parent removed", func() {
			bases := []channel.Channel{{Name: "base13", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc17", DataType: telem.Int64T, Virtual: true, Expression: "return base13"},
				{Name: "calc18", DataType: telem.Int64T, Virtual: true, Expression: "return calc17 * 2"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped[0]).To(HaveLen(2))
			Expect(g.Remove(calcs[1].Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
		It("Should not remove dep if still referenced by another channel", func() {
			bases := []channel.Channel{{Name: "base14", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc19", DataType: telem.Int64T, Virtual: true, Expression: "return base14"},
				{Name: "calc20", DataType: telem.Int64T, Virtual: true, Expression: "return calc19 + 1"},
				{Name: "calc21", DataType: telem.Int64T, Virtual: true, Expression: "return calc19 * 2"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped[0]).To(HaveLen(3))
			Expect(g.Remove(calcs[1].Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped[0]).To(HaveLen(2))
			Expect(g.Remove(calcs[2].Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
		It("Should handle explicit request on dependency", func() {
			bases := []channel.Channel{{Name: "base15", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc22", DataType: telem.Int64T, Virtual: true, Expression: "return base15"},
				{Name: "calc23", DataType: telem.Int64T, Virtual: true, Expression: "return calc22 + 1"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Remove(calcs[1].Key())).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped[0]).To(HaveLen(1))
			Expect(g.Remove(calcs[0].Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})
	})

	Describe("CalculatedKeys", func() {
		It("Should return all calculated channel keys", func() {
			bases := []channel.Channel{{Name: "base16", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc24", DataType: telem.Int64T, Virtual: true, Expression: "return base16"},
				{Name: "calc25", DataType: telem.Int64T, Virtual: true, Expression: "return calc24 * 2"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
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

		It("Should update after channel removal", func() {
			bases := []channel.Channel{{Name: "base17", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "calc26", DataType: telem.Int64T, Virtual: true, Expression: "return base17"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			keys := g.CalculatedKeys()
			Expect(keys).To(HaveLen(1))
			Expect(keys.Contains(calc.Key())).To(BeTrue())

			Expect(g.Remove(calc.Key())).To(Succeed())
			keys = g.CalculatedKeys()
			Expect(keys).To(BeEmpty())
		})
	})

	Describe("ConcreteBaseKeys", func() {
		It("Should return all concrete base channel keys", func() {
			bases := []channel.Channel{
				{Name: "base18", DataType: telem.Int64T, Virtual: true},
				{Name: "base19", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc27", DataType: telem.Int64T, Virtual: true, Expression: "return base18"},
				{Name: "calc28", DataType: telem.Int64T, Virtual: true, Expression: "return base18 + base19"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
			Expect(baseKeys.Contains(calcs[0].Key())).To(BeFalse())
			Expect(baseKeys.Contains(calcs[1].Key())).To(BeFalse())
		})

		It("Should handle nested calculated channels", func() {
			bases := []channel.Channel{{Name: "base20", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc29", DataType: telem.Int64T, Virtual: true, Expression: "return base20"},
				{Name: "calc30", DataType: telem.Int64T, Virtual: true, Expression: "return calc29 * 2"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
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

		It("Should update after channel removal", func() {
			bases := []channel.Channel{
				{Name: "base21", DataType: telem.Int64T, Virtual: true},
				{Name: "base22", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "calc31", DataType: telem.Int64T, Virtual: true, Expression: "return base21"},
				{Name: "calc32", DataType: telem.Int64T, Virtual: true, Expression: "return base22"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))

			Expect(g.Remove(calcs[0].Key())).To(Succeed())
			baseKeys = g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(1))
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[0].Key())).To(BeFalse())
		})
	})

	Describe("Update", func() {
		It("Should update channel expression without changing dependencies", func() {
			bases := []channel.Channel{{Name: "upbase1", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "upcalc1", DataType: telem.Int64T, Virtual: true, Expression: "return upbase1 * 2"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			calc.Expression = "return upbase1 * 4"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Update(ctx, calc)).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(1))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
		})

		It("Should add new dependencies when updating", func() {
			bases := []channel.Channel{
				{Name: "upbase2", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase3", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "upcalc2", DataType: telem.Int64T, Virtual: true, Expression: "return upbase2"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			calc.Expression = "return upbase2 + upbase3"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Update(ctx, calc)).To(Succeed())
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(2))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeTrue())
		})

		It("Should remove old dependencies when updating", func() {
			bases := []channel.Channel{
				{Name: "upbase4", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase5", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "upcalc3", DataType: telem.Int64T, Virtual: true, Expression: "return upbase4 + upbase5"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			calc.Expression = "return upbase4"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Update(ctx, calc)).To(Succeed())
			baseKeys := g.ConcreteBaseKeys()
			Expect(baseKeys).To(HaveLen(1))
			Expect(baseKeys.Contains(bases[0].Key())).To(BeTrue())
			Expect(baseKeys.Contains(bases[1].Key())).To(BeFalse())
		})

		It("Should change group when base dependencies change", func() {
			bases := []channel.Channel{
				{Name: "upbase6", DataType: telem.Int64T, Virtual: true},
				{Name: "upbase7", DataType: telem.Int64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "upcalc4", DataType: telem.Int64T, Virtual: true, Expression: "return upbase6"},
				{Name: "upcalc5", DataType: telem.Int64T, Virtual: true, Expression: "return upbase7"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(2))

			calcs[0].Expression = "return upbase7"
			Expect(dist.Channel.Create(ctx, &calcs[0])).To(Succeed())
			Expect(g.Update(ctx, calcs[0])).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
		})

		It("Should preserve reference counts after update", func() {
			bases := []channel.Channel{{Name: "upbase8", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calc := channel.Channel{Name: "upcalc6", DataType: telem.Int64T, Virtual: true, Expression: "return upbase8"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())
			Expect(g.Add(ctx, calc)).To(Succeed())

			calc.Expression = "return upbase8 * 3"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Update(ctx, calc)).To(Succeed())
			Expect(g.Remove(calc.Key())).To(Succeed())
			grouped := g.CalculateGrouped()
			Expect(grouped).To(HaveLen(1))
			Expect(g.Remove(calc.Key())).To(Succeed())
			grouped = g.CalculateGrouped()
			Expect(grouped).To(BeEmpty())
		})

		It("Should handle updating with calculated dependencies", func() {
			bases := []channel.Channel{{Name: "upbase9", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "upcalc7", DataType: telem.Int64T, Virtual: true, Expression: "return upbase9 * 2"},
				{Name: "upcalc8", DataType: telem.Int64T, Virtual: true, Expression: "return upbase9"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			calcs[1].Expression = "return upcalc7 + 1"
			Expect(dist.Channel.Create(ctx, &calcs[1])).To(Succeed())
			Expect(g.Update(ctx, calcs[1])).To(Succeed())
			flat := g.CalculateFlat()
			Expect(flat).To(HaveLen(2))
			Expect(flat[0].Channel.Name).To(Equal("upcalc7"))
			Expect(flat[1].Channel.Name).To(Equal("upcalc8"))
		})

		It("Should clean up orphaned calculated dependencies", func() {
			bases := []channel.Channel{{Name: "upbase10", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "upcalc9", DataType: telem.Int64T, Virtual: true, Expression: "return upbase10 * 2"},
				{Name: "upcalc10", DataType: telem.Int64T, Virtual: true, Expression: "return upcalc9 + 1"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			calcKeys := g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(2))

			calcs[1].Expression = "return upbase10"
			Expect(dist.Channel.Create(ctx, &calcs[1])).To(Succeed())
			Expect(g.Update(ctx, calcs[1])).To(Succeed())
			calcKeys = g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(1))
			Expect(calcKeys.Contains(calcs[1].Key())).To(BeTrue())
			Expect(calcKeys.Contains(calcs[0].Key())).To(BeFalse())
		})

		It("Should fail to update non-existent channel", func() {
			calc := channel.Channel{Name: "nonexistent", DataType: telem.Int64T, Virtual: true, Expression: "return 1 + 1"}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			Expect(g.Update(ctx, calc)).To(HaveOccurred())
		})

		It("Should detect circular dependency during update", func() {
			bases := []channel.Channel{{Name: "upbase12", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "upcirc1", DataType: telem.Int64T, Virtual: true, Expression: "return upbase12"},
				{Name: "upcirc2", DataType: telem.Int64T, Virtual: true, Expression: "return upcirc1"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[0])).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())

			calcs[0].Expression = "return upcirc2"
			Expect(dist.Channel.Create(ctx, &calcs[0])).To(Succeed())
			Expect(g.Update(ctx, calcs[0])).To(MatchError(ContainSubstring("circular dependency")))
		})

		It("Should not remove shared calculated dependencies", func() {
			bases := []channel.Channel{{Name: "upbase13", DataType: telem.Int64T, Virtual: true}}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			calcs := []channel.Channel{
				{Name: "upcalc11", DataType: telem.Int64T, Virtual: true, Expression: "return upbase13 * 2"},
				{Name: "upcalc12", DataType: telem.Int64T, Virtual: true, Expression: "return upcalc11 + 1"},
				{Name: "upcalc13", DataType: telem.Int64T, Virtual: true, Expression: "return upcalc11 * 3"},
			}
			Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
			Expect(g.Add(ctx, calcs[1])).To(Succeed())
			Expect(g.Add(ctx, calcs[2])).To(Succeed())

			calcKeys := g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(3))

			calcs[1].Expression = "return upbase13"
			Expect(dist.Channel.Create(ctx, &calcs[1])).To(Succeed())
			Expect(g.Update(ctx, calcs[1])).To(Succeed())
			calcKeys = g.CalculatedKeys()
			Expect(calcKeys).To(HaveLen(3))
			Expect(calcKeys.Contains(calcs[0].Key())).To(BeTrue())
		})
	})
})
