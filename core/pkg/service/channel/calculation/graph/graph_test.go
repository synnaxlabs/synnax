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
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	graph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx       context.Context
	dist      mock.Node
	statusSvc *status.Service
)

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
	labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	DeferCleanup(func() { Expect(labelSvc.Close()).To(Succeed()) })
	statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Group:    dist.Group,
		Signals:  dist.Signals,
		Ontology: dist.Ontology,
		Label:    labelSvc,
	}))
	DeferCleanup(func() { Expect(statusSvc.Close()).To(Succeed()) })
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})

func openGraph() *graph.Graph {
	g := MustSucceed(graph.Open(ctx, graph.Config{
		Channel: dist.Channel,
		Status:  statusSvc,
	}))
	DeferCleanup(func() { Expect(g.Close()).To(Succeed()) })
	return g
}

func fetchStatus(key channel.Key) (status.Status[graph.StatusDetails], bool) {
	var statuses []status.Status[graph.StatusDetails]
	err := status.NewRetrieve[graph.StatusDetails](statusSvc).
		WhereKeys(channel.OntologyID(key).String()).
		Entries(&statuses).
		Exec(ctx, nil)
	if err != nil || len(statuses) == 0 {
		return status.Status[graph.StatusDetails]{}, false
	}
	return statuses[0], true
}

func expectStatus(key channel.Key) status.Status[graph.StatusDetails] {
	var result status.Status[graph.StatusDetails]
	Eventually(func() bool {
		s, ok := fetchStatus(key)
		if ok && s.Variant == xstatus.VariantError {
			result = s
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
		"expected error status for channel %d", key)
	return result
}

func expectNoStatus(key channel.Key) {
	_, ok := fetchStatus(key)
	Expect(ok).To(BeFalse(), "expected no status for channel %d", key)
}

func eventuallyExpectNoStatus(key channel.Key) {
	Eventually(func() bool {
		_, ok := fetchStatus(key)
		return !ok
	}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
		"expected status to be cleared for channel %d", key)
}

func retrieveChannelDataType(key channel.Key) telem.DataType {
	var ch channel.Channel
	Expect(dist.Channel.NewRetrieve().WhereKeys(key).Entry(&ch).Exec(ctx, nil)).To(Succeed())
	return ch.DataType
}

var _ = Describe("Graph", func() {

	Describe("Open / Hydration", func() {

		It("Should open successfully with no calculated channels", func() {
			bases := []channel.Channel{
				{Name: "hy_base1", DataType: telem.Int64T, Virtual: true},
				{Name: "hy_base2", DataType: telem.Float64T, Virtual: true},
			}
			Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
			openGraph()
		})

		It("Should open with a valid calculated channel and set no status", func() {
			base := channel.Channel{Name: "hy_valid_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "hy_valid_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_valid_base * 2",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()
			expectNoStatus(calc.Key())
		})

		It("Should set error status for a syntax error in expression", func() {
			calc := channel.Channel{
				Name: "hy_syntax_err", DataType: telem.Int64T, Virtual: true,
				Expression: "return {{invalid syntax",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()
			expectStatus(calc.Key())
		})

		It("Should set error status for an unresolvable reference", func() {
			calc := channel.Channel{
				Name: "hy_unresolvable", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_nonexistent_xyz * 2",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()
			expectStatus(calc.Key())
		})

		It("Should handle a mix of valid and invalid calculated channels", func() {
			base := channel.Channel{Name: "hy_mix_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calcOk := channel.Channel{
				Name: "hy_mix_ok", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_mix_base + 1",
			}
			Expect(dist.Channel.Create(ctx, &calcOk)).To(Succeed())
			calcBad := channel.Channel{
				Name: "hy_mix_bad", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_no_such_channel",
			}
			Expect(dist.Channel.Create(ctx, &calcBad)).To(Succeed())
			openGraph()
			expectNoStatus(calcOk.Key())
			expectStatus(calcBad.Key())
		})

		It("Should handle an orphan calculated channel with no dependencies", func() {
			calc := channel.Channel{
				Name: "hy_orphan", DataType: telem.Int64T, Virtual: true,
				Expression: "return 42",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()
			expectNoStatus(calc.Key())
		})

		Context("Dependency Topologies", func() {
			It("Should hydrate a diamond dependency graph", func() {
				base := channel.Channel{Name: "hy_dia_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calcB := channel.Channel{
					Name: "hy_dia_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calcB)).To(Succeed())
				calcC := channel.Channel{
					Name: "hy_dia_c", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_base * 2",
				}
				Expect(dist.Channel.Create(ctx, &calcC)).To(Succeed())
				calcA := channel.Channel{
					Name: "hy_dia_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_b + hy_dia_c",
				}
				Expect(dist.Channel.Create(ctx, &calcA)).To(Succeed())
				openGraph()
				expectNoStatus(calcA.Key())
				expectNoStatus(calcB.Key())
				expectNoStatus(calcC.Key())
			})

			It("Should hydrate a deep chain (4 levels)", func() {
				base := channel.Channel{Name: "hy_deep_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "hy_deep_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "hy_deep_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "hy_deep_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c2 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "hy_deep_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c3 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c4)).To(Succeed())
				openGraph()
				expectNoStatus(c1.Key())
				expectNoStatus(c2.Key())
				expectNoStatus(c3.Key())
				expectNoStatus(c4.Key())
			})

			It("Should hydrate a fan-out topology", func() {
				base := channel.Channel{Name: "hy_fan_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "hy_fan_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_fan_base + 1",
				}
				c2 := channel.Channel{
					Name: "hy_fan_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_fan_base * 2",
				}
				c3 := channel.Channel{
					Name: "hy_fan_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_fan_base - 1",
				}
				calcs := []channel.Channel{c1, c2, c3}
				Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
				openGraph()
				expectNoStatus(calcs[0].Key())
				expectNoStatus(calcs[1].Key())
				expectNoStatus(calcs[2].Key())
			})

			It("Should hydrate a fan-in topology", func() {
				bases := []channel.Channel{
					{Name: "hy_fin_b1", DataType: telem.Int64T, Virtual: true},
					{Name: "hy_fin_b2", DataType: telem.Int64T, Virtual: true},
					{Name: "hy_fin_b3", DataType: telem.Int64T, Virtual: true},
				}
				Expect(dist.Channel.CreateMany(ctx, &bases)).To(Succeed())
				calc := channel.Channel{
					Name: "hy_fin_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_fin_b1 + hy_fin_b2 + hy_fin_b3",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				openGraph()
				expectNoStatus(calc.Key())
			})
		})

		Context("DataType Repair", func() {
			It("Should not repair when DataType already matches", func() {
				base := channel.Channel{Name: "hy_norep_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "hy_norep_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_norep_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				openGraph()
				Expect(retrieveChannelDataType(calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair a stale DataType during hydration", func() {
				base := channel.Channel{Name: "hy_rep_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name:       "hy_rep_calc",
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: "return hy_rep_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				Expect(retrieveChannelDataType(calc.Key())).To(Equal(telem.Float32T))
				openGraph()
				Expect(retrieveChannelDataType(calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair cascaded DataType when a dependent has a lower key than its dependency", func() {
				base := channel.Channel{Name: "hy_ooo_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())

				calc2 := channel.Channel{
					Name:       "hy_ooo_c2",
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: "return hy_ooo_c1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())

				calc1 := channel.Channel{
					Name:       "hy_ooo_c1",
					DataType:   telem.Float32T,
					Virtual:    true,
					Expression: "return hy_ooo_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())

				Expect(calc2.Key()).To(BeNumerically("<", calc1.Key()))
				Expect(retrieveChannelDataType(calc1.Key())).To(Equal(telem.Float32T))
				Expect(retrieveChannelDataType(calc2.Key())).To(Equal(telem.Float32T))

				openGraph()

				Expect(retrieveChannelDataType(calc1.Key())).To(Equal(telem.Int64T))
				Expect(retrieveChannelDataType(calc2.Key())).To(Equal(telem.Int64T))
			})
		})
	})

	Describe("Reactive Change Handling", func() {

		Context("Creating Channels", func() {
			It("Should inspect a new valid calculated channel", func() {
				openGraph()
				base := channel.Channel{Name: "rc_create_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_create_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_create_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())
			})

			It("Should set error status for a new invalid calculated channel", func() {
				openGraph()
				calc := channel.Channel{
					Name: "rc_create_bad", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_nonexistent_abc",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectStatus(calc.Key())
			})

			It("Should handle incrementally building a chain after graph open", func() {
				openGraph()
				base := channel.Channel{Name: "rc_chain_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_chain_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				expectNoStatus(calc1.Key())
				calc2 := channel.Channel{
					Name: "rc_chain_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_c1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
				expectNoStatus(calc2.Key())
			})

			It("Should process a batch CreateMany in a single handleChanges call", func() {
				openGraph()
				base := channel.Channel{Name: "rc_batch_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calcs := []channel.Channel{
					{Name: "rc_batch_c1", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base + 1"},
					{Name: "rc_batch_c2", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base * 2"},
					{Name: "rc_batch_c3", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base - 1"},
				}
				Expect(dist.Channel.CreateMany(ctx, &calcs)).To(Succeed())
				expectNoStatus(calcs[0].Key())
				expectNoStatus(calcs[1].Key())
				expectNoStatus(calcs[2].Key())
			})
		})

		Context("Deleting Channels", func() {
			It("Should set error status when a base dependency is deleted", func() {
				openGraph()
				base := channel.Channel{Name: "rc_del_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_del_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Deleting the base dependency")
				Expect(dist.Channel.Delete(ctx, base.Key(), false)).To(Succeed())
				expectStatus(calc.Key())
			})

			It("Should set error on downstream calc when intermediate calc is deleted", func() {
				openGraph()
				base := channel.Channel{Name: "rc_del_mid_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_del_mid_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_mid_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_del_mid_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_mid_c1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
				expectNoStatus(calc1.Key())
				expectNoStatus(calc2.Key())

				By("Deleting the intermediate calculated channel")
				Expect(dist.Channel.Delete(ctx, calc1.Key(), false)).To(Succeed())
				expectStatus(calc2.Key())
			})

			It("Should leave upstream unaffected when a leaf calc is deleted", func() {
				openGraph()
				base := channel.Channel{Name: "rc_del_leaf_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_del_leaf_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_leaf_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_del_leaf_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_leaf_c1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())

				By("Deleting the leaf calc")
				Expect(dist.Channel.Delete(ctx, calc2.Key(), false)).To(Succeed())
				expectNoStatus(calc1.Key())
			})

			It("Should not cascade invalidity through reconcileQueued in a diamond", func() {
				openGraph()
				base := channel.Channel{Name: "rc_del_dia_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calcB := channel.Channel{
					Name: "rc_del_dia_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calcB)).To(Succeed())
				calcC := channel.Channel{
					Name: "rc_del_dia_c", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_base * 2",
				}
				Expect(dist.Channel.Create(ctx, &calcC)).To(Succeed())
				calcA := channel.Channel{
					Name: "rc_del_dia_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_b + rc_del_dia_c",
				}
				Expect(dist.Channel.Create(ctx, &calcA)).To(Succeed())
				expectNoStatus(calcA.Key())

				By("Deleting the shared base dependency")
				Expect(dist.Channel.Delete(ctx, base.Key(), false)).To(Succeed())

				By("Verifying calc_b and calc_c get error statuses")
				expectStatus(calcB.Key())
				expectStatus(calcC.Key())

				By("Verifying calc_a does NOT get error status because " +
					"reconcileQueued continues without enqueueing dependents on error")
				expectNoStatus(calcA.Key())
			})
		})

		Context("Updating Channels", func() {
			It("Should update deps when expression changes to use a different base", func() {
				openGraph()
				base1 := channel.Channel{Name: "rc_upd_b1", DataType: telem.Int64T, Virtual: true}
				base2 := channel.Channel{Name: "rc_upd_b2", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base1)).To(Succeed())
				Expect(dist.Channel.Create(ctx, &base2)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_upd_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_b1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Updating expression to use a different base")
				calc.Expression = "return rc_upd_b2 * 2"
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Verifying old base deletion does not affect calc")
				Expect(dist.Channel.Delete(ctx, base1.Key(), false)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Verifying new base deletion does affect calc")
				Expect(dist.Channel.Delete(ctx, base2.Key(), false)).To(Succeed())
				expectStatus(calc.Key())
			})

			It("Should set error status when expression is updated to invalid", func() {
				openGraph()
				base := channel.Channel{Name: "rc_upd_bad_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_upd_bad_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_bad_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Updating to an invalid expression")
				calc.Expression = "return rc_no_such_thing"
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectStatus(calc.Key())
			})

			It("Should clear error status when a broken expression is fixed", func() {
				openGraph()
				calc := channel.Channel{
					Name: "rc_upd_fix", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_fix_missing_dep",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectStatus(calc.Key())

				By("Creating the missing dependency and fixing the expression")
				dep := channel.Channel{Name: "rc_fix_missing_dep", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &dep)).To(Succeed())
				calc.Expression = "return rc_fix_missing_dep + 1"
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(calc.Key())
			})
		})

		Context("Cascading Reconciliation", func() {
			It("Should not cascade invalidity from reconcileQueued to further dependents", func() {
				openGraph()
				base := channel.Channel{Name: "rc_cas_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_cas_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_cas_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_cas_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_cas_c1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
				expectNoStatus(calc1.Key())
				expectNoStatus(calc2.Key())

				By("Deleting the base. calc1 becomes invalid. " +
					"calc2 should NOT get error because reconcileQueued " +
					"does not enqueue dependents when a node errors")
				Expect(dist.Channel.Delete(ctx, base.Key(), false)).To(Succeed())
				expectStatus(calc1.Key())
				expectNoStatus(calc2.Key())
			})

			It("Should cascade deletion through a long chain", func() {
				openGraph()
				base := channel.Channel{Name: "rc_long_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "rc_long_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "rc_long_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "rc_long_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c2 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "rc_long_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c3 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c4)).To(Succeed())

				By("Deleting c2 from the middle of the chain")
				Expect(dist.Channel.Delete(ctx, c2.Key(), false)).To(Succeed())

				By("c1 is upstream and unaffected")
				expectNoStatus(c1.Key())

				By("c3 depends on c2 which is gone, so it gets error")
				expectStatus(c3.Key())

				By("c4 does not get error because reconcileQueued does not " +
					"cascade invalidity from c3's failure")
				expectNoStatus(c4.Key())
			})

			It("Should re-inspect dependents when a calculated channel is updated", func() {
				openGraph()
				base := channel.Channel{Name: "rc_reins_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_reins_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_reins_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_reins_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_reins_c1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
				expectNoStatus(calc1.Key())
				expectNoStatus(calc2.Key())

				By("Updating calc1 expression - calc2 should be re-inspected via BFS")
				calc1.Expression = "return rc_reins_base + 100"
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				expectNoStatus(calc1.Key())
				expectNoStatus(calc2.Key())
			})
		})

		Context("DataType Persistence", func() {
			It("Should persist DataType changes to the DB when a dependency type changes", func() {
				openGraph()

				By("Creating a base channel and a calc that depends on it")
				base := channel.Channel{Name: "rc_dtp_base", DataType: telem.Float32T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_dtp_calc", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtp_base * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Updating the calc expression to return a different type")
				calc.Expression = "return i64(rc_dtp_base)"
				calc.DataType = telem.Int64T
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectNoStatus(calc.Key())

				By("Verifying the DataType was persisted to the DB")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(calc.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})

			It("Should persist cascaded DataType changes through a chain", func() {
				openGraph()

				By("Building a chain: base -> calc1 -> calc2")
				base := channel.Channel{Name: "rc_dtpc_base", DataType: telem.Float32T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_dtpc_c1", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtpc_base * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_dtpc_c2", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtpc_c1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())

				By("Updating calc1 to return a different type, which should cascade to calc2")
				calc1.Expression = "return i64(rc_dtpc_base)"
				calc1.DataType = telem.Int64T
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())

				By("Verifying calc1 DataType was persisted")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(calc1.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))

				By("Verifying calc2 DataType was also updated via cascade")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(calc2.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})
		})

		Context("Unresolved Name Auto-Heal", func() {
			It("Should auto-fix a broken calc when the missing dependency is created", func() {
				openGraph()
				calc := channel.Channel{
					Name: "rc_unres_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_missing * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
				expectStatus(calc.Key())

				By("Creating the previously missing dependency")
				dep := channel.Channel{Name: "rc_unres_missing", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &dep)).To(Succeed())

				By("Verifying calc is auto-fixed")
				eventuallyExpectNoStatus(calc.Key())
			})

			It("Should auto-fix multiple calcs waiting on the same missing name", func() {
				openGraph()
				calc1 := channel.Channel{
					Name: "rc_unres_multi_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_shared_dep + 1",
				}
				calc2 := channel.Channel{
					Name: "rc_unres_multi_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_shared_dep * 2",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				Expect(dist.Channel.Create(ctx, &calc2)).To(Succeed())
				expectStatus(calc1.Key())
				expectStatus(calc2.Key())

				By("Creating the shared missing dependency")
				dep := channel.Channel{Name: "rc_unres_shared_dep", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &dep)).To(Succeed())

				By("Both calcs should auto-heal")
				eventuallyExpectNoStatus(calc1.Key())
				eventuallyExpectNoStatus(calc2.Key())
			})

			It("Should auto-fix a chain where a missing base is created", func() {
				openGraph()
				calc1 := channel.Channel{
					Name: "rc_unres_chain_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_chain_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &calc1)).To(Succeed())
				expectStatus(calc1.Key())

				By("Creating the missing base")
				base := channel.Channel{Name: "rc_unres_chain_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())

				By("calc1 should auto-heal")
				eventuallyExpectNoStatus(calc1.Key())
			})
		})

		Context("Multiple Independent Subgraphs", func() {
			It("Should isolate failures to their own subgraph", func() {
				openGraph()
				baseA := channel.Channel{Name: "rc_iso_base_a", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &baseA)).To(Succeed())
				calcA := channel.Channel{
					Name: "rc_iso_calc_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_iso_base_a + 1",
				}
				Expect(dist.Channel.Create(ctx, &calcA)).To(Succeed())

				baseB := channel.Channel{Name: "rc_iso_base_b", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &baseB)).To(Succeed())
				calcB := channel.Channel{
					Name: "rc_iso_calc_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_iso_base_b + 1",
				}
				Expect(dist.Channel.Create(ctx, &calcB)).To(Succeed())

				By("Deleting base_a should only affect calc_a")
				Expect(dist.Channel.Delete(ctx, baseA.Key(), false)).To(Succeed())
				expectStatus(calcA.Key())
				expectNoStatus(calcB.Key())
			})
		})
	})

	Describe("Status Communication", func() {
		It("Should set status with correct structure and details", func() {
			calc := channel.Channel{
				Name: "st_detail", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_missing_detail_dep",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()

			s := expectStatus(calc.Key())
			Expect(s.Variant).To(Equal(xstatus.VariantError))
			Expect(s.Message).To(Equal("invalid expression for st_detail"))
			Expect(s.Description).ToNot(BeEmpty())
			Expect(s.Details.Channel).To(Equal(calc.Key()))
			Expect(s.Key).To(Equal(channel.OntologyID(calc.Key()).String()))
			Expect(s.Name).To(Equal("st_detail"))
		})

		It("Should clear status when a broken expression is fixed", func() {
			openGraph()
			calc := channel.Channel{
				Name: "st_clear", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_clear_missing",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			expectStatus(calc.Key())

			By("Fixing the expression")
			base := channel.Channel{Name: "st_clear_dep", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc.Expression = "return st_clear_dep + 1"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			eventuallyExpectNoStatus(calc.Key())
		})

		It("Should overwrite status when expression changes to a different error", func() {
			openGraph()
			calc := channel.Channel{
				Name: "st_overwrite", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_missing_a",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			s1 := expectStatus(calc.Key())

			By("Updating to a different broken expression")
			calc.Expression = "return {{syntax error"
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			var s2 status.Status[graph.StatusDetails]
			Eventually(func() bool {
				s, ok := fetchStatus(calc.Key())
				if ok && s.Description != s1.Description {
					s2 = s
					return true
				}
				return false
			}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
				"expected status description to change")
			Expect(s2.Variant).To(Equal(xstatus.VariantError))
		})

		It("Should not create any status entry for valid channels", func() {
			base := channel.Channel{Name: "st_none_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "st_none_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_none_base + 1",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			openGraph()
			expectNoStatus(calc.Key())
		})
	})

	Describe("Lifecycle", func() {
		It("Should open and close without error", func() {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				Channel: dist.Channel,
				Status:  statusSvc,
			}))
			Expect(g.Close()).To(Succeed())
		})

		It("Should disconnect observer on Close", func() {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				Channel: dist.Channel,
				Status:  statusSvc,
			}))
			base := channel.Channel{Name: "lc_disc_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "lc_disc_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return lc_disc_base + 1",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			expectNoStatus(calc.Key())

			By("Closing the graph to disconnect the observer")
			Expect(g.Close()).To(Succeed())

			By("Deleting the base after close should not set error status")
			Expect(dist.Channel.Delete(ctx, base.Key(), false)).To(Succeed())
			expectNoStatus(calc.Key())
		})

		It("Should fail to open with missing config", func() {
			_, err := graph.Open(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Should fail to open with nil Channel", func() {
			_, err := graph.Open(ctx, graph.Config{Status: statusSvc})
			Expect(err).To(HaveOccurred())
		})

		It("Should fail to open with nil Status", func() {
			_, err := graph.Open(ctx, graph.Config{Channel: dist.Channel})
			Expect(err).To(HaveOccurred())
		})

		It("Should handle Close being called twice", func() {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				Channel: dist.Channel,
				Status:  statusSvc,
			}))
			Expect(g.Close()).To(Succeed())
			Expect(g.Close()).To(Succeed())
		})
	})

	Describe("Concurrency", func() {
		It("Should handle concurrent channel creation", func() {
			openGraph()
			var wg sync.WaitGroup
			const n = 5
			bases := make([]channel.Channel, n)
			calcs := make([]channel.Channel, n)
			for i := range n {
				bases[i] = channel.Channel{
					Name: fmt.Sprintf("cc_base_%d", i), DataType: telem.Int64T, Virtual: true,
				}
				Expect(dist.Channel.Create(ctx, &bases[i])).To(Succeed())
			}
			wg.Add(n)
			for i := range n {
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					calcs[i] = channel.Channel{
						Name:       fmt.Sprintf("cc_calc_%d", i),
						DataType:   telem.Int64T,
						Virtual:    true,
						Expression: fmt.Sprintf("return cc_base_%d + 1", i),
					}
					Expect(dist.Channel.Create(ctx, &calcs[i])).To(Succeed())
				}()
			}
			wg.Wait()
			for i := range n {
				expectNoStatus(calcs[i].Key())
			}
		})

		It("Should produce a consistent state under concurrent create and delete", func() {
			openGraph()
			base := channel.Channel{Name: "cc_race_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "cc_race_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return cc_race_base + 1",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			expectNoStatus(calc.Key())

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(dist.Channel.Delete(ctx, base.Key(), false)).To(Succeed())
			}()
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				newCalc := channel.Channel{
					Name: "cc_race_calc2", DataType: telem.Int64T, Virtual: true,
					Expression: "return cc_race_base * 2",
				}
				Expect(dist.Channel.Create(ctx, &newCalc)).To(Succeed())
			}()
			wg.Wait()
		})

		It("Should handle rapid sequential updates", func() {
			openGraph()
			base := channel.Channel{Name: "cc_rapid_base", DataType: telem.Int64T, Virtual: true}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "cc_rapid_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return cc_rapid_base + 1",
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())

			for i := range 10 {
				if i%2 == 0 {
					calc.Expression = "return cc_rapid_base + 1"
				} else {
					calc.Expression = "return cc_rapid_nonexistent"
				}
				Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			}
			expectNoStatus(calc.Key())
		})
	})

	Describe("Complex Dependency Topologies", Ordered, func() {
		Context("Build and Tear Down a 3-Level Diamond", func() {
			var (
				base1, base2 channel.Channel
				mid1, mid2   channel.Channel
				top          channel.Channel
			)

			BeforeAll(func() {
				openGraph()
				base1 = channel.Channel{Name: "topo_dia_b1", DataType: telem.Int64T, Virtual: true}
				base2 = channel.Channel{Name: "topo_dia_b2", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base1)).To(Succeed())
				Expect(dist.Channel.Create(ctx, &base2)).To(Succeed())
				mid1 = channel.Channel{
					Name: "topo_dia_m1", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_b1 + topo_dia_b2",
				}
				Expect(dist.Channel.Create(ctx, &mid1)).To(Succeed())
				mid2 = channel.Channel{
					Name: "topo_dia_m2", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_b1 * 2",
				}
				Expect(dist.Channel.Create(ctx, &mid2)).To(Succeed())
				top = channel.Channel{
					Name: "topo_dia_top", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_m1 + topo_dia_m2",
				}
				Expect(dist.Channel.Create(ctx, &top)).To(Succeed())
			})

			It("Should set up all levels as valid", func() {
				expectNoStatus(mid1.Key())
				expectNoStatus(mid2.Key())
				expectNoStatus(top.Key())
			})

			It("Should only affect mid1 when base2 is deleted", func() {
				By("Deleting base2 which is only used by mid1")
				Expect(dist.Channel.Delete(ctx, base2.Key(), false)).To(Succeed())

				By("mid1 depends on base2 so it gets error")
				expectStatus(mid1.Key())

				By("mid2 only depends on base1 so it stays valid")
				expectNoStatus(mid2.Key())

				By("top is not re-inspected because reconcileQueued " +
					"does not cascade invalidity from mid1")
				expectNoStatus(top.Key())
			})

			It("Should break top when mid1 is deleted", func() {
				Expect(dist.Channel.Delete(ctx, mid1.Key(), false)).To(Succeed())
				expectStatus(top.Key())
			})
		})

		Context("Long Chain With Mid-Chain Deletion", func() {
			BeforeAll(func() {
				openGraph()
			})
			It("Should only error the immediate dependent of a deleted node", func() {
				base := channel.Channel{Name: "topo_lc_base", DataType: telem.Int64T, Virtual: true}
				Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "topo_lc_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_base + 1",
				}
				Expect(dist.Channel.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "topo_lc_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c1 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "topo_lc_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c2 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "topo_lc_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c3 + 1",
				}
				Expect(dist.Channel.Create(ctx, &c4)).To(Succeed())

				By("Deleting c2 from the middle")
				Expect(dist.Channel.Delete(ctx, c2.Key(), false)).To(Succeed())

				By("c1 is upstream of deletion and unaffected")
				expectNoStatus(c1.Key())

				By("c3 directly depended on c2 and gets error")
				expectStatus(c3.Key())

				By("c4 is not re-inspected because invalidity does not cascade from c3")
				expectNoStatus(c4.Key())
			})
		})
	})
})
