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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
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
	retrieve := func(ctx SpecContext, output channel.Key) []calculation.Calculation {
		GinkgoHelper()
		var out []calculation.Calculation
		Expect(svc.NewRetrieve().
			Where(calculation.MatchOutputs(output)).
			Entries(&out).
			Exec(ctx, node.DB)).To(Succeed())
		return out
	}

	Describe("Create", func() {
		It("Should create a calculation on a virtual channel", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})

		It("Should fail on an output that stores its samples", func(ctx SpecContext) {
			index := createChannel(ctx, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			stored := createChannel(ctx, channel.Channel{
				Name:       "pressure",
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			})
			c := calculation.Calculation{Output: stored.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"channel pressure stores its samples, so no calculation writes it",
				)),
			))
		})

		It("Should fail on an output that another calculation writes", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			first := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &first)).To(Succeed())
			second := calculation.Calculation{Output: out.Key(), Expression: "return 3"}
			Expect(create(ctx, &second)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("another record already writes channel doubled"),
				),
			))
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(first))
		})

		It("Should fail on an output that does not exist", func(ctx SpecContext) {
			c := calculation.Calculation{Output: 1 << 20, Expression: "return 2"}
			Expect(create(ctx, &c)).To(MatchError(query.ErrNotFound))
		})

		It("Should fail on an empty expression", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key()}
			Expect(create(ctx, &c)).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, out.Key())).To(BeEmpty())
		})
	})

	Describe("Apply", func() {
		It("Should set the expression of a calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return 4",
					},
				))
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())[0].Expression).To(Equal("return 4"))
		})

		It("Should keep a calculation whose new expression is empty", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{Path: []string{"expression"}, Value: ""},
				))
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})
	})

	Describe("Delete", func() {
		It("Should keep the output of a deleted calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(BeEmpty())
			Expect(channels.NewRetrieve().
				Where(channel.MatchKeys(out.Key())).
				Exists(ctx, node.DB)).To(BeTrue())
		})

		It("Should delete the calculation of a deleted output", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Delete(ctx, out.Key(), false)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(calculation.MatchKeys(c.Key)).
				Exists(ctx, node.DB)).To(BeFalse())
		})
	})
})
