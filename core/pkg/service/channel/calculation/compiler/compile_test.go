// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	dist       mock.Node
	channelSvc *channel.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.MustOpenNode(ctx)
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		Search:       dist.Search,
	}))
})

var _ = Describe("Compile", func() {
	It("Should compile simple expression", func(ctx SpecContext) {
		base := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(channelSvc.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base * 2",
		}
		Expect(channelSvc.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with operations", func(ctx SpecContext) {
		base := channel.Channel{Name: "base2", DataType: telem.Int64T, Virtual: true}
		Expect(channelSvc.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc2",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base2 + 1",
			Operations: []channel.Operation{{Type: "avg", Duration: 5 * telem.Second}},
		}
		Expect(channelSvc.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
	})

	It("Should compile with multiple dependencies", func(ctx SpecContext) {
		channels := []channel.Channel{
			{Name: "base3", DataType: telem.Int64T, Virtual: true},
			{Name: "base4", DataType: telem.Int64T, Virtual: true},
		}
		Expect(channelSvc.CreateMany(ctx, &channels)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc3",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base3 + base4",
		}
		Expect(channelSvc.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		}))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElements(channel.KeysFromChannels(channels)))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with derivative operation", func(ctx SpecContext) {
		base := channel.Channel{Name: channel.NewRandomName(), DataType: telem.Float64T, Virtual: true}
		Expect(channelSvc.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       channel.NewRandomName(),
			DataType:   telem.Float64T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s", base.Name),
			Operations: []channel.Operation{{Type: "derivative"}},
		}
		Expect(channelSvc.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.Dependencies.Reads.Slice()).To(ContainElement(base.Key()))
	})

	It("Should fail with invalid expression", func(ctx SpecContext) {
		calc := channel.Channel{
			Name:       "calc4",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return invalid_syntax {{",
		}
		// Bypass analysis on create so the invalid expression reaches the compiler,
		// which is the component under test here.
		Expect(channelSvc.NewWriterNoAnalysis(nil).Create(ctx, &calc)).To(Succeed())
		Expect(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		})).Error().To(ContainSubstring("extraneous input '{'"))
	})
})
