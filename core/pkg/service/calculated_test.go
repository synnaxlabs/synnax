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

var _ = Describe("Calculated line", Ordered, func() {
	var (
		n      testutil.Node
		l      *service.Layer
		plain  channel.Channel
		output channel.Channel
		calc   calculation.Calculation
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		n = testutil.NewNode(ctx)
		l = MustOpen(svctestutil.OpenLayer(ctx, n))
		plain = channel.Channel{Name: "plain", DataType: telem.Float32T, Virtual: true}
		calc = calculation.Calculation{Expression: "return 1.0"}
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			if err := l.Channel.NewWriter(tx).Create(ctx, &plain); err != nil {
				return err
			}
			outputs, err := l.Calculation.NewWriter(tx).
				CreateOutput(ctx, &calc, "computed", nil)
			if err != nil {
				return err
			}
			output = outputs[0]
			return nil
		})).To(Succeed())
	})
	matched := func(
		ctx SpecContext,
		snap gorp.Snapshot,
		calculated bool,
	) channel.Keys {
		GinkgoHelper()
		var res []channel.Channel
		Expect(l.Channel.NewRetrieve().
			Where(channel.MatchCalculated(calculated)).
			Where(channel.MatchKeys(plain.Key(), output.Key())).
			Entries(&res).
			Exec(ctx, snap)).To(Succeed())
		return channel.KeysFromChannels(res)
	}

	It("Should match the channels that calculations write", func(ctx SpecContext) {
		Expect(matched(ctx, n.DB, true)).To(ConsistOf(output.Key()))
		Expect(matched(ctx, n.DB, false)).To(ConsistOf(plain.Key()))
	})

	It("Should match an output that the transaction writes", func(ctx SpecContext) {
		tx := n.DB.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		c := calculation.Calculation{Output: plain.Key(), Expression: "return 2.0"}
		Expect(l.Calculation.NewWriter(tx).Create(ctx, &c)).To(Succeed())
		Expect(matched(ctx, tx, true)).To(ConsistOf(output.Key(), plain.Key()))
		Expect(matched(ctx, tx, false)).To(BeEmpty())
		Expect(matched(ctx, n.DB, true)).To(ConsistOf(output.Key()))
	})

	It("Should stop matching an output once its calculation is deleted", func(
		ctx SpecContext,
	) {
		Expect(n.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return l.Calculation.NewWriter(tx).Delete(ctx, calc.Key)
		})).To(Succeed())
		Expect(matched(ctx, n.DB, true)).To(BeEmpty())
		Expect(matched(ctx, n.DB, false)).To(ConsistOf(plain.Key(), output.Key()))
	})
})
