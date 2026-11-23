// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculated line breaker", Ordered, func() {
	var (
		n      testutil.Node
		l      *service.Layer
		plain  channel.Channel
		other  channel.Channel
		output channel.Channel
		calc   calculation.Calculation
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = testutil.NewNode(ctx)
		l = MustOpen(svctestutil.OpenLayer(ctx, n))
		virtual := func(name string) channel.Channel {
			return channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		}
		plain, other = virtual("b_plain"), virtual("b_other")
		calc = calculation.Calculation{Expression: "return 1.0"}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			w := l.Channel.NewWriter(tx)
			if err := w.Create(ctx, &plain); err != nil {
				return err
			}
			if err := w.Create(ctx, &other); err != nil {
				return err
			}
			outputs, err := l.Calculation.NewWriter(tx).
				CreateOutput(ctx, &calc, "b_computed", nil)
			if err != nil {
				return err
			}
			output = outputs[0]
			return nil
		})).To(Succeed())
	})
	keys := func() []channel.Key {
		return []channel.Key{plain.Key(), other.Key(), output.Key()}
	}
	matched := func(
		ctx SpecContext,
		snap gorp.Snapshot,
		f channel.Filter,
	) channel.Keys {
		GinkgoHelper()
		var res []channel.Channel
		Expect(l.Channel.NewRetrieve().
			Where(f).
			Where(channel.MatchKeys(keys()...)).
			Entries(&res).
			Exec(ctx, snap)).To(Succeed())
		return channel.KeysFromChannels(res)
	}

	It("Should drop an output that the transaction creates and deletes", func(
		ctx SpecContext,
	) {
		tx := n.DB.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		c := calculation.Calculation{Output: plain.Key(), Expression: "return 2.0"}
		w := l.Calculation.NewWriter(tx)
		Expect(w.Create(ctx, &c)).To(Succeed())
		Expect(matched(ctx, tx, channel.MatchCalculated(true))).
			To(ConsistOf(output.Key(), plain.Key()))
		Expect(w.Delete(ctx, c.Key)).To(Succeed())
		Expect(matched(ctx, tx, channel.MatchCalculated(true))).
			To(ConsistOf(output.Key()))
		Expect(matched(ctx, tx, channel.MatchCalculated(false))).
			To(ConsistOf(plain.Key(), other.Key()))
	})

	It("Should keep an output that one calculation hands to another in a transaction",
		func(ctx SpecContext) {
			tx := n.DB.OpenTx()
			defer func() { Expect(tx.Close()).To(Succeed()) }()
			w := l.Calculation.NewWriter(tx)
			Expect(w.Delete(ctx, calc.Key)).To(Succeed())
			Expect(matched(ctx, tx, channel.MatchCalculated(true))).To(BeEmpty())
			next := calculation.Calculation{
				Output:     output.Key(),
				Expression: "return 3.0",
			}
			Expect(w.Create(ctx, &next)).To(Succeed())
			Expect(matched(ctx, tx, channel.MatchCalculated(true))).
				To(ConsistOf(output.Key()))
			Expect(matched(ctx, tx, channel.MatchCalculated(false))).
				To(ConsistOf(plain.Key(), other.Key()))
			Expect(matched(ctx, n.DB, channel.MatchCalculated(true))).
				To(ConsistOf(output.Key()))
		},
	)

	It("Should compose the line under and, or, and not", func(ctx SpecContext) {
		in, out := channel.MatchCalculated(true), channel.MatchCalculated(false)
		Expect(matched(ctx, n.DB, channel.And(in, out))).To(BeEmpty())
		Expect(matched(ctx, n.DB, channel.Or(in, out))).To(ConsistOf(keys()))
		Expect(matched(ctx, n.DB, channel.Not(in))).
			To(ConsistOf(plain.Key(), other.Key()))
		Expect(matched(ctx, n.DB, channel.Not(out))).To(ConsistOf(output.Key()))
		Expect(matched(ctx, n.DB, channel.Or(
			channel.And(in, channel.MatchKeys(output.Key())),
			channel.Not(channel.Or(in, channel.MatchKeys(plain.Key()))),
		))).To(ConsistOf(output.Key(), other.Key()))
	})

	It("Should give each of many concurrent retrieves its own answer", func(
		ctx SpecContext,
	) {
		cases := []struct {
			f    channel.Filter
			want []channel.Key
		}{
			{channel.MatchCalculated(true), []channel.Key{output.Key()}},
			{channel.MatchCalculated(false), []channel.Key{plain.Key(), other.Key()}},
			{
				channel.Or(
					channel.MatchCalculated(true),
					channel.MatchKeys(plain.Key()),
				),
				[]channel.Key{output.Key(), plain.Key()},
			},
			{channel.Not(channel.MatchCalculated(false)), []channel.Key{output.Key()}},
		}
		var wg sync.WaitGroup
		for g := range 16 {
			wg.Go(func() {
				defer GinkgoRecover()
				for i := range 40 {
					c := cases[(g+i)%len(cases)]
					var res []channel.Channel
					Expect(n.DB.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
						return l.Channel.NewRetrieve().
							Where(c.f).
							Where(channel.MatchKeys(keys()...)).
							Entries(&res).
							Exec(ctx, snap)
					})).To(Succeed())
					Expect(channel.KeysFromChannels(res)).To(ConsistOf(c.want))
				}
			})
		}
		wg.Wait()
	})

	It("Should drop an output whose channel the transaction deletes", func(
		ctx SpecContext,
	) {
		tx := n.DB.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		Expect(l.Channel.NewWriter(tx).Delete(ctx, output.Key(), false)).To(Succeed())
		Expect(l.Calculation.Written(ctx, tx)).ToNot(ContainElement(output.Key()))
		Expect(matched(ctx, tx, channel.MatchCalculated(false))).
			To(ConsistOf(plain.Key(), other.Key()))
	})
})
