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
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

var _ = Describe("Service", func() {
	Describe("ShouldValidateNames", func() {
		It("Should report true by default", func(ctx SpecContext) {
			Expect(svc.ShouldValidateNames()).To(BeTrue())
		})
		It(
			"Should report false when name validation is disabled",
			func(ctx SpecContext) {
				noValidateSvc := openService(
					ctx,
					testutil.NewNode(ctx),
					channel.ServiceConfig{
						ValidateNames: new(false),
					},
				)
				Expect(noValidateSvc.ShouldValidateNames()).To(BeFalse())
			},
		)
	})

	Describe("NewArcSymbolResolver", func() {
		It(
			"Should resolve a channel created through the service by name",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     "arc_resolver_by_name",
					DataType: telem.Float32T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				sym := MustSucceed(svc.NewArcSymbolResolver(db).Resolve(ctx, ch.Name))
				Expect(sym.Name).To(Equal(ch.Name))
				Expect(sym.ID).To(Equal(int(ch.Key())))
			},
		)
		It("Should fuzzy-search channels by name", func(ctx SpecContext) {
			ch := channel.Channel{
				Name:     "catalina",
				DataType: telem.Float32T,
				Virtual:  true,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(svc.NewArcSymbolResolver(db).Search(ctx, ch.Name)).
				To(ContainElement(HaveField("Name", ch.Name)))
		})
	})

	Describe("CalculationInputs", func() {
		// write runs f with a writer in a transaction of its own.
		write := func(ctx SpecContext, f func(channel.Writer) error) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return f(svc.NewWriter(tx))
			})).To(Succeed())
		}
		create := func(ctx SpecContext, expression string) channel.Channel {
			GinkgoHelper()
			ch := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: expression,
			}
			write(ctx, func(w channel.Writer) error { return w.Create(ctx, &ch) })
			return ch
		}
		inputs := func(ctx SpecContext, channels ...channel.Channel) channel.Keys {
			GinkgoHelper()
			return MustSucceed(svc.CalculationInputs(ctx, db, channels))
		}
		It(
			"Should return the channels a calculation reads through other calculations",
			func(ctx SpecContext) {
				base := create(ctx, "")
				inner := create(ctx, "return "+base.Name+" * 2")
				outer := create(ctx, "return "+inner.Name+" + 1")
				Expect(inputs(ctx, outer)).To(ConsistOf(inner.Key(), base.Key()))
			},
		)
		It("Should return the channels that reset its operations", func(
			ctx SpecContext,
		) {
			base, reset := create(ctx, ""), create(ctx, "")
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return " + base.Name,
				Operations: []channel.Operation{{
					Type:         channel.OperationTypeMax,
					ResetChannel: reset.Key(),
				}},
			}
			write(ctx, func(w channel.Writer) error { return w.Create(ctx, &calc) })
			Expect(inputs(ctx, calc)).To(ConsistOf(base.Key(), reset.Key()))
		})
		It("Should return a given channel that another given calculation reads", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			calc := create(ctx, "return "+base.Name+" * 2")
			Expect(inputs(ctx, calc, base)).To(ConsistOf(base.Key()))
		})
		It(
			"Should walk a stored calculation given as a channel that is not calculated",
			func(ctx SpecContext) {
				base := create(ctx, "")
				inner := create(ctx, "return "+base.Name+" * 2")
				outer := create(ctx, "return "+inner.Name+" + 1")
				plain := inner
				plain.Expression = ""
				Expect(inputs(ctx, plain, outer)).
					To(ConsistOf(inner.Key(), base.Key()))
			},
		)
		It("Should return the inputs of a calculation that is not stored", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return " + base.Name + " * 2",
			}
			Expect(inputs(ctx, calc)).To(ConsistOf(base.Key()))
		})
		It("Should read through the given snapshot", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			base := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(svc.NewWriter(tx).Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return " + base.Name + " * 2",
			}
			channels := []channel.Channel{calc}
			Expect(svc.CalculationInputs(ctx, tx, channels)).
				To(ConsistOf(base.Key()))
			Expect(inputs(ctx, calc)).To(BeEmpty())
		})
		It("Should return the error of a channel read that fails", func(
			ctx SpecContext,
		) {
			var (
				errLimit = errors.New("channel limit exceeded")
				limited  atomic.Bool
			)
			node := testutil.NewNode(ctx)
			limitSvc := openService(ctx, node, channel.ServiceConfig{
				Limit: func(types.Uint20) error {
					if limited.Load() {
						return errLimit
					}
					return nil
				},
			})
			idx := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			data := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.Int64T,
				Leaseholder: 1,
			}
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return " + data.Name + " * 2",
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				w := limitSvc.NewWriter(tx)
				if err := w.Create(ctx, &idx); err != nil {
					return err
				}
				data.LocalIndex = idx.LocalKey
				if err := w.Create(ctx, &data); err != nil {
					return err
				}
				return w.Create(ctx, &calc)
			})).To(Succeed())
			limited.Store(true)
			channels := []channel.Channel{calc}
			Expect(limitSvc.CalculationInputs(ctx, node.DB, channels)).
				Error().To(MatchError(errLimit))
		})
		DescribeTable(
			"Should return nothing",
			func(ctx SpecContext, channels func(SpecContext) []channel.Channel) {
				Expect(inputs(ctx, channels(ctx)...)).To(BeEmpty())
			},
			Entry("for a channel that is not calculated", func(
				ctx SpecContext,
			) []channel.Channel {
				return []channel.Channel{create(ctx, "")}
			}),
			Entry("for a given channel that is not stored", func(
				ctx SpecContext,
			) []channel.Channel {
				base := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return 1",
				}
				calc := channel.Channel{
					Name:       UniqueChannelName(),
					DataType:   telem.Int64T,
					Virtual:    true,
					Expression: "return " + base.Name + " * 2",
				}
				return []channel.Channel{base, calc}
			}),
			Entry("for a calculation that fails analysis", func(
				ctx SpecContext,
			) []channel.Channel {
				GinkgoHelper()
				base := create(ctx, "")
				calc := create(ctx, "return "+base.Name+" * 2")
				write(ctx, func(w channel.Writer) error {
					return w.Delete(ctx, base.Key(), false)
				})
				return []channel.Channel{calc}
			}),
		)
	})

	Describe("NewRetrieve", func() {
		It(
			"Should retrieve a channel created through the service",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float64T,
					Virtual:  true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				var retrieved channel.Channel
				Expect(svc.NewRetrieve().Where(
					channel.MatchKeys(ch.Key()),
				).Entry(&retrieved).Exec(ctx, db)).To(Succeed())
				Expect(retrieved.Name).To(Equal(ch.Name))
			},
		)
	})

	Describe("CountLimited", func() {
		It(
			"Should count the committed channels that are neither internal nor virtual",
			func(ctx SpecContext) {
				node := testutil.NewNode(ctx)
				countSvc := openService(ctx, node)
				base := countSvc.CountLimited()
				idx := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Leaseholder: 1,
				}
				data := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.Float64T,
					Leaseholder: 1,
				}
				virtual := channel.Channel{
					Name:     UniqueChannelName(),
					DataType: telem.Float64T,
					Virtual:  true,
				}
				internal := channel.Channel{
					Name:        UniqueChannelName(),
					DataType:    telem.TimestampT,
					IsIndex:     true,
					Internal:    true,
					Leaseholder: 1,
				}
				Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
					w := countSvc.NewWriter(tx)
					if err := w.Create(ctx, &idx); err != nil {
						return err
					}
					data.LocalIndex = idx.LocalKey
					if err := w.Create(ctx, &data); err != nil {
						return err
					}
					if err := w.Create(ctx, &virtual); err != nil {
						return err
					}
					if err := w.Create(ctx, &internal); err != nil {
						return err
					}
					Expect(countSvc.CountLimited()).To(Equal(base))
					return nil
				})).To(Succeed())
				Expect(countSvc.CountLimited()).To(Equal(base + 2))
				Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return countSvc.NewWriter(tx).Delete(ctx, data.Key(), false)
				})).To(Succeed())
				Expect(countSvc.CountLimited()).To(Equal(base + 1))
			},
		)
	})

	Describe("Observe", func() {
		It("Should return a non-nil observable", func(ctx SpecContext) {
			obs := svc.Observe()
			Expect(obs).ToNot(BeNil())
		})
		It("Should notify when a channel is created", func(ctx SpecContext) {
			var called atomic.Bool
			disconnect := svc.Observe().OnChange(
				func(context.Context, gorp.TxReader[channel.Key, channel.Channel]) {
					called.Store(true)
				},
			)
			defer disconnect()
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Eventually(called.Load).Should(BeTrue())
		})
	})

	Describe("Changes", func() {
		It("Should return the channel that a commit creates", func(ctx SpecContext) {
			var names []string
			DeferCleanup(
				db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
					for ch := range svc.Changes(ctx, c) {
						names = append(names, ch.Value.Name)
					}
				}),
			)
			ch := channel.Channel{
				Name:        UniqueChannelName(),
				DataType:    telem.TimestampT,
				IsIndex:     true,
				Leaseholder: 1,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &ch)
			})).To(Succeed())
			Expect(names).To(Equal([]string{ch.Name}))
		})
	})

	Describe("NewWriter", func() {
		It(
			"Should create a writer that infers types for calculated channels",
			func(ctx SpecContext) {
				ch := channel.Channel{
					Name:       UniqueChannelName(),
					Expression: "return 1 + 1",
					Virtual:    true,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).Create(ctx, &ch)
				})).To(Succeed())
				Expect(ch.DataType).To(Equal(telem.Int64T))
			},
		)
	})
})
