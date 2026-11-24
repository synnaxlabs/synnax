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
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	acttestutil "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Primary breaker", func() {
	var (
		node     disttestutil.Node
		channels *channel.Service
		svc      *calculation.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node = disttestutil.OpenNode(ctx)
		DeferCleanup(func() { Expect(node.Close()).To(Succeed()) })
		statuses := MustSucceed(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		DeferCleanup(func() { Expect(statuses.Close()).To(Succeed()) })
		channels = MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
			Framer:  node.Framer,
			Status:  statuses,
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
	})
	createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createVirtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		return createChannel(ctx, channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
		})
	}
	create := func(ctx SpecContext, c *calculation.Calculation) error {
		return node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, c)
		})
	}
	retrieve := func(ctx SpecContext, key calculation.Key) calculation.Calculation {
		GinkgoHelper()
		var c calculation.Calculation
		Expect(svc.NewRetrieve().
			Where(calculation.MatchKeys(key)).
			Entry(&c).
			Exec(ctx, node.DB)).To(Succeed())
		return c
	}
	apply := func(
		ctx SpecContext,
		key calculation.Key,
		acts ...calculation.Action,
	) ([]calculation.Action, error) {
		var computed []calculation.Action
		err := node.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			computed, err = svc.NewWriter(tx).Apply(ctx, key, acts...)
			return err
		})
		return computed, err
	}

	It("Should create a calculation that reads a stored channel created in the same "+
		"transaction", func(ctx SpecContext) {
		out := createVirtual(ctx, "brk_same_tx_out")
		c := calculation.Calculation{
			Output:     out.Key(),
			Expression: "return brk_same_tx_data * 2",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			cw := channels.NewWriter(tx)
			index := channel.Channel{
				Name:     "brk_same_tx_time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			}
			if err := cw.Create(ctx, &index); err != nil {
				return err
			}
			data := channel.Channel{
				Name:       "brk_same_tx_data",
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			}
			if err := cw.Create(ctx, &data); err != nil {
				return err
			}
			return svc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		Expect(retrieve(ctx, c.Key).Primary).ToNot(BeNil())
	})

	It("Should broadcast the primary it stores when the actions clear it", func(
		ctx SpecContext,
	) {
		a := createVirtual(ctx, "brk_clear_a")
		createVirtual(ctx, "brk_clear_b")
		c := calculation.Calculation{
			Output:     createVirtual(ctx, "brk_clear_out").Key(),
			Expression: "return brk_clear_a + brk_clear_b",
		}
		Expect(create(ctx, &c)).To(Succeed())
		Expect(c.Primary).To(HaveValue(Equal(a.Key())))
		rec := &acttestutil.Recorder[calculation.Key, calculation.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		clear := calculation.NewSetAction(calculation.SetPayload{
			Path:  []string{"primary"},
			Value: nil,
		})
		computed := MustSucceed(apply(ctx, c.Key, clear))
		stored := retrieve(ctx, c.Key)
		Expect(stored.Primary).To(HaveValue(Equal(a.Key())))
		// The sender applies its actions and the computed ones to its copy.
		sender := MustSucceed(calculation.Reduce(
			c,
			slices.Concat([]calculation.Action{clear}, computed)...,
		))
		// Another client applies each broadcast batch, minus the Core-only reads.
		Eventually(rec.Snapshot).ShouldNot(BeEmpty())
		cache := c
		for _, batch := range rec.Snapshot() {
			if batch.Key != c.Key {
				continue
			}
			for _, act := range batch.Actions {
				if act.Set != nil && slices.Equal(act.Set.Path, []string{"reads"}) {
					continue
				}
				cache = MustSucceed(calculation.Reduce(cache, act))
			}
		}
		Expect(cache.Primary).To(Equal(stored.Primary), "the broadcast cache")
		Expect(sender.Primary).To(Equal(stored.Primary), "the sender's copy")
	})

	It("Should fail a save whose actions set the held primary while the new "+
		"expression drops it", func(ctx SpecContext) {
		a := createVirtual(ctx, "brk_reset_a")
		createVirtual(ctx, "brk_reset_b")
		c := calculation.Calculation{
			Output:     createVirtual(ctx, "brk_reset_out").Key(),
			Expression: "return brk_reset_a + brk_reset_b",
		}
		Expect(create(ctx, &c)).To(Succeed())
		Expect(c.Primary).To(HaveValue(Equal(a.Key())))
		_, err := apply(
			ctx,
			c.Key,
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"expression"},
				Value: "return brk_reset_b",
			}),
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"primary"},
				Value: a.Key(),
			}),
		)
		Expect(err).To(MatchError(validate.ErrValidation))
	})

	It("Should pick the first read when its index holds no samples", func(
		ctx SpecContext,
	) {
		index := createChannel(ctx, channel.Channel{
			Name:     "brk_empty_time",
			DataType: telem.TimestampT,
			IsIndex:  true,
		})
		empty := createChannel(ctx, channel.Channel{
			Name:       "brk_empty",
			DataType:   telem.Float32T,
			LocalIndex: index.LocalKey,
		})
		createVirtual(ctx, "brk_empty_v")
		c := calculation.Calculation{
			Output:     createVirtual(ctx, "brk_empty_out").Key(),
			Expression: "return brk_empty + brk_empty_v",
		}
		Expect(create(ctx, &c)).To(Succeed())
		Expect(c.Primary).To(HaveValue(Equal(empty.Key())))
	})
})
