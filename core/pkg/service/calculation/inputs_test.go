// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	"context"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
)

// openChannels opens a channel service and its status service over node.
func openChannels(
	ctx context.Context,
	node testutil.Node,
	cfgs ...channel.ServiceConfig,
) *channel.Service {
	GinkgoHelper()
	statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: node.DB}))
	return MustOpen(channel.OpenService(ctx, append([]channel.ServiceConfig{{
		Channel: node.Channel,
		TS:      node.Storage.TS,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statuses,
	}}, cfgs...)...))
}

// split returns the keys of the channels of given that have no expression, and an
// unsaved calculation for each one that has.
func split(given []channel.Channel) (channel.Keys, []calculation.Unsaved) {
	var (
		outputs channel.Keys
		unsaved []calculation.Unsaved
	)
	for _, ch := range given {
		if ch.Expression == "" {
			outputs = append(outputs, ch.Key())
			continue
		}
		unsaved = append(unsaved, calculation.Unsaved{
			Output:     ch,
			Expression: ch.Expression,
			Operations: calculation.FromChannelOperations(ch.Operations),
		})
	}
	return outputs, unsaved
}

var _ = Describe("Service", func() {
	var (
		db       *gorp.DB
		channels *channel.Service
		svc      *calculation.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node := testutil.NewNode(ctx)
		db = node.DB
		channels = openChannels(ctx, node)
		svc = MustOpen(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      db,
			Channel: channels,
		}))
	})
	// createChannel creates ch, with its calculation when it has an expression.
	createChannel := func(ctx SpecContext, ch *channel.Channel) {
		GinkgoHelper()
		created := []channel.Channel{*ch}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateChannels(ctx, &created, false)
		})).To(Succeed())
		*ch = created[0]
	}
	Describe("Inputs", func() {
		// write runs f with a writer in a transaction of its own.
		write := func(ctx SpecContext, f func(channel.Writer) error) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return f(channels.NewWriter(tx))
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
			createChannel(ctx, &ch)
			return ch
		}
		inputs := func(ctx SpecContext, given ...channel.Channel) channel.Keys {
			GinkgoHelper()
			outputs, unsaved := split(given)
			return MustSucceed(svc.Inputs(ctx, db, outputs, unsaved))
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
			base := create(ctx, "")
			reset := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.BooleanT,
				Virtual:  true,
			}
			createChannel(ctx, &reset)
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
			createChannel(ctx, &calc)
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
			Expect(channels.NewWriter(tx).Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return " + base.Name + " * 2",
			}
			channels := []channel.Channel{calc}
			outputs, unsaved := split(channels)
			Expect(svc.Inputs(ctx, tx, outputs, unsaved)).
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
			limitSvc := openChannels(ctx, node, channel.ServiceConfig{
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
			limitCalcs := MustOpen(calculation.OpenService(
				ctx,
				calculation.ServiceConfig{DB: node.DB, Channel: limitSvc},
			))
			outputs, unsaved := split(channels)
			Expect(limitCalcs.Inputs(ctx, node.DB, outputs, unsaved)).
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
})
