// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx    context.Context
	arcSvc *arc.Service
	dist   mock.Node
)

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
	labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Framer:   dist.Framer,
		Status:   statusSvc,
		Signals:  dist.Signals,
	}))
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})

var _ = Describe("Compile", func() {
	It("Should compile simple expression", func() {
		base := channel.Channel{Name: "base", DataType: telem.Int64T, Virtual: true}
		Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base * 2",
		}
		Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			Channels:       dist.Channel,
			Channel:        calc,
			SymbolResolver: arcSvc.SymbolResolver(),
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.StateConfig.Reads.Keys()).To(ContainElement(base.Key()))
		Expect(mod.StateConfig.Writes.Keys()).To(ContainElement(calc.Key()))
	})

	It("Should compile expression with operations", func() {
		base := channel.Channel{Name: "base2", DataType: telem.Int64T, Virtual: true}
		Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc2",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base2 + 1",
			Operations: []channel.Operation{{Type: "avg", Duration: 5 * telem.Second}},
		}
		Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			Channels:       dist.Channel,
			Channel:        calc,
			SymbolResolver: arcSvc.SymbolResolver(),
		}))
		Expect(mod.Channel.Key()).To(Equal(calc.Key()))
		Expect(mod.StateConfig.Reads.Keys()).To(ContainElement(base.Key()))
	})

	It("Should compile with multiple dependencies", func() {
		channels := []channel.Channel{
			{Name: "base3", DataType: telem.Int64T, Virtual: true},
			{Name: "base4", DataType: telem.Int64T, Virtual: true},
		}
		Expect(dist.Channel.CreateMany(ctx, &channels)).To(Succeed())
		calc := channel.Channel{
			Name:       "calc3",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base3 + base4",
		}
		Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			Channels:       dist.Channel,
			Channel:        calc,
			SymbolResolver: arcSvc.SymbolResolver(),
		}))
		Expect(mod.StateConfig.Reads.Keys()).To(ContainElements(channel.KeysFromChannels(channels)))
		Expect(mod.StateConfig.Writes.Keys()).To(ContainElement(calc.Key()))
	})

	It("Should fail with invalid expression", func() {
		calc := channel.Channel{
			Name:       "calc4",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return invalid_syntax {{",
		}
		Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
		_, err := compiler.Compile(ctx, compiler.Config{
			Channels:       dist.Channel,
			Channel:        calc,
			SymbolResolver: arcSvc.SymbolResolver(),
		})
		Expect(err).To(HaveOccurred())
	})
})
