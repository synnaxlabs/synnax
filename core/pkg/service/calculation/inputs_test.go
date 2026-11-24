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
			Framer:  node.Framer,
		}))
	})
	// createChannel creates ch, and the calculation of expression and ops that writes
	// it when expression is not empty.
	createChannel := func(
		ctx SpecContext,
		ch *channel.Channel,
		expression string,
		ops ...calculation.Operation,
	) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if expression == "" {
				return channels.NewWriter(tx).Create(ctx, ch)
			}
			c := calculation.Calculation{Expression: expression, Operations: ops}
			created, err := svc.NewWriter(tx).CreateOutput(ctx, &c, ch.Name, nil)
			if err == nil {
				*ch = created[0]
			}
			return err
		})).To(Succeed())
	}
	// unsaved returns a calculation of expression whose output is not stored.
	unsaved := func(expression string) calculation.Unsaved {
		return calculation.Unsaved{
			Output: channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			},
			Expression: expression,
		}
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
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
			createChannel(ctx, &ch, expression)
			return ch
		}
		inputs := func(
			ctx SpecContext,
			outputs channel.Keys,
			unsaved ...calculation.Unsaved,
		) channel.Keys {
			GinkgoHelper()
			return MustSucceed(svc.Inputs(ctx, db, outputs, unsaved))
		}
		keys := channel.KeysFromChannels
		It(
			"Should return the channels a calculation reads through other calculations",
			func(ctx SpecContext) {
				base := create(ctx, "")
				inner := create(ctx, "return "+base.Name+" * 2")
				outer := create(ctx, "return "+inner.Name+" + 1")
				Expect(inputs(ctx, keys([]channel.Channel{outer}))).
					To(ConsistOf(inner.Key(), base.Key()))
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
			createChannel(ctx, &reset, "")
			calc := channel.Channel{Name: UniqueChannelName(), Virtual: true}
			createChannel(ctx, &calc, "return "+base.Name, calculation.Operation{
				Variant: calculation.MaxOperation{
					Window: calculation.Window{Reset: new(reset.Key())},
				},
			})
			Expect(inputs(ctx, keys([]channel.Channel{calc}))).
				To(ConsistOf(base.Key(), reset.Key()))
		})
		It("Should return a given channel that another given calculation reads", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			calc := create(ctx, "return "+base.Name+" * 2")
			Expect(inputs(ctx, keys([]channel.Channel{calc, base}))).
				To(ConsistOf(base.Key()))
		})
		It(
			"Should return a given calculation that another given calculation reads",
			func(ctx SpecContext) {
				base := create(ctx, "")
				inner := create(ctx, "return "+base.Name+" * 2")
				outer := create(ctx, "return "+inner.Name+" + 1")
				Expect(inputs(ctx, keys([]channel.Channel{inner, outer}))).
					To(ConsistOf(inner.Key(), base.Key()))
			},
		)
		It("Should follow the current expression of a calculation whose input is gone",
			func(ctx SpecContext) {
				kept := create(ctx, "")
				gone := create(ctx, "")
				calc := create(ctx, "return "+kept.Name+" + "+gone.Name)
				write(ctx, func(w channel.Writer) error {
					return w.Delete(ctx, gone.Key(), false)
				})
				Expect(inputs(ctx, keys([]channel.Channel{calc}))).
					To(ConsistOf(kept.Key()))
			},
		)
		It("Should read the reads of a compiled calculation without analysis", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			calc := unsaved("return a_channel_that_does_not_exist")
			calc.Compiled = true
			calc.Reads = channel.Keys{base.Key()}
			Expect(inputs(ctx, nil, calc)).To(ConsistOf(base.Key()))
		})
		It("Should return the inputs of a calculation that is not stored", func(
			ctx SpecContext,
		) {
			base := create(ctx, "")
			Expect(inputs(ctx, nil, unsaved("return "+base.Name+" * 2"))).
				To(ConsistOf(base.Key()))
		})
		It("Should read through the given snapshot", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			base := channel.Channel{
				Name:     UniqueChannelName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}
			Expect(channels.NewWriter(tx).Create(ctx, &base)).To(Succeed())
			calc := unsaved("return " + base.Name + " * 2")
			Expect(svc.Inputs(ctx, tx, nil, []calculation.Unsaved{calc})).
				To(ConsistOf(base.Key()))
			Expect(inputs(ctx, nil, calc)).To(BeEmpty())
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
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				w := limitSvc.NewWriter(tx)
				if err := w.Create(ctx, &idx); err != nil {
					return err
				}
				data.LocalIndex = idx.LocalKey
				return w.Create(ctx, &data)
			})).To(Succeed())
			limited.Store(true)
			limitCalcs := MustOpen(calculation.OpenService(
				ctx,
				calculation.ServiceConfig{
					DB:      node.DB,
					Channel: limitSvc,
					Framer:  node.Framer,
				},
			))
			Expect(limitCalcs.Inputs(
				ctx,
				node.DB,
				nil,
				[]calculation.Unsaved{unsaved("return " + data.Name + " * 2")},
			)).Error().To(MatchError(errLimit))
		})
		DescribeTable(
			"Should return nothing",
			func(
				ctx SpecContext,
				given func(SpecContext) (channel.Keys, []calculation.Unsaved),
			) {
				outputs, calcs := given(ctx)
				Expect(inputs(ctx, outputs, calcs...)).To(BeEmpty())
			},
			Entry("for a channel that is not calculated", func(
				ctx SpecContext,
			) (channel.Keys, []calculation.Unsaved) {
				return channel.Keys{create(ctx, "").Key()}, nil
			}),
			Entry("for a given channel that is not stored", func(
				ctx SpecContext,
			) (channel.Keys, []calculation.Unsaved) {
				base := unsaved("return 1")
				calc := unsaved("return " + base.Output.Name + " * 2")
				return nil, []calculation.Unsaved{base, calc}
			}),
			Entry("for a calculation that fails analysis", func(
				ctx SpecContext,
			) (channel.Keys, []calculation.Unsaved) {
				GinkgoHelper()
				base := create(ctx, "")
				calc := create(ctx, "return "+base.Name+" * 2")
				write(ctx, func(w channel.Writer) error {
					return w.Delete(ctx, base.Key(), false)
				})
				return channel.Keys{calc.Key()}, nil
			}),
		)
	})
})
