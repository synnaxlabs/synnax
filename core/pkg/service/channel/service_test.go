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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("ShouldValidateNames", func() {
		It("Should report true by default", func() {
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
					if err := w.Declare(ctx, &internal); err != nil {
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
		It("Should return a non-nil observable", func() {
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
					for ch := range svc.Changes(c) {
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
