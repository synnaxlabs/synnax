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
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Access", func() {
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
			AnnounceLinked: func(
				ctx context.Context,
				tx gorp.Tx,
				w channel.Writer,
				keys ...channel.Key,
			) error {
				return svc.AnnounceOutputs(ctx, tx, w, keys...)
			},
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
			Framer:  node.Framer,
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
	})

	virtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	// calculate creates a calculation that doubles input.
	calculate := func(ctx SpecContext, input channel.Channel) calculation.Calculation {
		GinkgoHelper()
		out := virtual(ctx, input.Name+"_doubled")
		c := calculation.Calculation{
			Output:     out.Key(),
			Expression: fmt.Sprintf("return %s * 2", input.Name),
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		return c
	}
	// commitOf returns the commit that write makes.
	commitOf := func(ctx SpecContext, write func(gorp.Tx) error) gorp.Commit {
		GinkgoHelper()
		var commit gorp.Commit
		disconnect := node.DB.Observe().
			OnChange(func(_ context.Context, c gorp.Commit) {
				commit = c
			})
		defer disconnect()
		Expect(node.DB.WithTx(ctx, write)).To(Succeed())
		return commit
	}
	setExpression := func(
		ctx context.Context,
		tx gorp.Tx,
		c calculation.Calculation,
		input channel.Channel,
	) error {
		_, err := svc.NewWriter(tx).Apply(
			ctx,
			c.Key,
			calculation.NewSetAction(calculation.SetPayload{
				Path:  []string{"expression"},
				Value: fmt.Sprintf("return %s * 3", input.Name),
			}),
		)
		return err
	}
	// except returns a check that admits every resource but refused.
	except := func(refused ...resource.ID) func(resource.ID) bool {
		return func(id resource.ID) bool { return !slices.Contains(refused, id) }
	}

	Describe("Denied", func() {
		It("Should refuse a subject that cannot retrieve the calculation", func(
			ctx SpecContext,
		) {
			c := calculate(ctx, virtual(ctx, "denied_calc"))
			id := calculation.ResourceID(c.Key)
			denied, ok := svc.Denied(c, except(id))
			Expect(ok).To(BeTrue())
			Expect(denied).To(Equal(id))
		})

		It("Should return the first channel that the subject cannot retrieve", func(
			ctx SpecContext,
		) {
			input := virtual(ctx, "denied_input")
			c := calculate(ctx, input)
			id := channel.ResourceID(input.Key())
			denied, ok := svc.Denied(c, except(id))
			Expect(ok).To(BeTrue())
			Expect(denied).To(Equal(id))
		})

		It("Should admit a subject that can retrieve each resource", func(
			ctx SpecContext,
		) {
			c := calculate(ctx, virtual(ctx, "admitted_input"))
			_, ok := svc.Denied(c, except())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Readable", func() {
		It("Should admit only a subject that reads every channel", func(
			ctx SpecContext,
		) {
			input := virtual(ctx, "readable_input")
			check := svc.Readable(calculate(ctx, input))
			Expect(check(except())).To(BeTrue())
			Expect(check(except(channel.ResourceID(input.Key())))).To(BeFalse())
		})
	})

	Describe("BatchChecks", func() {
		It("Should check each batch by the calculation as that batch left it", func(
			ctx SpecContext,
		) {
			open, secret := virtual(ctx, "batch_open"), virtual(ctx, "batch_secret")
			c := calculate(ctx, open)
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				if err := setExpression(ctx, tx, c, secret); err != nil {
					return err
				}
				return setExpression(ctx, tx, c, open)
			})
			batches := svc.Actions().Of(commit)
			Expect(batches).To(HaveLen(2))
			lookup := svc.BatchChecks(commit)
			reader := except(channel.ResourceID(secret.Key()))
			first := MustBeOk(lookup(c.Key, batches[0].Seq))
			last := MustBeOk(lookup(c.Key, batches[1].Seq))
			Expect(first(reader)).To(BeFalse())
			Expect(last(reader)).To(BeTrue())
		})

		It("Should hold no check of a batch that the commit did not attach", func(
			ctx SpecContext,
		) {
			c := calculate(ctx, virtual(ctx, "batch_absent"))
			other := virtual(ctx, "batch_other")
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				return setExpression(ctx, tx, c, other)
			})
			_, ok := svc.BatchChecks(commit)(c.Key, 1<<40)
			Expect(ok).To(BeFalse())
		})
	})

	Describe("ChannelBatchChecks", func() {
		It("Should check the announce of an output by its calculation and leave every "+
			"other batch of the commit to its channel", func(ctx SpecContext) {
			input := virtual(ctx, "announce_input")
			output := virtual(ctx, "announce_output")
			renamed := virtual(ctx, "announce_renamed")
			c := calculation.Calculation{
				Output:     output.Key(),
				Expression: fmt.Sprintf("return %s * 2", input.Name),
			}
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Create(ctx, &c); err != nil {
					return err
				}
				return channels.NewWriter(tx).
					Rename(ctx, renamed.Key(), "announce_renamed_2", false)
			})
			lookup := svc.ChannelBatchChecks(commit)
			var announce, rename func(func(resource.ID) bool) bool
			for _, b := range channels.Actions().Of(commit) {
				check := MustBeOk(lookup(b.Key, b.Seq))
				switch b.Key {
				case output.Key():
					announce = check
				case renamed.Key():
					rename = check
				}
			}
			Expect(announce).ToNot(BeNil())
			Expect(announce(except())).To(BeTrue())
			Expect(announce(except(channel.ResourceID(input.Key())))).To(BeFalse())
			Expect(rename).To(BeNil())
		})

		It("Should leave each batch of a commit with no announce to its channel", func(
			ctx SpecContext,
		) {
			ch := virtual(ctx, "announce_none")
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).
					Rename(ctx, ch.Key(), "announce_none_2", false)
			})
			batches := channels.Actions().Of(commit)
			Expect(batches).To(HaveLen(1))
			check := MustBeOk(svc.ChannelBatchChecks(commit)(ch.Key(), batches[0].Seq))
			Expect(check).To(BeNil())
		})
	})

	Describe("DeleteChecks", func() {
		It("Should admit only a subject that read the deleted calculation", func(
			ctx SpecContext,
		) {
			input := virtual(ctx, "delete_input")
			c := calculate(ctx, input)
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, c.Key)
			})
			check := MustBeOk(svc.DeleteChecks(commit)(c.Key))
			Expect(check(except())).To(BeTrue())
			Expect(check(except(channel.ResourceID(input.Key())))).To(BeFalse())
			Expect(check(except(calculation.ResourceID(c.Key)))).To(BeFalse())
		})

		It("Should hold no check of a calculation that the commit did not delete", func(
			ctx SpecContext,
		) {
			c := calculate(ctx, virtual(ctx, "delete_kept"))
			other := calculate(ctx, virtual(ctx, "delete_other"))
			commit := commitOf(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, other.Key)
			})
			_, ok := svc.DeleteChecks(commit)(c.Key)
			Expect(ok).To(BeFalse())
		})
	})
})
