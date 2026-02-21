// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("SymbolResolver", func() {
	It("Should resolve set_status by name", func() {
		sym, ok := arcstatus.SymbolResolver["set_status"]
		Expect(ok).To(BeTrue())
		Expect(sym.Name).To(Equal("set_status"))
		Expect(sym.Kind).To(Equal(symbol.KindFunction))
	})

	It("Should have the correct type signature", func() {
		sym := arcstatus.SymbolResolver["set_status"]
		Expect(sym.Type.Kind).To(Equal(types.KindFunction))
		Expect(sym.Type.Config).To(HaveLen(4))
		Expect(sym.Type.Config[0].Name).To(Equal("status_key"))
		Expect(sym.Type.Config[1].Name).To(Equal("variant"))
		Expect(sym.Type.Config[2].Name).To(Equal("message"))
		Expect(sym.Type.Config[3].Name).To(Equal("name"))
		Expect(sym.Type.Inputs).To(HaveLen(1))
		Expect(sym.Type.Inputs[0].Name).To(Equal(ir.DefaultOutputParam))
		Expect(sym.Type.Inputs[0].Type).To(Equal(types.U8()))
	})

	It("Should not resolve an unknown name", func() {
		_, ok := arcstatus.SymbolResolver["unknown"]
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("Module", func() {
	var mod *arcstatus.Module

	BeforeEach(func() {
		mod = arcstatus.NewModule(statSvc)
	})

	Describe("Resolve", func() {
		It("Should resolve set_status", func() {
			sym, err := mod.Resolve(ctx, "set_status")
			Expect(err).ToNot(HaveOccurred())
			Expect(sym.Name).To(Equal("set_status"))
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
		})

		It("Should return error for unknown symbol", func() {
			_, err := mod.Resolve(ctx, "nonexistent")
			Expect(err).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("Create", func() {
		It("Should return not found for wrong node type", func() {
			cfg := node.Config{
				Node: ir.Node{Type: "wrong_type"},
			}
			_, err := mod.Create(ctx, cfg)
			Expect(err).To(MatchError(query.ErrNotFound))
		})

		It("Should create a node with valid config", func() {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: "test_alarm"},
						{Name: "variant", Value: "success"},
						{Name: "message", Value: "All systems nominal"},
					},
				},
			}
			n, err := mod.Create(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(n).ToNot(BeNil())
		})
	})

	Describe("BindTo", func() {
		It("Should return nil", func() {
			Expect(mod.BindTo(nil)).To(Succeed())
		})
	})
})
