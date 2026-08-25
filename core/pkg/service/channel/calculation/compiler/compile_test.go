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
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	channelSvc    *channel.Service
	channelWriter channel.Writer
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	channelWriter = channelSvc.NewWriter(nil)
})

var _ = Describe("Compile", func() {
	It("Should compile simple expression", func(ctx SpecContext) {
		base := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(channelWriter.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base * 2",
		}
		Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
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
		Expect(channelWriter.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc2",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base2 + 1",
			Operations: []channel.Operation{{Type: "avg", Duration: 5 * telem.Second}},
		}
		Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
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
		Expect(channelWriter.CreateMany(ctx, &channels)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc3",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base3 + base4",
		}
		Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		}))
		Expect(
			mod.Dependencies.Reads.Slice(),
		).To(ContainElements(channel.KeysFromChannels(channels)))
		Expect(mod.Dependencies.Writes.Slice()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with derivative operation", func(ctx SpecContext) {
		base := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float64T,
			Virtual:  true,
		}
		Expect(channelWriter.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       UniqueChannelName(),
			DataType:   telem.Float64T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s", base.Name),
			Operations: []channel.Operation{{Type: "derivative"}},
		}
		Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
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
			Expression: "return 1",
		}
		Expect(channelWriter.Create(ctx, &calc)).To(Succeed())
		calc.Expression = "return invalid_syntax {{"
		Expect(compiler.Compile(ctx, compiler.Config{
			ChannelService: channelSvc,
			Channel:        calc,
		})).Error().To(ContainSubstring("extraneous input '{'"))
	})
})
