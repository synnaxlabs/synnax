// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package stl_test

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

func allModules() []*symbol.Symbol {
	var mods []*symbol.Symbol
	for _, s := range stl.NewSymbols() {
		if s.Kind == symbol.KindModule {
			mods = append(mods, s)
		}
	}
	return mods
}

var _ = Describe("STL Symbols", func() {
	It("Should set KindFunction on every module member with a function type", func() {
		var violations []string
		for _, mod := range allModules() {
			for _, sym := range mod.Children() {
				if sym.Type.Kind != types.KindFunction {
					continue
				}
				if sym.Kind != symbol.KindFunction {
					violations = append(violations, fmt.Sprintf(
						"%s.%s (Kind is %s, expected KindFunction)", mod.Name, sym.Name, sym.Kind,
					))
				}
			}
		}
		Expect(violations).To(BeEmpty(),
			"Module members with function types missing Kind: symbol.KindFunction:\n  "+
				strings.Join(violations, "\n  "))
	})

	It("Should set ExecContext on every KindFunction symbol", func() {
		var violations []string
		for _, mod := range allModules() {
			for _, sym := range mod.Children() {
				if sym.Kind != symbol.KindFunction {
					continue
				}
				if sym.Exec == 0 {
					violations = append(violations, fmt.Sprintf(
						"%s.%s (Exec is 0, must be ExecWASM, ExecFlow, or ExecBoth)",
						mod.Name, sym.Name,
					))
				}
			}
		}
		Expect(violations).To(BeEmpty(),
			"KindFunction symbols with unset ExecContext:\n  "+
				strings.Join(violations, "\n  "))
	})

	It("Should obey the ExecBoth structural contract on every ExecBoth symbol", func() {
		var violations []string
		for _, mod := range allModules() {
			for _, sym := range mod.Children() {
				if sym.Kind != symbol.KindFunction || sym.Exec != symbol.ExecBoth {
					continue
				}
				inputs := sym.Type.Inputs
				config := sym.Type.Config
				if len(inputs) != len(config) {
					violations = append(violations, fmt.Sprintf(
						"%s.%s (Inputs has %d params, Config has %d; ExecBoth requires "+
							"one-to-one mirroring)",
						mod.Name, sym.Name, len(inputs), len(config),
					))
					continue
				}
				for i := range inputs {
					if inputs[i].Name != config[i].Name || !types.Equal(inputs[i].Type, config[i].Type) {
						violations = append(violations, fmt.Sprintf(
							"%s.%s (Inputs[%d]={%s,%s} does not match Config[%d]={%s,%s})",
							mod.Name, sym.Name,
							i, inputs[i].Name, inputs[i].Type,
							i, config[i].Name, config[i].Type,
						))
					}
				}
			}
		}
		Expect(violations).To(BeEmpty(),
			"ExecBoth symbols violating the dual-shape contract:\n  "+
				strings.Join(violations, "\n  "))
	})

	It("Should use DefaultOutputParam on user-callable single-output functions", func() {
		var violations []string
		for _, mod := range allModules() {
			for _, sym := range mod.Children() {
				if sym.Internal || sym.Type.Kind != types.KindFunction || len(sym.Type.Outputs) != 1 {
					continue
				}
				out := sym.Type.Outputs[0]
				if out.Name != ir.DefaultOutputParam {
					violations = append(violations, fmt.Sprintf(
						"%s.%s output is named %q, expected %q",
						mod.Name, sym.Name, out.Name, ir.DefaultOutputParam,
					))
				}
			}
		}
		Expect(violations).To(BeEmpty(),
			"User-callable single-output functions with non-default output name (will be rejected as non-callable):\n  "+
				strings.Join(violations, "\n  "))
	})
})
