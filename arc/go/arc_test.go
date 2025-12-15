// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	compile := func(code string, resolver arc.SymbolResolver) arc.Module {
		t := arc.Text{Raw: code}
		Expect(t.Raw).ToNot(BeEmpty())
		return MustSucceed(arc.CompileText(ctx, t, arc.WithResolver(resolver)))
	}
	It("Should compile a basic calculated channel", func() {
		mod := compile(
			`func calc(val f32) f32 {
    			return val * 2
			}

			ox_pt_1 -> calc{} -> ox_pt_doubled
			`,
			symbol.MapResolver{
				"ox_pt_1": arc.Symbol{
					Name: "ox_pt_1",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   1,
				},
				"ox_pt_doubled": arc.Symbol{
					Name: "ox_pt_doubled",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   2,
				},
			})

		Expect(mod.Nodes).To(HaveLen(3))
		Expect(mod.Nodes[0].Type).To(Equal("on"))
		Expect(mod.Nodes[0].Channels.Read.Contains(uint32(1))).To(BeTrue())
		Expect(mod.Nodes[0].Outputs).To(HaveLen(1))
		Expect(mod.Nodes[1].Type).To(Equal("calc"))
		Expect(mod.Nodes[2].Type).To(Equal("write"))
		Expect(mod.Nodes[2].Channels.Write.Contains(uint32(2))).To(BeTrue())
		Expect(mod.Nodes[2].Inputs).To(HaveLen(1))
		Expect(mod.Edges).To(HaveLen(2))
	})

	It("Should compile a one-stage sequence", func() {
		mod := compile(
			`sequence seg {
				stage init {
					1 -> output
				}
			}`,
			symbol.MapResolver{
				"output": arc.Symbol{
					Name: "output",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   1,
				},
			})
		fmt.Println(mod)
		Expect(mod.Nodes).To(HaveLen(3))
	})
})
