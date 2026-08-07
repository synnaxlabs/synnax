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

// allSymbols returns every registered stdlib symbol, recursing into module members.
func allSymbols() []*symbol.Symbol {
	var out []*symbol.Symbol
	var walk func([]*symbol.Symbol)
	walk = func(syms []*symbol.Symbol) {
		for _, s := range syms {
			out = append(out, s)
			walk(s.Children())
		}
	}
	walk(stl.NewSymbols())
	return out
}

// findSym resolves an stl symbol by qualified ("module.member") or bare
// ("select") name, returning nil when absent.
func findSym(qualified string) *symbol.Symbol {
	name, member, hasMember := strings.Cut(qualified, ".")
	if !hasMember {
		for _, s := range stl.NewSymbols() {
			if s.Name == name && s.Kind == symbol.KindFunction {
				return s
			}
		}
		return nil
	}
	for _, mod := range allModules() {
		if mod.Name != name {
			continue
		}
		for _, m := range mod.Children() {
			if m.Name == member {
				return m
			}
		}
	}
	return nil
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
						"%s.%s (Kind is %s, expected KindFunction)",
						mod.Name,
						sym.Name,
						sym.Kind,
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

	It("Should attach an AnalyzeArguments hook only to KindFunction symbols", func() {
		var violations []string
		for _, sym := range allSymbols() {
			if sym.AnalyzeArguments != nil && sym.Kind != symbol.KindFunction {
				violations = append(violations, fmt.Sprintf(
					"%s (Kind is %s, but has an AnalyzeArguments hook)",
					sym.Name,
					sym.Kind,
				))
			}
		}
		Expect(violations).To(BeEmpty(),
			"Non-function symbols with an AnalyzeArguments hook:\n  "+
				strings.Join(violations, "\n  "))
	})

	It("Should bind every non-empty Trigger.Target to an existing input param", func() {
		var violations []string
		for _, sym := range allSymbols() {
			target := sym.Trigger.Target
			if target == "" {
				continue
			}
			if !sym.Type.Inputs.Has(target) {
				violations = append(violations, fmt.Sprintf(
					"%s (Trigger.Target %q names no param in Inputs)", sym.Name, target,
				))
			}
		}
		Expect(violations).To(BeEmpty(),
			"Symbols whose Trigger.Target names no existing input param:\n  "+
				strings.Join(violations, "\n  "))
	})

	It(
		"Should use DefaultOutputParam on user-callable single-output functions",
		func() {
			var violations []string
			for _, mod := range allModules() {
				for _, sym := range mod.Children() {
					if sym.Internal || sym.Type.Kind != types.KindFunction ||
						len(sym.Type.Outputs) != 1 {
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
					strings.Join(violations, "\n  "),
			)
		},
	)

	DescribeTable(
		"Should declare the unified Inputs and Trigger",
		func(qualified string, wantInputNames []string, wantTrigger symbol.TriggerBinding) {
			sym := findSym(qualified)
			Expect(sym).ToNot(BeNil(), "symbol %q not registered", qualified)
			gotNames := make([]string, len(sym.Type.Inputs))
			for i, p := range sym.Type.Inputs {
				gotNames[i] = p.Name
			}
			Expect(gotNames).To(Equal(wantInputNames))
			Expect(sym.Trigger).To(Equal(wantTrigger))
		},
		Entry(
			"math.avg",
			"math.avg",
			[]string{
				ir.DefaultInputParam,
				"duration",
				"count",
				"reset",
			},
			symbol.TriggerInput(ir.DefaultInputParam),
		),
		Entry(
			"math.min",
			"math.min",
			[]string{
				ir.DefaultInputParam,
				"duration",
				"count",
				"reset",
			},
			symbol.TriggerInput(ir.DefaultInputParam),
		),
		Entry(
			"math.max",
			"math.max",
			[]string{
				ir.DefaultInputParam,
				"duration",
				"count",
				"reset",
			},
			symbol.TriggerInput(ir.DefaultInputParam),
		),
		Entry("math.derivative", "math.derivative",
			[]string{ir.DefaultInputParam}, symbol.TriggerInput(ir.DefaultInputParam)),
		Entry("math.pow", "math.pow",
			[]string{"base", "exp"}, symbol.TriggerOnly),
		Entry(
			"op.ge",
			"ge",
			[]string{
				ir.LHSInputParam,
				ir.RHSInputParam,
			},
			symbol.TriggerInput(ir.LHSInputParam),
		),
		Entry(
			"op.and",
			"and",
			[]string{
				ir.LHSInputParam,
				ir.RHSInputParam,
			},
			symbol.TriggerInput(ir.LHSInputParam),
		),
		Entry("op.not", "not",
			[]string{ir.DefaultInputParam}, symbol.TriggerInput(ir.DefaultInputParam)),
		Entry("on", "on",
			[]string{"channel"}, symbol.TriggerOnly),
		Entry(
			"write",
			"write",
			[]string{
				ir.DefaultInputParam,
				"channel",
			},
			symbol.TriggerInput(ir.DefaultInputParam),
		),
		Entry(
			"stable.for",
			"stable.for",
			[]string{
				ir.DefaultInputParam,
				"duration",
			},
			symbol.TriggerInput(ir.DefaultInputParam),
		),
		Entry(
			"select",
			"select",
			[]string{
				ir.DefaultOutputParam,
			},
			symbol.TriggerInput(ir.DefaultOutputParam),
		),
		Entry("time.interval", "time.interval",
			[]string{"period"}, symbol.TriggerOnly),
		Entry("time.wait", "time.wait",
			[]string{"duration"}, symbol.TriggerOnly),
		Entry("time.now", "time.now",
			[]string{}, symbol.TriggerOnly),
		Entry("constant", "constant",
			[]string{"value"}, symbol.TriggerOnly),
	)
})
