// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"fmt"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	freightertestutil "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/streamer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculation change stream breaker", func() {
	var client freighter.ClientStream[
		apiframer.StreamerRequest,
		apiframer.StreamerResponse,
	]
	find := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		var ch channel.Channel
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchNames(name)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch
	}
	// serve opens a stream of ch as a new user that may retrieve ch and granted, on a
	// test stream with buffers, and returns the error that the stream ends with.
	serve := func(
		ctx SpecContext,
		ch channel.Channel,
		buffers []int,
		granted ...resource.ID,
	) <-chan error {
		GinkgoHelper()
		u := createUserGranted(
			ctx,
			append(svcframer.ResourceIDs(channel.Keys{ch.Key()}), granted...)...,
		)
		var server apiframer.StreamerStream
		client, server = freightertestutil.NewStreams[
			apiframer.StreamerRequest,
			apiframer.StreamerResponse,
		](ctx, buffers...)
		done := make(chan error, 1)
		go func() { done <- apiSvc.Stream(subjectCtx(ctx, u.ResourceID()), server) }()
		Expect(client.Send(apiframer.StreamerRequest{Keys: channel.Keys{ch.Key()}})).
			To(Succeed())
		Expect(MustSucceed(client.Receive()).Frame.Empty()).To(BeTrue())
		return done
	}
	// open opens a stream of ch as a new user that may retrieve ch and granted, and
	// closes it after the spec.
	open := func(ctx SpecContext, ch channel.Channel, granted ...resource.ID) {
		GinkgoHelper()
		done := serve(ctx, ch, nil, granted...)
		DeferCleanup(func() {
			Expect(client.CloseSend()).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive())
		})
	}
	// samples returns the samples of ch in the next frame.
	samples := func(ch channel.Channel) []string {
		GinkgoHelper()
		res := MustSucceed(client.Receive())
		var out []string
		for _, series := range res.Frame.Get(ch.Key()).Series {
			for sample := range series.Samples() {
				out = append(out, string(sample))
			}
		}
		return out
	}
	// count counts the samples of ch that the stream receives from now on.
	count := func(ch channel.Channel) *atomic.Int64 {
		var n atomic.Int64
		go func() {
			defer GinkgoRecover()
			for {
				res, err := client.Receive()
				if err != nil {
					return
				}
				for _, series := range res.Frame.Get(ch.Key()).Series {
					n.Add(series.Len())
				}
			}
		}()
		return &n
	}
	virtual := func(ctx SpecContext) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	calculate := func(ctx SpecContext, expression string) calculation.Calculation {
		GinkgoHelper()
		output := MustSucceed(creator.Create(ctx, calctestutil.Calculation{
			Name:       UniqueChannelName(),
			Expression: expression,
		}))
		return MustSucceed(creator.CalculationOf(ctx, output.Key()))
	}
	setExpression := func(expression string) calculation.Action {
		return calculation.NewSetAction(calculation.SetPayload{
			Path:  []string{"expression"},
			Value: expression,
		})
	}
	edit := func(ctx SpecContext, c calculation.Calculation, expression string) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.Calculation.NewWriter(tx).Apply(
				ctx,
				c.Key,
				setExpression(expression),
			)
			return err
		})).To(Succeed())
	}
	times := func(ch channel.Channel, n int) string {
		return fmt.Sprintf("return %s * %d", ch.Name, n)
	}
	// links returns the calculation type and each channel that c links.
	links := func(c calculation.Calculation) []resource.ID {
		return append(
			[]resource.ID{{Type: resource.TypeCalculation}},
			svc.Calculation.Channels(c)...,
		)
	}

	It(
		"Should not send an edit that makes a calculation read a channel that the "+
			"subject cannot read when a later edit in the same commit undoes it",
		func(ctx SpecContext) {
			visible, secret := virtual(ctx), virtual(ctx)
			c := calculate(ctx, times(visible, 2))
			ch := find(ctx, "sy_calculation_action_set")
			open(ctx, ch, links(c)...)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := svc.Calculation.NewWriter(tx)
				_, err := w.Apply(ctx, c.Key, setExpression(times(secret, 2)))
				if err != nil {
					return err
				}
				_, err = w.Apply(ctx, c.Key, setExpression(times(visible, 3)))
				return err
			})).To(Succeed())
			received := samples(ch)
			Expect(received).ToNot(BeEmpty())
			for _, s := range received {
				Expect(s).ToNot(ContainSubstring(secret.Name))
			}
		},
		SpecTimeout(30*time.Second),
	)

	DescribeTable(
		"Should check the oldest change that a lagging stream holds against the "+
			"window of change checks",
		func(ctx SpecContext, name string, past bool) {
			in := virtual(ctx)
			c := calculate(ctx, times(in, 2))
			ch := find(ctx, name)
			// With no response buffer, the stream checks the first change and holds it
			// for the spec, so the second change is the oldest that it checks late.
			done := serve(ctx, ch, []int{1, 0}, links(c)...)
			// The window holds the frames that the relay, a streamer, and its reader
			// buffer, and the 300 frames of the request pipe of a signals feed.
			edits := relayBuffer + streamer.BufferSize + 300 + 1
			if past {
				edits++
			}
			for i := range edits {
				edit(ctx, c, times(in, i+3))
			}
			Expect(samples(ch)).To(HaveLen(1))
			if past {
				Eventually(done).WithTimeout(10 * time.Second).Should(Receive(
					MatchError("[signals] - the check of a change is gone"),
				))
				Expect(client.CloseSend()).To(Succeed())
				return
			}
			// A stream that fails sends nothing more, so the read is bounded.
			next := make(chan []string, 1)
			go func() {
				defer GinkgoRecover()
				next <- samples(ch)
			}()
			Eventually(next).WithTimeout(10 * time.Second).Should(Receive(HaveLen(1)))
			count(ch)
			Expect(client.CloseSend()).To(Succeed())
			Eventually(done).WithTimeout(5 * time.Second).Should(Receive(BeNil()))
		},
		Entry(
			"a calculation set inside the window",
			"sy_calculation_set",
			false,
			SpecTimeout(2*time.Minute),
		),
		Entry(
			"a calculation set past the window",
			"sy_calculation_set",
			true,
			SpecTimeout(2*time.Minute),
		),
		Entry(
			"a calculation action inside the window",
			"sy_calculation_action_set",
			false,
			SpecTimeout(2*time.Minute),
		),
		Entry(
			"a calculation action past the window",
			"sy_calculation_action_set",
			true,
			SpecTimeout(2*time.Minute),
		),
	)

	It(
		"Should send the delete of a calculation only to a subject that may "+
			"retrieve it",
		func(ctx SpecContext) {
			visible, secret := virtual(ctx), virtual(ctx)
			readable := calculate(ctx, times(visible, 2))
			denied := calculate(ctx, times(secret, 2))
			ch := find(ctx, "sy_calculation_delete")
			open(ctx, ch, links(readable)...)
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Calculation.NewWriter(tx).
					Delete(ctx, readable.Key, denied.Key)
			})).To(Succeed())
			res := MustSucceed(client.Receive())
			var keys []uuid.UUID
			for _, series := range res.Frame.Get(ch.Key()).Series {
				for sample := range series.Samples() {
					keys = append(keys, uuid.UUID(sample))
				}
			}
			Expect(keys).To(Equal([]uuid.UUID{readable.Key}))
		},
		SpecTimeout(30*time.Second),
	)

	It("Should send each label set to a stream that lags 280 frames", func(
		ctx SpecContext,
	) {
		const creates = 280
		ch := find(ctx, "sy_label_set")
		open(ctx, ch, resource.ID{Type: resource.TypeLabel})
		for range creates {
			l := label.Label{Key: uuid.New(), Name: "lag", Color: labelColor}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).Create(ctx, &l)
			})).To(Succeed())
		}
		n := count(ch)
		Eventually(n.Load).
			WithTimeout(10 * time.Second).
			Should(BeEquivalentTo(creates))
	}, SpecTimeout(60*time.Second))

	DescribeTable(
		"Should send each change to a subject that reads the calculation when its "+
			"stream lags 280 frames",
		func(ctx SpecContext, name string) {
			const edits = 280
			in := virtual(ctx)
			c := calculate(ctx, times(in, 2))
			ch := find(ctx, name)
			open(ctx, ch, links(c)...)
			for i := range edits {
				edit(ctx, c, times(in, i+3))
			}
			n := count(ch)
			Eventually(n.Load).
				WithTimeout(10 * time.Second).
				Should(BeEquivalentTo(edits))
		},
		Entry("a calculation set", "sy_calculation_set", SpecTimeout(60*time.Second)),
		Entry(
			"a calculation action",
			"sy_calculation_action_set",
			SpecTimeout(60*time.Second),
		),
	)
})
