// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func fetchCalcStatus(
	ctx context.Context,
	statusSvc *status.Service,
	key channel.Key,
) (channel.CalculationStatus, bool) {
	var statuses []channel.CalculationStatus
	err := status.NewRetrieve[types.Nil](statusSvc).
		Where(status.MatchKeys[types.Nil](channel.CalculationStatusKey(key))).
		Entries(&statuses).
		Exec(ctx, nil)
	if err != nil || len(statuses) == 0 {
		return channel.CalculationStatus{}, false
	}
	return statuses[0], true
}

func expectCalcStatus(
	ctx context.Context,
	statusSvc *status.Service,
	key channel.Key,
) channel.CalculationStatus {
	var result channel.CalculationStatus
	Eventually(func() bool {
		s, ok := fetchCalcStatus(ctx, statusSvc, key)
		if ok && s.Variant == status.VariantError {
			result = s
			return true
		}
		return false
	}, 2*time.Second, 10*time.Millisecond).Should(BeTrue(),
		"expected error status for channel %d", key)
	return result
}

// expectNoCalcStatus asserts that the calculated channel identified by key settles
// with no error status. The graph observes channel changes through an asynchronous,
// buffered observable, so it may transiently publish a status for an intermediate
// state — for example, an update event whose expression still references a dependency
// that a later, not-yet-processed event removes — before it catches up and clears it.
// The assertion therefore polls until the status has settled to absent rather than
// reading it once; a status that never clears still fails the spec.
func expectNoCalcStatus(ctx context.Context, statusSvc *status.Service, key channel.Key) {
	Eventually(func() bool {
		_, ok := fetchCalcStatus(ctx, statusSvc, key)
		return ok
	}, 2*time.Second, 10*time.Millisecond).Should(BeFalse(),
		"expected status to be cleared for channel %d", key)
}

func retrieveDataType(
	ctx context.Context,
	channelSvc *channel.Service,
	key channel.Key,
) telem.DataType {
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
func createBrokenCalc(
	ctx context.Context,
	w channel.Writer,
	name, depName string,
) channel.Channel {
	GinkgoHelper()
	createDep(ctx, w, depName)
	calc := channel.Channel{
		Name: name, DataType: telem.Int64T, Virtual: true,
		Expression: "return " + depName + " + 1",
	}
	Expect(w.Create(ctx, &calc)).To(Succeed())
	deleteDep(ctx, w, depName)
	return calc
}

// createDep creates (or recreates) a virtual Int64 base channel with the given name.
func createDep(ctx context.Context, w channel.Writer, depName string) channel.Channel {
	GinkgoHelper()
	dep := channel.Channel{Name: depName, DataType: telem.Int64T, Virtual: true}
	Expect(w.Create(ctx, &dep)).To(Succeed())
	return dep
}

// deleteDep deletes the base channel with the given name.
func deleteDep(ctx context.Context, w channel.Writer, depName string) {
	GinkgoHelper()
	Expect(w.DeleteManyByNames(ctx, []string{depName}, false)).To(Succeed())
}

// reopenService provisions a fresh node, opens a channel service on it, runs setup
// against that service's writer, and closes the service. It then runs atRest (if
// non-nil) against the node's bare DB — no service open, so changes are invisible to
// any graph — and reopens a fresh service stack over the same DB, exercising the
// hydration path in OpenService against at-rest state.
func reopenService(
	ctx context.Context,
	setup func(w channel.Writer),
	atRest func(db *gorp.DB),
) (*channel.Service, channel.ServiceConfig) {
	GinkgoHelper()
	node := mock.NewNode(ctx)
	firstCfg := serviceConfig(ctx, node)
	first := MustSucceed(channel.OpenService(ctx, firstCfg))
	setup(first.NewWriter(nil))
	Expect(first.Close()).To(Succeed())
	if atRest != nil {
		atRest(node.DB)
	}
	cfg := serviceConfig(ctx, node)
	return MustOpen(channel.OpenService(ctx, cfg)), cfg
}

// staleAtRest overwrites the stored DataType of an existing channel with a wrong
// value via a raw gorp update, without analysis, so hydration has something to
// repair. It mirrors how a stored DataType drifts from what a calculated channel's
// expression now infers.
func staleAtRest(
	ctx context.Context,
	db *gorp.DB,
	key channel.Key,
	stale telem.DataType,
) {
	GinkgoHelper()
	Expect(gorp.NewUpdate[channel.Key, channel.Channel]().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](key)).
		Change(func(_ gorp.Context, ch channel.Channel) channel.Channel {
			ch.DataType = stale
			return ch
		}).Exec(ctx, db)).To(Succeed())
	var ch channel.Channel
	Expect(gorp.NewRetrieve[channel.Key, channel.Channel]().
		Where(gorp.MatchKeys[channel.Key, channel.Channel](key)).
		Entry(&ch).Exec(ctx, db)).To(Succeed())
	Expect(ch.DataType).To(Equal(stale))
}

var _ = Describe("Calculation Graph", func() {

	Describe("Hydration", func() {

		It("Should open successfully with no calculated channels", func(ctx SpecContext) {
			reopenService(ctx, func(w channel.Writer) {
				bases := []channel.Channel{
					{Name: "hy_base1", DataType: telem.Int64T, Virtual: true},
					{Name: "hy_base2", DataType: telem.Float64T, Virtual: true},
				}
				Expect(w.CreateMany(ctx, &bases)).To(Succeed())
			}, nil)
		})

		It("Should open with a valid calculated channel and set no status", func(ctx SpecContext) {
			var calc channel.Channel
			_, cfg := reopenService(ctx, func(w channel.Writer) {
				base := channel.Channel{Name: "hy_valid_base", DataType: telem.Int64T, Virtual: true}
				Expect(w.Create(ctx, &base)).To(Succeed())
				calc = channel.Channel{
					Name: "hy_valid_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_valid_base * 2",
				}
				Expect(w.Create(ctx, &calc)).To(Succeed())
			}, nil)
			expectNoCalcStatus(ctx, cfg.Status, calc.Key())
		})

		It("Should set error status for an invalid expression at open", func(ctx SpecContext) {
			// A syntax error cannot exist at rest under strict creation; the graph's
			// handling of an invalid expression is exercised via an unresolvable
			// reference (a dependency deleted after the calc was created). Syntax-error
			// analysis itself is covered by the analyzer and compiler suites.
			var calc channel.Channel
			_, cfg := reopenService(ctx, func(w channel.Writer) {
				calc = createBrokenCalc(ctx, w, "hy_invalid", "hy_invalid_dep")
			}, nil)
			expectCalcStatus(ctx, cfg.Status, calc.Key())
		})

		It("Should set error status for an unresolvable reference", func(ctx SpecContext) {
			var calc channel.Channel
			_, cfg := reopenService(ctx, func(w channel.Writer) {
				base := channel.Channel{
					Name: "hy_unresolvable_base", DataType: telem.Int64T, Virtual: true,
				}
				Expect(w.Create(ctx, &base)).To(Succeed())
				calc = channel.Channel{
					Name: "hy_unresolvable", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_unresolvable_base * 2",
				}
				Expect(w.Create(ctx, &calc)).To(Succeed())
				Expect(w.Delete(ctx, base.Key(), false)).To(Succeed())
			}, nil)
			expectCalcStatus(ctx, cfg.Status, calc.Key())
		})

		It("Should handle a mix of valid and invalid calculated channels", func(ctx SpecContext) {
			var calcOk, calcBad channel.Channel
			_, cfg := reopenService(ctx, func(w channel.Writer) {
				base := channel.Channel{Name: "hy_mix_base", DataType: telem.Int64T, Virtual: true}
				Expect(w.Create(ctx, &base)).To(Succeed())
				calcOk = channel.Channel{
					Name: "hy_mix_ok", DataType: telem.Int64T, Virtual: true,
					Expression: "return hy_mix_base + 1",
				}
				Expect(w.Create(ctx, &calcOk)).To(Succeed())
				calcBad = createBrokenCalc(ctx, w, "hy_mix_bad", "hy_mix_bad_dep")
			}, nil)
			expectNoCalcStatus(ctx, cfg.Status, calcOk.Key())
			expectCalcStatus(ctx, cfg.Status, calcBad.Key())
		})

		It("Should handle an orphan calculated channel with no dependencies", func(ctx SpecContext) {
			var calc channel.Channel
			_, cfg := reopenService(ctx, func(w channel.Writer) {
				calc = channel.Channel{
					Name: "hy_orphan", DataType: telem.Int64T, Virtual: true,
					Expression: "return 42",
				}
				Expect(w.Create(ctx, &calc)).To(Succeed())
			}, nil)
			expectNoCalcStatus(ctx, cfg.Status, calc.Key())
		})

		Context("Dependency Topologies", func() {
			It("Should hydrate a diamond dependency graph", func(ctx SpecContext) {
				var calcA, calcB, calcC channel.Channel
				_, cfg := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_dia_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					calcB = channel.Channel{
						Name: "hy_dia_b", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_dia_base + 1",
					}
					Expect(w.Create(ctx, &calcB)).To(Succeed())
					calcC = channel.Channel{
						Name: "hy_dia_c", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_dia_base * 2",
					}
					Expect(w.Create(ctx, &calcC)).To(Succeed())
					calcA = channel.Channel{
						Name: "hy_dia_a", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_dia_b + hy_dia_c",
					}
					Expect(w.Create(ctx, &calcA)).To(Succeed())
				}, nil)
				expectNoCalcStatus(ctx, cfg.Status, calcA.Key())
				expectNoCalcStatus(ctx, cfg.Status, calcB.Key())
				expectNoCalcStatus(ctx, cfg.Status, calcC.Key())
			})

			It("Should hydrate a deep chain (4 levels)", func(ctx SpecContext) {
				var c1, c2, c3, c4 channel.Channel
				_, cfg := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_deep_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					c1 = channel.Channel{
						Name: "hy_deep_c1", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_deep_base + 1",
					}
					Expect(w.Create(ctx, &c1)).To(Succeed())
					c2 = channel.Channel{
						Name: "hy_deep_c2", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_deep_c1 + 1",
					}
					Expect(w.Create(ctx, &c2)).To(Succeed())
					c3 = channel.Channel{
						Name: "hy_deep_c3", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_deep_c2 + 1",
					}
					Expect(w.Create(ctx, &c3)).To(Succeed())
					c4 = channel.Channel{
						Name: "hy_deep_c4", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_deep_c3 + 1",
					}
					Expect(w.Create(ctx, &c4)).To(Succeed())
				}, nil)
				expectNoCalcStatus(ctx, cfg.Status, c1.Key())
				expectNoCalcStatus(ctx, cfg.Status, c2.Key())
				expectNoCalcStatus(ctx, cfg.Status, c3.Key())
				expectNoCalcStatus(ctx, cfg.Status, c4.Key())
			})

			It("Should hydrate a fan-out topology", func(ctx SpecContext) {
				var calcs []channel.Channel
				_, cfg := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_fan_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					calcs = []channel.Channel{
						{
							Name: "hy_fan_c1", DataType: telem.Int64T, Virtual: true,
							Expression: "return hy_fan_base + 1",
						},
						{
							Name: "hy_fan_c2", DataType: telem.Int64T, Virtual: true,
							Expression: "return hy_fan_base * 2",
						},
						{
							Name: "hy_fan_c3", DataType: telem.Int64T, Virtual: true,
							Expression: "return hy_fan_base - 1",
						},
					}
					Expect(w.CreateMany(ctx, &calcs)).To(Succeed())
				}, nil)
				expectNoCalcStatus(ctx, cfg.Status, calcs[0].Key())
				expectNoCalcStatus(ctx, cfg.Status, calcs[1].Key())
				expectNoCalcStatus(ctx, cfg.Status, calcs[2].Key())
			})

			It("Should hydrate a fan-in topology", func(ctx SpecContext) {
				var calc channel.Channel
				_, cfg := reopenService(ctx, func(w channel.Writer) {
					bases := []channel.Channel{
						{Name: "hy_fin_b1", DataType: telem.Int64T, Virtual: true},
						{Name: "hy_fin_b2", DataType: telem.Int64T, Virtual: true},
						{Name: "hy_fin_b3", DataType: telem.Int64T, Virtual: true},
					}
					Expect(w.CreateMany(ctx, &bases)).To(Succeed())
					calc = channel.Channel{
						Name: "hy_fin_calc", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_fin_b1 + hy_fin_b2 + hy_fin_b3",
					}
					Expect(w.Create(ctx, &calc)).To(Succeed())
				}, nil)
				expectNoCalcStatus(ctx, cfg.Status, calc.Key())
			})
		})

		Context("DataType Repair", func() {
			It("Should not repair when DataType already matches", func(ctx SpecContext) {
				var calc channel.Channel
				hydrated, _ := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_norep_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					calc = channel.Channel{
						Name: "hy_norep_calc", DataType: telem.Int64T, Virtual: true,
						Expression: "return hy_norep_base + 1",
					}
					Expect(w.Create(ctx, &calc)).To(Succeed())
				}, nil)
				Expect(retrieveDataType(ctx, hydrated, calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair a stale DataType during hydration", func(ctx SpecContext) {
				var calc channel.Channel
				hydrated, _ := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_rep_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					calc = channel.Channel{
						Name:       "hy_rep_calc",
						DataType:   telem.Int64T,
						Virtual:    true,
						Expression: "return hy_rep_base + 1",
					}
					Expect(w.Create(ctx, &calc)).To(Succeed())
				}, func(db *gorp.DB) {
					staleAtRest(ctx, db, calc.Key(), telem.Float32T)
				})
				Expect(retrieveDataType(ctx, hydrated, calc.Key())).To(Equal(telem.Int64T))
			})

			It("Should repair cascaded DataType when a dependent has a lower key than its dependency", func(ctx SpecContext) {
				var calc1, calc2 channel.Channel
				hydrated, _ := reopenService(ctx, func(w channel.Writer) {
					base := channel.Channel{Name: "hy_ooo_base", DataType: telem.Int64T, Virtual: true}
					Expect(w.Create(ctx, &base)).To(Succeed())
					calc1 = channel.Channel{
						Name:       "hy_ooo_c1",
						DataType:   telem.Int64T,
						Virtual:    true,
						Expression: "return hy_ooo_base + 1",
					}
					Expect(w.Create(ctx, &calc1)).To(Succeed())
					calc2 = channel.Channel{
						Name:       "hy_ooo_c2",
						DataType:   telem.Int64T,
						Virtual:    true,
						Expression: "return hy_ooo_c1 + 1",
					}
					Expect(w.Create(ctx, &calc2)).To(Succeed())
					Expect(w.Delete(ctx, calc1.Key(), false)).To(Succeed())
					calc1 = channel.Channel{
						Name:       "hy_ooo_c1",
						DataType:   telem.Int64T,
						Virtual:    true,
						Expression: "return hy_ooo_base + 1",
					}
					Expect(w.Create(ctx, &calc1)).To(Succeed())
					Expect(calc2.Key()).To(BeNumerically("<", calc1.Key()))
				}, func(db *gorp.DB) {
					staleAtRest(ctx, db, calc1.Key(), telem.Float32T)
					staleAtRest(ctx, db, calc2.Key(), telem.Float32T)
				})
				Expect(retrieveDataType(ctx, hydrated, calc1.Key())).To(Equal(telem.Int64T))
				Expect(retrieveDataType(ctx, hydrated, calc2.Key())).To(Equal(telem.Int64T))
			})
		})
	})

	Describe("Reactive Change Handling", func() {

		Context("Creating Channels", func() {
			It("Should inspect a new valid calculated channel", func(ctx SpecContext) {
				base := channel.Channel{Name: "rc_create_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_create_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_create_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should set error status for a newly-invalidated calculated channel", func(ctx SpecContext) {
				calc := createBrokenCalc(ctx, channelWriter, "rc_create_bad", "rc_create_bad_dep")
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should handle incrementally building a chain", func(ctx SpecContext) {
				base := channel.Channel{Name: "rc_chain_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc1 := channel.Channel{
					Name: "rc_chain_c1", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				calc2 := channel.Channel{
					Name: "rc_chain_c2", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_chain_c1 * 2",
				}
				Expect(channelWriter.Create(ctx, &calc2)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())
			})

			It("Should process a batch CreateMany in a single handleChanges call", func(ctx SpecContext) {
				base := channel.Channel{Name: "rc_batch_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calcs := []channel.Channel{
					{Name: "rc_batch_c1", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base + 1"},
					{Name: "rc_batch_c2", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base * 2"},
					{Name: "rc_batch_c3", DataType: telem.Int64T, Virtual: true, Expression: "return rc_batch_base - 1"},
				}
				Expect(channelWriter.CreateMany(ctx, &calcs)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calcs[0].Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calcs[1].Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calcs[2].Key())
			})
		})

		Context("Deleting Channels", func() {
			It("Should set error status when a base dependency is deleted", func(ctx SpecContext) {
				base := channel.Channel{Name: "rc_del_base", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_del_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_del_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Deleting the base dependency")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).
					To(Succeed())
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should set error on downstream calc when intermediate calc is deleted", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())

				By("Deleting the intermediate calculated channel")
				Expect(channelWriter.Delete(ctx, calc1.Key(), false)).To(Succeed())
				expectCalcStatus(ctx, svcCfg.Status, calc2.Key())
			})

			It("Should leave upstream unaffected when a leaf calc is deleted", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
			})

			It("Should not cascade invalidity through reconciliation in a diamond", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, calcA.Key())

				By("Deleting the shared base dependency")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())

				By("Verifying calc_b and calc_c get error statuses")
				expectCalcStatus(ctx, svcCfg.Status, calcB.Key())
				expectCalcStatus(ctx, svcCfg.Status, calcC.Key())

				By("Verifying calc_a does NOT get error status because " +
					"reconciliation continues without enqueueing dependents on error")
				expectNoCalcStatus(ctx, svcCfg.Status, calcA.Key())
			})
		})

		Context("Updating Channels", func() {
			It("Should update deps when expression changes to use a different base", func(ctx SpecContext) {
				base1 := channel.Channel{Name: "rc_upd_b1", DataType: telem.Int64T, Virtual: true}
				base2 := channel.Channel{Name: "rc_upd_b2", DataType: telem.Int64T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base1)).To(Succeed())
				Expect(channelWriter.Create(ctx, &base2)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_upd_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_b1 + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Updating expression to use a different base")
				calc.Expression = "return rc_upd_b2 * 2"
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Verifying old base deletion does not affect calc")
				Expect(channelWriter.Delete(ctx, base1.Key(), false)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Verifying new base deletion does affect calc")
				Expect(channelWriter.Delete(ctx, base2.Key(), false)).To(Succeed())
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should set error status when a dependency is removed", func(ctx SpecContext) {
				createDep(ctx, channelWriter, "rc_upd_bad_base")
				calc := channel.Channel{
					Name: "rc_upd_bad_calc", DataType: telem.Int64T, Virtual: true,
					Expression: "return rc_upd_bad_base + 1",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Removing the dependency so the expression no longer resolves")
				deleteDep(ctx, channelWriter, "rc_upd_bad_base")
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should clear error status when a broken dependency is restored", func(ctx SpecContext) {
				calc := createBrokenCalc(ctx, channelWriter, "rc_upd_fix", "rc_fix_missing_dep")
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Recreating the missing dependency")
				createDep(ctx, channelWriter, "rc_fix_missing_dep")
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
			})
		})

		Context("Cascading Reconciliation", func() {
			It("Should not cascade invalidity from reconciliation to further dependents", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())

				By("Deleting the base. calc1 becomes invalid. " +
					"calc2 should NOT get error because reconciliation " +
					"does not enqueue dependents when a node errors")
				Expect(channelWriter.Delete(ctx, base.Key(), false)).To(Succeed())
				expectCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())
			})

			It("Should cascade deletion through a long chain", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, c1.Key())

				By("c3 depends on c2 which is gone, so it gets error")
				expectCalcStatus(ctx, svcCfg.Status, c3.Key())

				By("c4 does not get error because reconciliation does not " +
					"cascade invalidity from c3's failure")
				expectNoCalcStatus(ctx, svcCfg.Status, c4.Key())
			})

			It("Should re-inspect dependents when a calculated channel is updated", func(ctx SpecContext) {
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
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())

				By("Updating calc1 expression - calc2 should be re-inspected via BFS")
				calc1.Expression = "return rc_reins_base + 100"
				Expect(channelWriter.Create(ctx, &calc1)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())
			})
		})

		Context("DataType Persistence", func() {
			It("Should persist DataType changes to the DB when a dependency type changes", func(ctx SpecContext) {
				By("Creating a base channel and a calc that depends on it")
				base := channel.Channel{Name: "rc_dtp_base", DataType: telem.Float32T, Virtual: true}
				Expect(channelWriter.Create(ctx, &base)).To(Succeed())
				calc := channel.Channel{
					Name: "rc_dtp_calc", DataType: telem.Float32T, Virtual: true,
					Expression: "return rc_dtp_base * 2",
				}
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Updating the calc expression to return a different type")
				calc.Expression = "return i64(rc_dtp_base)"
				calc.DataType = telem.Int64T
				Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Verifying the DataType was persisted to the DB")
				Eventually(func() telem.DataType {
					return retrieveDataType(ctx, svc, calc.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})

			It("Should persist cascaded DataType changes through a chain", func(ctx SpecContext) {
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
					return retrieveDataType(ctx, svc, calc1.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))

				By("Verifying calc2 DataType was also updated via cascade")
				Eventually(func() telem.DataType {
					return retrieveDataType(ctx, svc, calc2.Key())
				}, 2*time.Second, 10*time.Millisecond).Should(Equal(telem.Int64T))
			})
		})

		Context("Unresolved Name Auto-Heal", func() {
			It("Should auto-fix a broken calc when its deleted dependency is recreated", func(ctx SpecContext) {
				calc := createBrokenCalc(ctx, channelWriter, "rc_unres_calc", "rc_unres_missing")
				expectCalcStatus(ctx, svcCfg.Status, calc.Key())

				By("Recreating the previously deleted dependency")
				createDep(ctx, channelWriter, "rc_unres_missing")

				By("Verifying calc is auto-fixed")
				expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
			})

			It("Should auto-fix multiple calcs waiting on the same restored name", func(ctx SpecContext) {
				createDep(ctx, channelWriter, "rc_unres_shared_dep")
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
				deleteDep(ctx, channelWriter, "rc_unres_shared_dep")
				expectCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectCalcStatus(ctx, svcCfg.Status, calc2.Key())

				By("Recreating the shared dependency")
				createDep(ctx, channelWriter, "rc_unres_shared_dep")

				By("Both calcs should auto-heal")
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calc2.Key())
			})

			It("Should auto-fix a chain when a deleted base is recreated", func(ctx SpecContext) {
				calc1 := createBrokenCalc(ctx, channelWriter, "rc_unres_chain_c1", "rc_unres_chain_base")
				expectCalcStatus(ctx, svcCfg.Status, calc1.Key())

				By("Recreating the deleted base")
				createDep(ctx, channelWriter, "rc_unres_chain_base")

				By("calc1 should auto-heal")
				expectNoCalcStatus(ctx, svcCfg.Status, calc1.Key())
			})
		})

		Context("Multiple Independent Subgraphs", func() {
			It("Should isolate failures to their own subgraph", func(ctx SpecContext) {
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
				expectCalcStatus(ctx, svcCfg.Status, calcA.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, calcB.Key())
			})
		})
	})

	Describe("Status Communication", func() {
		It("Should set status with correct structure and details", func(ctx SpecContext) {
			calc := createBrokenCalc(ctx, channelWriter, "st_detail", "st_missing_detail_dep")

			s := expectCalcStatus(ctx, svcCfg.Status, calc.Key())
			Expect(s.Variant).To(Equal(status.VariantError))
			Expect(s.Message).To(Equal("invalid expression for st_detail"))
			Expect(s.Description).ToNot(BeEmpty())
			Expect(s.Key).To(Equal(channel.OntologyID(calc.Key()).String()))
			Expect(s.Name).To(Equal("st_detail"))
		})

		It("Should clear status when a broken dependency is restored", func(ctx SpecContext) {
			calc := createBrokenCalc(ctx, channelWriter, "st_clear", "st_clear_dep")
			expectCalcStatus(ctx, svcCfg.Status, calc.Key())

			By("Recreating the dependency")
			createDep(ctx, channelWriter, "st_clear_dep")
			expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
		})

		It("Should overwrite status when the error changes", func(ctx SpecContext) {
			createDep(ctx, channelWriter, "st_ow_a")
			createDep(ctx, channelWriter, "st_ow_b")
			calc := channel.Channel{
				Name: "st_overwrite", DataType: telem.Int64T, Virtual: true,
				Expression: "return st_ow_a + st_ow_b",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

			By("Deleting st_ow_a so the calc breaks on a missing st_ow_a")
			deleteDep(ctx, channelWriter, "st_ow_a")
			s1 := expectCalcStatus(ctx, svcCfg.Status, calc.Key())

			By("Restoring st_ow_a and deleting st_ow_b so the error becomes a different one")
			createDep(ctx, channelWriter, "st_ow_a")
			deleteDep(ctx, channelWriter, "st_ow_b")
			var s2 channel.CalculationStatus
			Eventually(func() bool {
				s, ok := fetchCalcStatus(ctx, svcCfg.Status, calc.Key())
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
			expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
		})
	})

	Describe("Lifecycle", func() {
		It("Should stop reacting to channel changes after the service closes", func(ctx SpecContext) {
			node := mock.NewNode(ctx)
			cfg := serviceConfig(ctx, node)
			closedSvc := MustSucceed(channel.OpenService(ctx, cfg))
			w := closedSvc.NewWriter(nil)
			base := channel.Channel{Name: "lc_disc_base", DataType: telem.Int64T, Virtual: true}
			Expect(w.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name: "lc_disc_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return lc_disc_base + 1",
			}
			Expect(w.Create(ctx, &calc)).To(Succeed())
			expectNoCalcStatus(ctx, cfg.Status, calc.Key())

			By("Closing the service to disconnect the graph observer")
			Expect(closedSvc.Close()).To(Succeed())

			By("Deleting the base at rest should not set error status")
			Expect(gorp.NewDelete[channel.Key, channel.Channel]().
				Where(gorp.MatchKeys[channel.Key, channel.Channel](base.Key())).
				Exec(ctx, node.DB)).To(Succeed())
			Consistently(func() bool {
				_, ok := fetchCalcStatus(ctx, cfg.Status, calc.Key())
				return ok
			}, 250*time.Millisecond, 25*time.Millisecond).Should(BeFalse())
		})
	})

	Describe("Concurrency", func() {
		It("Should handle concurrent channel creation", func(ctx SpecContext) {
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
					Expect(svc.NewWriter(nil).Create(ctx, &calcs[i])).To(Succeed())
				}()
			}
			wg.Wait()
			for i := range n {
				expectNoCalcStatus(ctx, svcCfg.Status, calcs[i].Key())
			}
		})

		It("Should produce a consistent state under concurrent create and delete", func(ctx SpecContext) {
			base := channel.Channel{Name: "cc_race_base", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &base)).To(Succeed())
			stable := channel.Channel{Name: "cc_race_stable", DataType: telem.Int64T, Virtual: true}
			Expect(channelWriter.Create(ctx, &stable)).To(Succeed())
			calc := channel.Channel{
				Name: "cc_race_calc", DataType: telem.Int64T, Virtual: true,
				Expression: "return cc_race_base + 1",
			}
			Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
			expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				Expect(svc.NewWriter(nil).Delete(ctx, base.Key(), false)).To(Succeed())
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
				Expect(svc.NewWriter(nil).Create(ctx, &newCalc)).To(Succeed())
			}()
			wg.Wait()
		})

		It("Should handle rapid sequential updates", func(ctx SpecContext) {
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
			expectNoCalcStatus(ctx, svcCfg.Status, calc.Key())
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
				expectNoCalcStatus(ctx, svcCfg.Status, mid1.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, mid2.Key())
				expectNoCalcStatus(ctx, svcCfg.Status, top.Key())
			})

			It("Should only affect mid1 when base2 is deleted", func(ctx SpecContext) {
				By("Deleting base2 which is only used by mid1")
				Expect(channelWriter.Delete(ctx, base2.Key(), false)).To(Succeed())

				By("mid1 depends on base2 so it gets error")
				expectCalcStatus(ctx, svcCfg.Status, mid1.Key())

				By("mid2 only depends on base1 so it stays valid")
				expectNoCalcStatus(ctx, svcCfg.Status, mid2.Key())

				By("top is not re-inspected because reconciliation " +
					"does not cascade invalidity from mid1")
				expectNoCalcStatus(ctx, svcCfg.Status, top.Key())
			})

			It("Should break top when mid1 is deleted", func(ctx SpecContext) {
				Expect(channelWriter.Delete(ctx, mid1.Key(), false)).To(Succeed())
				expectCalcStatus(ctx, svcCfg.Status, top.Key())
			})
		})

		Context("Long Chain With Mid-Chain Deletion", func() {
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
				expectNoCalcStatus(ctx, svcCfg.Status, c1.Key())

				By("c3 directly depended on c2 and gets error")
				expectCalcStatus(ctx, svcCfg.Status, c3.Key())

				By("c4 is not re-inspected because invalidity does not cascade from c3")
				expectNoCalcStatus(ctx, svcCfg.Status, c4.Key())
			})
		})
	})
})
