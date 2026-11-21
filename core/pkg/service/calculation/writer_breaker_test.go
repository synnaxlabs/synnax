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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer breaker", func() {
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
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
	})
	createVirtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	create := func(ctx SpecContext, c *calculation.Calculation) error {
		return node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, c)
		})
	}
	maxResetOn := func(reset channel.Key) []calculation.Operation {
		return []calculation.Operation{{Variant: calculation.MaxOperation{
			Window: calculation.Window{Reset: new(reset)},
		}}}
	}

	It(
		"Should fail a save whose operation resets on a calculation that reads its "+
			"output",
		func(ctx SpecContext) {
			createVirtual(ctx, "base")
			first := createVirtual(ctx, "first")
			second := calculation.Calculation{
				Output:     createVirtual(ctx, "second").Key(),
				Expression: "return first > 0",
			}
			Expect(create(ctx, &second)).To(Succeed())
			c := calculation.Calculation{
				Output:     first.Key(),
				Expression: "return base * 2",
				Operations: maxResetOn(second.Output),
			}
			Expect(create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("first -> second -> first")),
			))
		},
	)

	It(
		"Should fail the same cycle when the reading calculation is saved last",
		func(ctx SpecContext) {
			createVirtual(ctx, "base")
			// A boolean second leaves the cycle as the one fault of the last save.
			second := channel.Channel{
				Name:     "second",
				DataType: telem.BooleanT,
				Virtual:  true,
			}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Create(ctx, &second)
			})).To(Succeed())
			first := calculation.Calculation{
				Output:     createVirtual(ctx, "first").Key(),
				Expression: "return base * 2",
				Operations: maxResetOn(second.Key()),
			}
			Expect(create(ctx, &first)).To(Succeed())
			c := calculation.Calculation{
				Output:     second.Key(),
				Expression: "return first > 0",
			}
			Expect(create(ctx, &c)).To(MatchError(validate.ErrValidation))
		},
	)

	It("Should fail a save whose operation resets on its own output", func(
		ctx SpecContext,
	) {
		createVirtual(ctx, "base")
		out := createVirtual(ctx, "positive")
		c := calculation.Calculation{
			Output:     out.Key(),
			Expression: "return base > 0",
			Operations: maxResetOn(out.Key()),
		}
		Expect(create(ctx, &c)).To(MatchError(validate.ErrValidation))
	})

	It(
		"Should fail a save that closes a cycle through a calculation whose "+
			"expression no longer compiles",
		func(ctx SpecContext) {
			createVirtual(ctx, "dep")
			first := createVirtual(ctx, "first")
			second := calculation.Calculation{
				Output:     createVirtual(ctx, "second").Key(),
				Expression: "return first + dep",
			}
			Expect(create(ctx, &second)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).DeleteManyByNames(
					ctx,
					[]string{"dep"},
					false,
				)
			})).To(Succeed())
			c := calculation.Calculation{
				Output:     first.Key(),
				Expression: "return second * 2",
			}
			Expect(create(ctx, &c)).To(MatchError(validate.ErrValidation))
		},
	)

	It("Should fail a save whose operation resets on a channel that is not "+
		"a boolean", func(ctx SpecContext) {
		createVirtual(ctx, "base")
		flag := createVirtual(ctx, "flag")
		c := calculation.Calculation{
			Output:     createVirtual(ctx, "doubled").Key(),
			Expression: "return base * 2",
			Operations: maxResetOn(flag.Key()),
		}
		Expect(create(ctx, &c)).To(MatchError(ContainSubstring("reset")))
	})
})
