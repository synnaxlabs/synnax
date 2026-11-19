// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	. "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Creator", Ordered, func() {
	var (
		creator Creator
		input   string
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		dist := disttestutil.NewNode(ctx)
		statusSvc := MustOpen(
			status.OpenService(ctx, status.ServiceConfig{DB: dist.DB}),
		)
		svc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: dist.Channel,
			TS:      dist.Storage.TS,
			Framer:  dist.Framer,
			DB:      dist.DB,
			HostKey: dist.Identity.HostKey,
			Status:  statusSvc,
		}))
		creator = Creator{DB: dist.DB, Channel: svc}
		keys := MustSucceed(channeltestutil.CreateVirtual(ctx, dist.DB, svc, 1))
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(keys...)).
			Entry(&ch).
			Exec(ctx, dist.DB)).To(Succeed())
		input = ch.Name
	})

	Describe("Create", func() {
		It(
			"Should create a calculation and return its output channel",
			func(ctx SpecContext) {
				name := channeltestutil.UniqueChannelName()
				out := MustSucceed(creator.Create(ctx, Calculation{
					Name:       name,
					Expression: "return " + input + " * 2",
				}))
				Expect(out.Name).To(Equal(name))
				Expect(out.Expression).To(Equal("return " + input + " * 2"))
				Expect(out.Key()).ToNot(BeZero())
				Expect(out.Index()).ToNot(BeZero())
				Expect(out.Virtual).To(BeTrue())
				Expect(out.Leaseholder).To(Equal(node.KeyFree))
				var stored channel.Channel
				Expect(creator.Channel.NewRetrieve().
					Where(channel.MatchKeys(out.Key())).
					Entry(&stored).
					Exec(ctx, creator.DB)).To(Succeed())
				Expect(stored.Expression).To(Equal(out.Expression))
			},
		)
		It(
			"Should replace the data type with the type of the expression",
			func(ctx SpecContext) {
				out := MustSucceed(creator.Create(ctx, Calculation{
					Name:       channeltestutil.UniqueChannelName(),
					Expression: "return " + input + " * 2",
					DataType:   telem.Uint8T,
				}))
				Expect(out.DataType).To(Equal(telem.Float64T))
			},
		)
		It(
			"Should return the error of a calculation that does not compile",
			func(ctx SpecContext) {
				Expect(creator.Create(ctx, Calculation{
					Name:       channeltestutil.UniqueChannelName(),
					Expression: "return no_such_channel",
				})).Error().To(MatchError(ContainSubstring("no_such_channel")))
			},
		)
	})

	Describe("CreateMany", func() {
		It(
			"Should return the output channels in the order of the calculations",
			func(ctx SpecContext) {
				names := []string{
					channeltestutil.UniqueChannelName(),
					channeltestutil.UniqueChannelName(),
				}
				outs := MustSucceed(creator.CreateMany(ctx, []Calculation{
					{Name: names[0], Expression: "return " + input + " + 1"},
					{Name: names[1], Expression: "return " + input + " + 2"},
				}))
				Expect(channel.Names(outs)).To(Equal(names))
				Expect(outs).To(HaveEach(HaveField("Virtual", BeTrue())))
			},
		)
		It("Should create nothing for no calculations", func(ctx SpecContext) {
			Expect(creator.CreateMany(ctx, nil)).To(BeEmpty())
		})
	})

	Describe("Indexes", func() {
		It(
			"Should return the index channel of each output in order",
			func(ctx SpecContext) {
				outs := MustSucceed(creator.CreateMany(ctx, []Calculation{
					{
						Name:       channeltestutil.UniqueChannelName(),
						Expression: "return " + input,
					},
					{
						Name:       channeltestutil.UniqueChannelName(),
						Expression: "return " + input,
					},
				}))
				indexes := MustSucceed(creator.Indexes(ctx, outs))
				Expect(channel.KeysFromChannels(indexes)).
					To(Equal(channel.Keys{outs[0].Index(), outs[1].Index()}))
				Expect(indexes).To(HaveEach(HaveField("IsIndex", BeTrue())))
				Expect(channel.Names(indexes)).
					To(Equal([]string{outs[0].Name + "_time", outs[1].Name + "_time"}))
			},
		)
		It("Should return no channels for no outputs", func(ctx SpecContext) {
			Expect(creator.Indexes(ctx, nil)).To(BeEmpty())
		})
	})
})
