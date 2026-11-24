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

var _ = Describe("CreateOutput breaker", func() {
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
	createInput := func(ctx SpecContext, name string, dt telem.DataType) {
		GinkgoHelper()
		ch := channel.Channel{Name: name, DataType: dt, Virtual: true}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
	}
	exists := func(ctx SpecContext, filter channel.Filter) bool {
		GinkgoHelper()
		return MustSucceed(channels.NewRetrieve().Where(filter).Exists(ctx, node.DB))
	}
	createOutput := func(ctx SpecContext, name string) []channel.Channel {
		GinkgoHelper()
		var created []channel.Channel
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			c := calculation.Calculation{Expression: "return wide * 2"}
			created, err = svc.NewWriter(tx).CreateOutput(ctx, &c, name, nil)
			return err
		})).To(Succeed())
		return created
	}

	It("Should delete the free index of an output when the output is deleted", func(
		ctx SpecContext,
	) {
		createInput(ctx, "wide", telem.Int64T)
		created := createOutput(ctx, "doubled")
		Expect(created).To(HaveLen(2))
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Delete(ctx, created[0].Key(), false)
		})).To(Succeed())
		Expect(exists(ctx, channel.MatchKeys(created[1].Key()))).To(BeFalse())
	})

	It("Should make no output when the record of the output fails to save", func(
		ctx SpecContext,
	) {
		createInput(ctx, "wide", telem.Int64T)
		createInput(ctx, "level", telem.Float32T)
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			c := calculation.Calculation{
				Expression: "return wide * 2",
				Operations: []calculation.Operation{{
					Variant: calculation.MaxOperation{Window: calculation.Window{
						Reset: new(channelKey(ctx, channels, node, "level")),
					}},
				}},
			}
			_, err := svc.NewWriter(tx).CreateOutput(ctx, &c, "doubled", nil)
			return err
		})).To(MatchError(ContainSubstring("reset")))
		Expect(exists(ctx, channel.MatchNames("doubled", "doubled_time"))).
			To(BeFalse())
	})

	It("Should refuse an output name that another channel holds", func(
		ctx SpecContext,
	) {
		createInput(ctx, "wide", telem.Int64T)
		createInput(ctx, "taken", telem.Int64T)
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			c := calculation.Calculation{Expression: "return wide * 2"}
			_, err := svc.NewWriter(tx).CreateOutput(ctx, &c, "taken", nil)
			return err
		})).To(MatchError(validate.ErrValidation))
		Expect(exists(ctx, channel.MatchNames("taken_time"))).To(BeFalse())
	})
})

func channelKey(
	ctx SpecContext,
	channels *channel.Service,
	node disttestutil.Node,
	name string,
) channel.Key {
	GinkgoHelper()
	var ch channel.Channel
	Expect(channels.NewRetrieve().
		Where(channel.MatchNames(name)).
		Entry(&ch).
		Exec(ctx, node.DB)).To(Succeed())
	return ch.Key()
}
