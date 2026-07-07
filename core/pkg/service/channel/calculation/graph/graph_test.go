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
	"go/types"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	graph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db            *gorp.DB
	statusSvc     *status.Service
	channelSvc    *channel.Service
	channelWriter channel.Writer
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	db = node.DB
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostResolver: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	channelWriter = channelSvc.NewWriter(nil)
})

func openGraph(ctx context.Context) *graph.Graph {
	return MustOpen(graph.Open(ctx, graph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
}

func fetchStatus(ctx context.Context, key channel.Key) (status.Status[types.Nil], bool) {
	var statuses []status.Status[types.Nil]
	err := status.NewRetrieve[types.Nil](statusSvc).
		Where(status.MatchKeys[types.Nil](channel.OntologyID(key).String())).
		Entries(&statuses).
		Exec(ctx, nil)
	if err != nil || len(statuses) == 0 {
		return status.Status[types.Nil]{}, false
	}
	return statuses[0], true
}

func expectStatus(ctx context.Context, key channel.Key) status.Status[types.Nil] {
	var result status.Status[types.Nil]
	Eventually(func() bool {
		s, ok := fetchStatus(ctx, key)
		if ok && s.Variant == status.VariantError {
			result = s
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
		"expected error status for channel %d", key)
	return result
}

// eventuallyExpectNoStatus asserts that the calculated channel identified by key
// settles with no error status. The graph observes channel changes through an
// asynchronous, buffered observable, so it may transiently publish a status for an
// intermediate state — for example, an update event whose expression still references a
// dependency that a later, not-yet-processed event removes — before it catches up and
// clears it. The assertion therefore polls until the status has settled to absent
// rather than reading it once; a status that never clears still fails the spec.
func eventuallyExpectNoStatus(ctx context.Context, key channel.Key) {
	Eventually(func() bool {
		_, ok := fetchStatus(ctx, key)
		return ok
	}, 2*time.Second, 10*time.Millisecond).Should(BeFalse(),
		"expected status to be cleared for channel %d", key)
}

func retrieveChannelDataType(ctx context.Context, key channel.Key) telem.DataType {
	var ch channel.Channel
	Expect(channelSvc.NewRetrieve().Where(channel.MatchKeys(key)).Entry(&ch).Exec(ctx, nil)).To(Succeed())
	return ch.DataType
}

// createBrokenCalc creates a calculated channel referencing depName, then deletes
// depName, leaving the channel with an unresolvable reference — the way a calculated
// channel actually becomes invalid at rest (an upstream channel is deleted). Strict
// creation forbids creating a channel against a name that never existed, so tests
// reproduce broken channels this way. Callers can recreate depName (see createDep) to
// heal it.
func createBrokenCalc(ctx context.Context, name, depName string) channel.Channel {
	GinkgoHelper()
	createDep(ctx, depName)
	calc := channel.Channel{
		Name: name, DataType: telem.Int64T, Virtual: true,
		Expression: "return " + depName + " + 1",
	}
	Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
	deleteDep(ctx, depName)
	return calc
}

// createDep creates (or recreates) a virtual Int64 base channel with the given name.
func createDep(ctx context.Context, depName string) channel.Channel {
	GinkgoHelper()
	dep := channel.Channel{Name: depName, DataType: telem.Int64T, Virtual: true}
	Expect(channelWriter.Create(ctx, &dep)).To(Succeed())
	return dep
}

// deleteDep deletes the base channel with the given name.
func deleteDep(ctx context.Context, depName string) {
	GinkgoHelper()
	Expect(channelWriter.DeleteManyByNames(
		ctx, []string{depName}, false,
	)).To(Succeed())
}

// makeStale overwrites the stored DataType of an existing channel with a wrong value,
// without analysis, so the graph has something to repair. It mirrors how a stored
// DataType drifts from what a calculated channel's expression now infers.
func makeStale(ctx context.Context, ch channel.Channel, stale telem.DataType) {
	GinkgoHelper()
	Expect(channelWriter.ChangeDataType(ctx, ch.Key(), stale)).To(Succeed())
}

var _ = Describe("Graph", func() {

	Describe("Open / Hydration", func() {

		It("Should open successfully with no calculated channels", func(ctx SpecContext) {
			bases := []channel.Channel{
				{Name: "hy_base1", DataType: telem.Int64T, Virtual: true},
				{Name: "hy_base2", DataType: telem.Float64T, Virtual: true},
			}
			Expect(channelWriter.CreateMany(ctx, &bases)).To(Succeed())
			openGraph(ctx)
		})

		It("Should open with a valid calculated channel and set no status", func(ctx SpecContext) {
			base := channel.Channel{Name: "hy_valid_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "hy_valid_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_valid_base * 2",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			openGraph(ctx)
			eventuallyExpectNoStatus(ctx, calc.Key())
		})

		It("Should set error status for an invalid expression at open", func(ctx SpecContext) {
			// A syntax error cannot exist at rest under strict creation; the graph's
			// handling of an invalid expression is exercised via an unresolvable
			// reference (a dependency deleted after the calc was created). Syntax-error
			// analysis itself is covered by the analyzer and compiler suites.
			calc := createBrokenCalc(ctx, "hy_invalid", "hy_invalid_dep")
			openGraph(ctx)
			expectStatus(ctx, calc.Key())
		})

		It("Should set error status for an unresolvable reference", func(ctx SpecContext) {
			base := channel.Channel{
				Name: "hy_unresolvable_base", DataType: telem.Int64T, Virtual: true,
			}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "hy_unresolvable", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_unresolvable_base * 2",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			Expect(channelWriter.Delete(
				ctx, base.Key(), false,
			)).To(Succeed())
			openGraph(ctx)
			expectStatus(ctx, calc.Key())
		})

		It("Should handle a mix of valid and invalid calculated channels", func(ctx SpecContext) {
			base := channel.Channel{Name: "hy_mix_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calcOk := channel.Channel{
				Name: "hy_mix_ok", DataType: telem.Int64T, Virtual: true,
				Expression: "return hy_mix_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calcOk)).To(Succeed())
			calcBad := createBrokenCalc(ctx, "hy_mix_bad", "hy_mix_bad_dep")
			openGraph(ctx)
			eventuallyExpectNoStatus(ctx, calcOk.Key())
			expectStatus(ctx, calcBad.Key())
		})

		It("Should handle an orphan calculated channel with no dependencies", func(ctx SpecContext) {
			calc := channel.Channel{
				Name: "hy_orphan", DataType: telem.Int64T, Virtual: true,
				Expression: "return 42",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			openGraph(ctx)
			eventuallyExpectNoStatus(ctx, calc.Key())
		})

		Context("Dependency Topologies", func() {
			It("Should hydrate a diamond dependency graph", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_dia_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calcB := channel.Channel{
					Name: "hy_dia_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calcB)).To(Succeed())
				calcC := channel.Channel{
					Name: "hy_dia_c", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_base * 2",
				}
				Expect(channelWriter.Create(ctx, &calcC)).To(Succeed())
				calcA := channel.Channel{
					Name: "hy_dia_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_dia_b + hy_dia_c",
				}
				Expect(channelWriter.Create(ctx, &calcA)).To(Succeed())
				openGraph(ctx)
				eventuallyExpectNoStatus(ctx, calcA.Key())
				eventuallyExpectNoStatus(ctx, calcB.Key())
				eventuallyExpectNoStatus(ctx, calcC.Key())
			})

			It("Should hydrate a deep chain (4 levels)", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_deep_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "hy_deep_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_base + 1",
				}
				Expect(channelWriter.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "hy_deep_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c1 + 1",
				}
				Expect(channelWriter.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "hy_deep_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c2 + 1",
				}
				Expect(channelWriter.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "hy_deep_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_deep_c3 + 1",
				}
				Expect(channelWriter.Create(ctx, &c4)).To(Succeed())
				openGraph(ctx)
				eventuallyExpectNoStatus(ctx, c1.Key())
				eventuallyExpectNoStatus(ctx, c2.Key())
				eventuallyExpectNoStatus(ctx, c3.Key())
				eventuallyExpectNoStatus(ctx, c4.Key())
			})

			It("Should hydrate a fan-out topology", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_fan_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
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
				Expect(channelWriter.CreateMany(ctx, &calcs)).To(Succeed())
				openGraph(ctx)
				eventuallyExpectNoStatus(ctx, calcs[0].Key())
				eventuallyExpectNoStatus(ctx, calcs[1].Key())
				eventuallyExpectNoStatus(ctx, calcs[2].Key())
			})

			It("Should hydrate a fan-in topology", func(ctx SpecContext) {
				bases := []channel.Channel{
					{Name: "hy_fin_b1", DataType: telem.Int64T, Virtual: true},
					{Name: "hy_fin_b2", DataType: telem.Int64T, Virtual: true},
					{Name: "hy_fin_b3", DataType: telem.Int64T, Virtual: true},
				}
				Expect(channelWriter.CreateMany(ctx, &bases)).To(Succeed())
				calc := channel.Channel{
					Name: "hy_fin_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_fin_b1 + hy_fin_b2 + hy_fin_b3",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				openGraph(ctx)
				eventuallyExpectNoStatus(ctx, calc.Key())
			})
		})

		Context("DataType Repair", func() {
			It("Should not repair when DataType already matches", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_norep_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "hy_norep_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_norep_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				openGraph(ctx)
				Expect(retrieveChannelDataType(ctx, calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair a stale DataType during hydration", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_rep_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name:       "hy_rep_calc",
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return hy_rep_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				makeStale(ctx, calc, telem.Float32T)
				Expect(retrieveChannelDataType(ctx, calc.Key())).To(Equal(telem.Float32T))
				openGraph(ctx)
				Expect(retrieveChannelDataType(ctx, calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair cascaded DataType when a dependent has a lower key than its dependency", func(ctx SpecContext) {
				base := channel.Channel{Name: "hy_ooo_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())

				calc1 := channel.Channel{
					Name:       "hy_ooo_c1",
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return hy_ooo_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name:       "hy_ooo_c2",
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return hy_ooo_c1 + 1",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				Expect(
					channelWriter.Delete(ctx, calc1.Key(), false),
				).To(Succeed())
				calc1 = channel.Channel{
					Name:       "hy_ooo_c1",
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return hy_ooo_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())

				Expect(calc2.Key()).To(BeNumerically("<", calc1.Key()))
				makeStale(ctx, calc1, telem.Float32T)
				makeStale(ctx, calc2, telem.Float32T)
				Expect(retrieveChannelDataType(ctx, calc1.Key())).To(Equal(telem.Float32T))
				Expect(retrieveChannelDataType(ctx, calc2.Key())).To(Equal(telem.Float32T))

				openGraph(ctx)

				Expect(retrieveChannelDataType(ctx, calc1.Key())).To(Equal(telem.Int64T))
				Expect(retrieveChannelDataType(ctx, calc2.Key())).To(Equal(telem.Int64T))
			})
		})
	})

	Describe("Reactive Change Handling", func() {

		Context("Creating Channels", func() {
			It("Should inspect a new valid calculated channel", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_create_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_create_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_create_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())
			})

			It("Should set error status for a newly-invalidated calculated channel", func(ctx SpecContext) {
				openGraph(ctx)
				calc := createBrokenCalc(ctx, "rc_create_bad", "rc_create_bad_dep")
				expectStatus(ctx, calc.Key())
			})

			It("Should handle incrementally building a chain after graph open", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_chain_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_chain_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
				calc2 := channel.Channel{
					Name: "rc_chain_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc2.Key())
			})

			It("Should process a batch CreateMany in a single handleChanges call", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_batch_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calcs := []channel.Channel{
					{Name: "rc_batch_c1", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base + 1"},
					{Name: "rc_batch_c2", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base * 2"},
					{Name: "rc_batch_c3", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base - 1"},
				}
				Expect(channelWriter.CreateMany(ctx, &calcs)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calcs[0].Key())
				eventuallyExpectNoStatus(ctx, calcs[1].Key())
				eventuallyExpectNoStatus(ctx, calcs[2].Key())
			})
		})

		Context("Deleting Channels", func() {
			It("Should set error status when a base dependency is deleted", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_del_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_del_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Deleting the base dependency")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).
					To(Succeed())
				expectStatus(ctx, calc.Key())
			})

			It("Should set error on downstream calc when intermediate calc is deleted", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_del_mid_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_del_mid_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_mid_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_del_mid_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_mid_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())

				By("Deleting the intermediate calculated channel")
				Expect(channelWriter.Delete(ctx, calc1.Key(), false)).To(Succeed())
				expectStatus(ctx, calc2.Key())
			})

			It("Should leave upstream unaffected when a leaf calc is deleted", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_del_leaf_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_del_leaf_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_leaf_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_del_leaf_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_leaf_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())

				By("Deleting the leaf calc")
				Expect(channelWriter.Delete(ctx, calc2.Key(), false)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
			})

			It("Should not cascade invalidity through reconcileQueued in a diamond", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_del_dia_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calcB := channel.Channel{
					Name: "rc_del_dia_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calcB)).To(Succeed())
				calcC := channel.Channel{
					Name: "rc_del_dia_c", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_base * 2",
				}
				Expect(channelWriter.Create(ctx, &calcC)).To(Succeed())
				calcA := channel.Channel{
					Name: "rc_del_dia_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_dia_b + rc_del_dia_c",
				}
				Expect(channelWriter.Create(ctx, &calcA)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calcA.Key())

				By("Deleting the shared base dependency")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())

				By("Verifying calc_b and calc_c get error statuses")
				expectStatus(ctx, calcB.Key())
				expectStatus(ctx, calcC.Key())

				By("Verifying calc_a does NOT get error status because " +
					"reconcileQueued continues without enqueueing dependents on error")
				eventuallyExpectNoStatus(ctx, calcA.Key())
			})
		})

		Context("Updating Channels", func() {
			It("Should update deps when expression changes to use a different base", func(ctx SpecContext) {
				openGraph(ctx)
				base1 := channel.Channel{Name: "rc_upd_b1", DataType: telem.Int64T, Virtual: true}
				base2 := channel.Channel{Name: "rc_upd_b2", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base1)).To(Succeed())
				Expect(channelWriter.Create(ctx, &base2)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_upd_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_b1 + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Updating expression to use a different base")
				calc.Expression = "return rc_upd_b2 * 2"
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Verifying old base deletion does not affect calc")
				Expect(channelWriter.Delete(ctx, base1.Key(), false)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Verifying new base deletion does affect calc")
				Expect(channelWriter.Delete(ctx, base2.Key(), false)).To(Succeed())
				expectStatus(ctx, calc.Key())
			})

			It("Should set error status when a dependency is removed", func(ctx SpecContext) {
				openGraph(ctx)
				createDep(ctx, "rc_upd_bad_base")
				calc := channel.Channel{
					Name: "rc_upd_bad_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_bad_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Removing the dependency so the expression no longer resolves")
				deleteDep(ctx, "rc_upd_bad_base")
				expectStatus(ctx, calc.Key())
			})

			It("Should clear error status when a broken dependency is restored", func(ctx SpecContext) {
				openGraph(ctx)
				calc := createBrokenCalc(ctx, "rc_upd_fix", "rc_fix_missing_dep")
				expectStatus(ctx, calc.Key())

				By("Recreating the missing dependency")
				createDep(ctx, "rc_fix_missing_dep")
				eventuallyExpectNoStatus(ctx, calc.Key())
			})
		})

		Context("Cascading Reconciliation", func() {
			It("Should not cascade invalidity from reconcileQueued to further dependents", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_cas_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_cas_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_cas_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_cas_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_cas_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())

				By("Deleting the base. calc1 becomes invalid. " +
					"calc2 should NOT get error because reconcileQueued " +
					"does not enqueue dependents when a node errors")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
				expectStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())
			})

			It("Should cascade deletion through a long chain", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_long_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "rc_long_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_base + 1",
				}
				Expect(channelWriter.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "rc_long_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c1 + 1",
				}
				Expect(channelWriter.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "rc_long_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c2 + 1",
				}
				Expect(channelWriter.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "rc_long_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_long_c3 + 1",
				}
				Expect(channelWriter.Create(ctx, &c4)).To(Succeed())

				By("Deleting c2 from the middle of the chain")
				Expect(channelWriter.Delete(ctx, c2.Key(), false)).To(Succeed())

				By("c1 is upstream and unaffected")
				eventuallyExpectNoStatus(ctx, c1.Key())

				By("c3 depends on c2 which is gone, so it gets error")
				expectStatus(ctx, c3.Key())

				By("c4 does not get error because reconcileQueued does not " +
					"cascade invalidity from c3's failure")
				eventuallyExpectNoStatus(ctx, c4.Key())
			})

			It("Should re-inspect dependents when a calculated channel is updated", func(ctx SpecContext) {
				openGraph(ctx)
				base := channel.Channel{Name: "rc_reins_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_reins_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_reins_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_reins_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_reins_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())

				By("Updating calc1 expression - calc2 should be re-inspected via BFS")
				calc1.Expression = "return rc_reins_base + 100"
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())
			})
		})

		Context("DataType Persistence", func() {
			It("Should persist DataType changes to the DB when a dependency type changes", func(ctx SpecContext) {
				openGraph(ctx)

				By("Creating a base channel and a calc that depends on it")
				base := channel.Channel{Name: "rc_dtp_base", DataType: telem.Float32T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_dtp_calc", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtp_base * 2",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Updating the calc expression to return a different type")
				calc.Expression = "return i64(rc_dtp_base)"
				calc.DataType = telem.Int64T
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				eventuallyExpectNoStatus(ctx, calc.Key())

				By("Verifying the DataType was persisted to the DB")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(ctx, calc.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})

			It("Should persist cascaded DataType changes through a chain", func(ctx SpecContext) {
				openGraph(ctx)

				By("Building a chain: base -> calc1 -> calc2")
				base := channel.Channel{Name: "rc_dtpc_base", DataType: telem.Float32T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_dtpc_c1", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtpc_base * 2",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				calc2 := channel.Channel{
					Name: "rc_dtpc_c2", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtpc_c1 + 1",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())

				By("Updating calc1 to return a different type, which should cascade to calc2")
				calc1.Expression = "return i64(rc_dtpc_base)"
				calc1.DataType = telem.Int64T
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())

				By("Verifying calc1 DataType was persisted")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(ctx, calc1.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))

				By("Verifying calc2 DataType was also updated via cascade")
				Eventually(func() telem.DataType {
					return retrieveChannelDataType(ctx, calc2.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})
		})

		Context("Unresolved Name Auto-Heal", func() {
			It("Should auto-fix a broken calc when its deleted dependency is recreated", func(ctx SpecContext) {
				openGraph(ctx)
				calc := createBrokenCalc(ctx, "rc_unres_calc", "rc_unres_missing")
				expectStatus(ctx, calc.Key())

				By("Recreating the previously deleted dependency")
				createDep(ctx, "rc_unres_missing")

				By("Verifying calc is auto-fixed")
				eventuallyExpectNoStatus(ctx, calc.Key())
			})

			It("Should auto-fix multiple calcs waiting on the same restored name", func(ctx SpecContext) {
				openGraph(ctx)
				createDep(ctx, "rc_unres_shared_dep")
				calc1 := channel.Channel{
					Name: "rc_unres_multi_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_shared_dep + 1",
				}
				calc2 := channel.Channel{
					Name: "rc_unres_multi_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_unres_shared_dep * 2",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())

				By("Deleting the shared dependency so both calcs break")
				deleteDep(ctx, "rc_unres_shared_dep")
				expectStatus(ctx, calc1.Key())
				expectStatus(ctx, calc2.Key())

				By("Recreating the shared dependency")
				createDep(ctx, "rc_unres_shared_dep")

				By("Both calcs should auto-heal")
				eventuallyExpectNoStatus(ctx, calc1.Key())
				eventuallyExpectNoStatus(ctx, calc2.Key())
			})

			It("Should auto-fix a chain when a deleted base is recreated", func(ctx SpecContext) {
				openGraph(ctx)
				calc1 := createBrokenCalc(ctx, "rc_unres_chain_c1", "rc_unres_chain_base")
				expectStatus(ctx, calc1.Key())

				By("Recreating the deleted base")
				createDep(ctx, "rc_unres_chain_base")

				By("calc1 should auto-heal")
				eventuallyExpectNoStatus(ctx, calc1.Key())
			})
		})

		Context("Multiple Independent Subgraphs", func() {
			It("Should isolate failures to their own subgraph", func(ctx SpecContext) {
				openGraph(ctx)
				baseA := channel.Channel{Name: "rc_iso_base_a", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &baseA)).To(Succeed())
				calcA := channel.Channel{
					Name: "rc_iso_calc_a", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_iso_base_a + 1",
				}
				Expect(channelWriter.Create(ctx, &calcA)).To(Succeed())

				baseB := channel.Channel{Name: "rc_iso_base_b", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &baseB)).To(Succeed())
				calcB := channel.Channel{
					Name: "rc_iso_calc_b", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_iso_base_b + 1",
				}
				Expect(channelWriter.Create(ctx, &calcB)).To(Succeed())

				By("Deleting base_a should only affect calc_a")
				Expect(channelWriter.Delete(ctx, baseA.Key(), false)).To(Succeed())
				expectStatus(ctx, calcA.Key())
				eventuallyExpectNoStatus(ctx, calcB.Key())
			})
		})
	})

	Describe("Status Communication", func() {
		It("Should set status with correct structure and details", func(ctx SpecContext) {
			calc := createBrokenCalc(ctx, "st_detail", "st_missing_detail_dep")
			openGraph(ctx)

			s := expectStatus(ctx, calc.Key())
			Expect(s.Variant).To(Equal(status.VariantError))
			Expect(s.Message).To(Equal("invalid expression for st_detail"))
			Expect(s.Description).ToNot(BeEmpty())
			Expect(s.Key).To(Equal(channel.OntologyID(calc.Key()).String()))
			Expect(s.Name).To(Equal("st_detail"))
		})

		It("Should clear status when a broken dependency is restored", func(ctx SpecContext) {
			openGraph(ctx)
			calc := createBrokenCalc(ctx, "st_clear", "st_clear_dep")
			expectStatus(ctx, calc.Key())

			By("Recreating the dependency")
			createDep(ctx, "st_clear_dep")
			eventuallyExpectNoStatus(ctx, calc.Key())
		})

		It("Should overwrite status when the error changes", func(ctx SpecContext) {
			openGraph(ctx)
			createDep(ctx, "st_ow_a")
			createDep(ctx, "st_ow_b")
			calc := channel.Channel{
				Name: "st_overwrite", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_ow_a + st_ow_b",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())

			By("Deleting st_ow_a so the calc breaks on a missing st_ow_a")
			deleteDep(ctx, "st_ow_a")
			s1 := expectStatus(ctx, calc.Key())

			By("Restoring st_ow_a and deleting st_ow_b so the error becomes a different one")
			createDep(ctx, "st_ow_a")
			deleteDep(ctx, "st_ow_b")
			var s2 status.Status[types.Nil]
			Eventually(func() bool {
				s, ok := fetchStatus(ctx, calc.Key())
				if ok && s.Description != s1.Description {
					s2 = s
					return true
				}
				return false
			}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
				"expected status description to change")
			Expect(s2.Variant).To(Equal(status.VariantError))
		})

		It("Should not create any status entry for valid channels", func(ctx SpecContext) {
			base := channel.Channel{Name: "st_none_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "st_none_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_none_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			openGraph(ctx)
			eventuallyExpectNoStatus(ctx, calc.Key())
		})
	})

	Describe("Lifecycle", func() {
		It("Should open and close without error", func(ctx SpecContext) {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				DB:      db,
				Channel: channelSvc,
				Status:  statusSvc,
			}))
			Expect(g.Close()).To(Succeed())
		})

		It("Should disconnect observer on Close", func(ctx SpecContext) {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				DB:      db,
				Channel: channelSvc,
				Status:  statusSvc,
			}))
			base := channel.Channel{Name: "lc_disc_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "lc_disc_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return lc_disc_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())

			By("Closing the graph to disconnect the observer")
			Expect(g.Close()).To(Succeed())

			By("Deleting the base after close should not set error status")
			Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())
		})

		It("Should fail to open with missing config", func(ctx SpecContext) {
			_, err := graph.Open(ctx)
			Expect(err).To(HaveOccurred())
		})

		It("Should fail to open with nil Channel", func(ctx SpecContext) {
			_, err := graph.Open(ctx, graph.Config{Status: statusSvc})
			Expect(err).To(HaveOccurred())
		})

		It("Should fail to open with nil Status", func(ctx SpecContext) {
			_, err := graph.Open(ctx, graph.Config{Channel: channelSvc})
			Expect(err).To(HaveOccurred())
		})

		It("Should fail to open with nil DB", func(ctx SpecContext) {
			Expect(graph.Open(ctx, graph.Config{
				Channel: channelSvc,
				Status:  statusSvc,
			})).Error().To(HaveOccurred())
		})

		It("Should handle Close being called twice", func(ctx SpecContext) {
			g := MustSucceed(graph.Open(ctx, graph.Config{
				DB:      db,
				Channel: channelSvc,
				Status:  statusSvc,
			}))
			Expect(g.Close()).To(Succeed())
			Expect(g.Close()).To(Succeed())
		})
	})

	Describe("Concurrency", func() {
		It("Should handle concurrent channel creation", func(ctx SpecContext) {
			openGraph(ctx)
			var wg sync.WaitGroup
			const n = 5
			bases := make([]channel.Channel, n)
			calcs := make([]channel.Channel, n)
			for i := range n {
				bases[i] = channel.Channel{
					Name: fmt.Sprintf("cc_base_%d", i), DataType: telem.Int64T, Virtual: true,
				}
				Expect(channelWriter.Create(ctx, &bases[i])).To(Succeed())
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
					Expect(channelWriter.Create(ctx, &calcs[i])).To(Succeed())
				}()
			}
			wg.Wait()
			for i := range n {
				eventuallyExpectNoStatus(ctx, calcs[i].Key())
			}
		})

		It("Should produce a consistent state under concurrent create and delete", func(ctx SpecContext) {
			openGraph(ctx)
			base := channel.Channel{Name: "cc_race_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			stable := channel.Channel{Name: "cc_race_stable", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &stable)).To(Succeed())
			calc := channel.Channel{
				Name: "cc_race_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return cc_race_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
			}()
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				// References a stable base (not the one being deleted) so strict
				// creation always succeeds regardless of how it races with the delete.
				newCalc := channel.Channel{
					Name: "cc_race_calc2", DataType: telem.Int64T, Virtual: true,
					Expression: "return cc_race_stable * 2",
				}
				Expect(channelWriter.Create(ctx, &newCalc)).To(Succeed())
			}()
			wg.Wait()
		})

		It("Should handle rapid sequential updates", func(ctx SpecContext) {
			openGraph(ctx)
			base := channel.Channel{Name: "cc_rapid_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			base2 := channel.Channel{Name: "cc_rapid_base2", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base2)).To(Succeed())
			calc := channel.Channel{
				Name: "cc_rapid_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return cc_rapid_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())

			// Alternate the expression between two valid dependencies; an update to an
			// invalid expression is rejected at creation, so the reachable form of
			// "rapid updates" churns between valid states.
			for i := range 10 {
				if i%2 == 0 {
					calc.Expression = "return cc_rapid_base + 1"
				} else {
					calc.Expression = "return cc_rapid_base2 * 2"
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			}
			calc.Expression = "return cc_rapid_base + 1"
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			eventuallyExpectNoStatus(ctx, calc.Key())
		})
	})

	Describe("Complex Dependency Topologies", Ordered, func() {
		Context("Build and Tear Down a 3-Level Diamond", func() {
			var (
				base1, base2 channel.Channel
				mid1, mid2   channel.Channel
				top          channel.Channel
			)

			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				openGraph(ctx)
				base1 = channel.Channel{Name: "topo_dia_b1", DataType: telem.Int64T, Virtual: true}
				base2 = channel.Channel{Name: "topo_dia_b2", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base1)).To(Succeed())
				Expect(channelWriter.Create(ctx, &base2)).To(Succeed())
				mid1 = channel.Channel{
					Name: "topo_dia_m1", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_b1 + topo_dia_b2",
				}
				Expect(channelWriter.Create(ctx, &mid1)).To(Succeed())
				mid2 = channel.Channel{
					Name: "topo_dia_m2", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_b1 * 2",
				}
				Expect(channelWriter.Create(ctx, &mid2)).To(Succeed())
				top = channel.Channel{
					Name: "topo_dia_top", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_dia_m1 + topo_dia_m2",
				}
				Expect(channelWriter.Create(ctx, &top)).To(Succeed())
			})

			It("Should set up all levels as valid", func(ctx SpecContext) {
				eventuallyExpectNoStatus(ctx, mid1.Key())
				eventuallyExpectNoStatus(ctx, mid2.Key())
				eventuallyExpectNoStatus(ctx, top.Key())
			})

			It("Should only affect mid1 when base2 is deleted", func(ctx SpecContext) {
				By("Deleting base2 which is only used by mid1")
				Expect(channelWriter.Delete(ctx, base2.Key(), false)).To(Succeed())

				By("mid1 depends on base2 so it gets error")
				expectStatus(ctx, mid1.Key())

				By("mid2 only depends on base1 so it stays valid")
				eventuallyExpectNoStatus(ctx, mid2.Key())

				By("top is not re-inspected because reconcileQueued " +
					"does not cascade invalidity from mid1")
				eventuallyExpectNoStatus(ctx, top.Key())
			})

			It("Should break top when mid1 is deleted", func(ctx SpecContext) {
				Expect(channelWriter.Delete(ctx, mid1.Key(), false)).To(Succeed())
				expectStatus(ctx, top.Key())
			})
		})

		Context("Long Chain With Mid-Chain Deletion", func() {
			BeforeAll(func(ctx SpecContext) {
				ShouldNotLeakGoroutines()
				openGraph(ctx)
			})
			It("Should only error the immediate dependent of a deleted node", func(ctx SpecContext) {
				base := channel.Channel{Name: "topo_lc_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				c1 := channel.Channel{
					Name: "topo_lc_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_base + 1",
				}
				Expect(channelWriter.Create(ctx, &c1)).To(Succeed())
				c2 := channel.Channel{
					Name: "topo_lc_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c1 + 1",
				}
				Expect(channelWriter.Create(ctx, &c2)).To(Succeed())
				c3 := channel.Channel{
					Name: "topo_lc_c3", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c2 + 1",
				}
				Expect(channelWriter.Create(ctx, &c3)).To(Succeed())
				c4 := channel.Channel{
					Name: "topo_lc_c4", DataType: telem.Int64T, Virtual: true,
					Expression: "return topo_lc_c3 + 1",
				}
				Expect(channelWriter.Create(ctx, &c4)).To(Succeed())

				By("Deleting c2 from the middle")
				Expect(channelWriter.Delete(ctx, c2.Key(), false)).To(Succeed())

				By("c1 is upstream of deletion and unaffected")
				eventuallyExpectNoStatus(ctx, c1.Key())

				By("c3 directly depended on c2 and gets error")
				expectStatus(ctx, c3.Key())

				By("c4 is not re-inspected because invalidity does not cascade from c3")
				eventuallyExpectNoStatus(ctx, c4.Key())
			})
		})
	})
})
