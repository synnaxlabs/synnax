// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"encoding/binary"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/ir"
	programpb "github.com/synnaxlabs/arc/program/pb"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Arc", func() {
	compile := func(ctx SpecContext, code string, resolver arc.SymbolResolver) arc.Program {
		t := arc.Text{Raw: code}
		Expect(t.Raw).ToNot(BeEmpty())
		return MustSucceed(arc.CompileText(ctx, t, arc.WithResolver(resolver)))
	}

	findNodeByType := func(nodes ir.Nodes, nodeType string) ir.Node {
		for _, n := range nodes {
			if n.Type == nodeType {
				return n
			}
		}
		Fail("node with type " + nodeType + " not found")
		return ir.Node{}
	}

	// findTopLevelScope returns the top-level Scope member whose key matches.
	// Fails the spec if no such member exists.
	findTopLevelScope := func(prog arc.Program, key string) ir.Scope {
		for _, stratum := range prog.Root.Strata {
			for _, m := range stratum {
				if m.Scope != nil && m.Scope.Key == key {
					return *m.Scope
				}
			}
		}
		Fail("expected top-level scope '" + key + "'")
		return ir.Scope{}
	}

	// findMember returns the member with the given key in a scope's
	// Steps (sequential) or its Strata (parallel).
	findMember := func(scope ir.Scope, key string) ir.Member {
		for _, m := range scope.Steps {
			if m.Key() == key {
				return m
			}
		}
		for _, stratum := range scope.Strata {
			for _, m := range stratum {
				if m.Key() == key {
					return m
				}
			}
		}
		Fail("expected member '" + key + "' in scope '" + scope.Key + "'")
		return ir.Member{}
	}

	// scopeNodeRefs collects every leaf-node key reachable within a scope
	// (across all strata and direct steps).
	scopeNodeRefs := func(scope ir.Scope) []string {
		var keys []string
		var walk func(s ir.Scope)
		walk = func(s ir.Scope) {
			for _, stratum := range s.Strata {
				for _, m := range stratum {
					if m.NodeKey != nil {
						keys = append(keys, *m.NodeKey)
					} else if m.Scope != nil {
						walk(*m.Scope)
					}
				}
			}
			for _, m := range s.Steps {
				if m.NodeKey != nil {
					keys = append(keys, *m.NodeKey)
				} else if m.Scope != nil {
					walk(*m.Scope)
				}
			}
		}
		walk(scope)
		return keys
	}

	// nextMember returns the member following key in a sequential scope's
	// Steps slice, or ok=false if key is the last or not found.
	nextMember := func(scope ir.Scope, key string) (ir.Member, bool) {
		for i, m := range scope.Steps {
			if m.Key() == key {
				if i+1 < len(scope.Steps) {
					return scope.Steps[i+1], true
				}
				return ir.Member{}, false
			}
		}
		return ir.Member{}, false
	}

	// isStageMember reports whether the member wraps a parallel scope
	// (mirrors the old Step.IsStage).
	isStageMember := func(m ir.Member) bool {
		return m.Scope != nil && m.Scope.Mode == ir.ScopeModeParallel
	}

	// isFlowMember reports whether the member wraps a parallel scope that
	// came from an anonymous flow step. Under the current analyzer both
	// stage and flow steps lower to PARALLEL+GATED scopes; flow steps are
	// identified by a synthesized "step_N" key.
	isFlowMember := func(m ir.Member) bool {
		return m.Scope != nil &&
			m.Scope.Mode == ir.ScopeModeParallel &&
			strings.HasPrefix(m.Key(), "step_")
	}

	_ = nextMember
	_ = isStageMember
	_ = isFlowMember
	_ = scopeNodeRefs
	_ = findMember
	_ = findTopLevelScope

	It("Should compile a basic calculated channel", func(ctx SpecContext) {
		mod := compile(ctx,
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

		Expect(mod.Functions).To(HaveLen(1))
		calcFunc := MustBeOk(mod.Functions.Find("calc"))
		Expect(calcFunc.Key).To(Equal("calc"))
		Expect(calcFunc.Inputs).To(HaveLen(1))
		valParam := MustBeOk(calcFunc.Inputs.Get("val"))
		Expect(valParam.Type.Kind).To(Equal(types.KindF32))
		Expect(calcFunc.Outputs).To(HaveLen(1))
		outputParam := MustBeOk(calcFunc.Outputs.Get("output"))
		Expect(outputParam.Type.Kind).To(Equal(types.KindF32))

		Expect(mod.Nodes).To(HaveLen(3))

		onNode := findNodeByType(mod.Nodes, "on")
		Expect(onNode.Channels.Read).To(HaveKey(uint32(1)))
		Expect(onNode.Outputs).To(HaveLen(1))
		Expect(onNode.Outputs.Has("output")).To(BeTrue())

		calcNode := findNodeByType(mod.Nodes, "calc")
		Expect(calcNode.Inputs).To(HaveLen(1))
		Expect(calcNode.Inputs.Has("val")).To(BeTrue())
		Expect(calcNode.Outputs).To(HaveLen(1))
		Expect(calcNode.Outputs.Has("output")).To(BeTrue())

		writeNode := findNodeByType(mod.Nodes, "write")
		Expect(writeNode.Channels.Write).To(HaveKey(uint32(2)))
		Expect(writeNode.Inputs).To(HaveLen(1))

		Expect(mod.Edges).To(HaveLen(2))

		edge1 := MustBeOk(mod.Edges.FindByTarget(ir.Handle{Node: calcNode.Key, Param: "val"}))
		Expect(edge1.Source.Node).To(Equal(onNode.Key))
		Expect(edge1.Source.Param).To(Equal("output"))
		Expect(edge1.Kind).To(Equal(ir.EdgeKindContinuous))

		edge2 := MustBeOk(mod.Edges.FindBySource(ir.Handle{Node: calcNode.Key, Param: "output"}))
		Expect(edge2.Target.Node).To(Equal(writeNode.Key))
		Expect(edge2.Kind).To(Equal(ir.EdgeKindContinuous))

		// Root strata reflect the linear dependency chain: on → calc → write.
		Expect(mod.Root.Strata).To(HaveLen(3))
		stratumKeys := func(stratum ir.Members) []string {
			keys := make([]string, 0, len(stratum))
			for _, m := range stratum {
				keys = append(keys, m.Key())
			}
			return keys
		}
		Expect(stratumKeys(mod.Root.Strata[0])).To(ContainElement(onNode.Key))
		Expect(stratumKeys(mod.Root.Strata[1])).To(ContainElement(calcNode.Key))
		Expect(stratumKeys(mod.Root.Strata[2])).To(ContainElement(writeNode.Key))
	})

	It("Should compile a one-stage sequence", func(ctx SpecContext) {
		mod := compile(ctx,
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

		seg := findTopLevelScope(mod, "seg")
		Expect(seg.Mode).To(Equal(ir.ScopeModeSequential))
		Expect(seg.Steps).To(HaveLen(1))
		Expect(seg.Steps[0].Key()).To(Equal("init"))

		init := findMember(seg, "init")
		Expect(init.Scope).ToNot(BeNil())
		Expect(scopeNodeRefs(*init.Scope)).To(HaveLen(2))

		// Only constant + write nodes exist; no synthesized stage_entry
		// under the new IR.
		Expect(mod.Nodes).To(HaveLen(2))
		constNode := findNodeByType(mod.Nodes, "constant")
		Expect(constNode.Config).To(HaveLen(1))

		writeNode := findNodeByType(mod.Nodes, "write")
		Expect(writeNode.Channels.Write).To(HaveKey(uint32(1)))

		Expect(mod.Edges).To(HaveLen(1))
		edge := MustBeOk(mod.Edges.FindByTarget(ir.Handle{Node: writeNode.Key, Param: "input"}))
		Expect(edge.Source.Node).To(Equal(constNode.Key))
		Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))

		Expect(init.Scope.Strata).To(HaveLen(2))
		stratumKeys := func(stratum ir.Members) []string {
			out := make([]string, 0, len(stratum))
			for _, m := range stratum {
				out = append(out, m.Key())
			}
			return out
		}
		Expect(stratumKeys(init.Scope.Strata[0])).To(ContainElement(constNode.Key))
		Expect(stratumKeys(init.Scope.Strata[1])).To(ContainElement(writeNode.Key))
	})

	It("Should compile a three stage sequence", func(ctx SpecContext) {
		mod := compile(ctx, `
start_seq_cmd => main

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt > 50 => next
    }
    stage stop {
        0 -> press_vlv_cmd
    }
}
`, symbol.MapResolver{
			"press_vlv_cmd": arc.Symbol{
				Name: "press_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   1,
			},
			"vent_vlv_cmd": arc.Symbol{
				Name: "vent_vlv_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   4,
			},
			"press_pt": arc.Symbol{
				Name: "press_pt",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   2,
			},
			"start_seq_cmd": arc.Symbol{
				Name: "start_seq_cmd",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
				ID:   3,
			},
		})

		main := findTopLevelScope(mod, "main")
		Expect(main.Steps).To(HaveLen(2))

		press := findMember(main, "press")
		Expect(press.Scope).ToNot(BeNil())
		Expect(scopeNodeRefs(*press.Scope)).ToNot(BeEmpty())

		stop := findMember(main, "stop")
		Expect(stop.Scope).ToNot(BeNil())
		Expect(scopeNodeRefs(*stop.Scope)).ToNot(BeEmpty())

		nextAfterPress, ok := nextMember(main, "press")
		Expect(ok).To(BeTrue())
		Expect(nextAfterPress.Key()).To(Equal("stop"))

		_, ok = nextMember(main, "stop")
		Expect(ok).To(BeFalse())

		// press → stop transition comes from `press_pt > 50 => next`.
		Expect(main.Transitions).ToNot(BeEmpty())

		continuousEdges := mod.Edges.GetByKind(ir.EdgeKindContinuous)
		Expect(continuousEdges).ToNot(BeEmpty())
	})

	It("Should correctly generate strata for a loop", func(ctx SpecContext) {
		mod := compile(ctx, `
		start_seq_cmd => main

		func expr(in f32) u8 {
			return in > 2
		}

		func expr2(in f32) u8 {
			return in < 0.3
		}

		sequence main {
			stage press {
				1 -> press_vlv_cmd,
				press_pt -> expr{} => next
			}
			stage vent {
				1 -> vent_vlv_cmd,
				0 -> press_vlv_cmd,
				press_pt -> expr2{} => press
			}
		}
		`,
			symbol.MapResolver{
				"press_vlv_cmd": arc.Symbol{
					Name: "press_vlv_cmd",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   1,
				},
				"vent_vlv_cmd": arc.Symbol{
					Name: "vent_vlv_cmd",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   4,
				},
				"press_pt": arc.Symbol{
					Name: "press_pt",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   2,
				},
				"start_seq_cmd": arc.Symbol{
					Name: "start_seq_cmd",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   3,
				},
			})

		Expect(mod.Functions).To(HaveLen(2))
		MustBeOk(mod.Functions.Find("expr"))
		MustBeOk(mod.Functions.Find("expr2"))

		main := findTopLevelScope(mod, "main")
		Expect(main.Steps).To(HaveLen(2))

		press := findMember(main, "press")
		Expect(press.Scope).ToNot(BeNil())
		Expect(press.Scope.Strata).ToNot(BeEmpty())

		vent := findMember(main, "vent")
		Expect(vent.Scope).ToNot(BeNil())
		Expect(vent.Scope.Strata).ToNot(BeEmpty())

		nextAfterPress, ok := nextMember(main, "press")
		Expect(ok).To(BeTrue())
		Expect(nextAfterPress.Key()).To(Equal("vent"))

		_, ok = nextMember(main, "vent")
		Expect(ok).To(BeFalse())

		// Two => transitions: press→next and vent→press.
		Expect(main.Transitions).To(HaveLen(2))
	})

	It("Should correctly compile a node with a unit literal", func(ctx SpecContext) {
		mod := compile(ctx, `
			sequence main {
				stage initial {
					wait{duration=5s} => next
				}
				stage end {
				}
			}
		`, time.SymbolResolver)
		// Under the new IR the wait node is the only IR node; stages and
		// entries no longer materialize as nodes.
		Expect(mod.Nodes).To(HaveLen(1))
		Expect(mod.Nodes[0].Type).To(Equal("wait"))
	})

	It("Should generate typed state imports for stateful variables", func(ctx SpecContext) {
		// Regression test: stateful variables must produce typed WASM imports
		// like "state::load_i64", not bare "state::load". This mirrors the
		// exact program used in the C++ NodeTest.StatefulVariablesAreIsolatedBetweenNodeInstances.
		channelResolver := symbol.MapResolver{
			"trigger": arc.Symbol{
				Name: "trigger",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I64()),
				ID:   1,
			},
			"output_a": arc.Symbol{
				Name: "output_a",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I64()),
				ID:   2,
			},
			"output_b": arc.Symbol{
				Name: "output_b",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I64()),
				ID:   3,
			},
		}
		fullResolver := symbol.CompoundResolver{stl.SymbolResolver, channelResolver}

		mod := compile(ctx, `
func counter(trigger i64) i64 {
    count i64 $= 0
    count = count + 1
    return count
}
trigger -> counter{} -> output_a
trigger -> counter{} -> output_b
`, fullResolver)

		Expect(mod.WASM).ToNot(BeEmpty())

		imports := parseWASMImports(mod.WASM)
		Expect(imports).ToNot(BeEmpty())

		for _, imp := range imports {
			if imp.module == "state" {
				// Every state import must have a type suffix (e.g., load_i64, store_i64)
				Expect(imp.name).ToNot(Equal("load"),
					"state::load should be state::load_i64 (missing type suffix)")
				Expect(imp.name).ToNot(Equal("store"),
					"state::store should be state::store_i64 (missing type suffix)")
				Expect(
					strings.HasPrefix(imp.name, "load_") || strings.HasPrefix(imp.name, "store_"),
				).To(BeTrue(), "unexpected state import: %s", imp.name)
			}
		}
	})

	It("Should return a compile error when () is used instead of {} in a flow", func(ctx SpecContext) {
		resolver := symbol.CompoundResolver{
			stl.SymbolResolver,
			symbol.MapResolver{
				"some_ch": arc.Symbol{
					Name: "some_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.I64()),
					ID:   1,
				},
			},
		}
		t := arc.Text{Raw: `
some_ch -> check()

func check() {
    a := 1
}
`}
		_, err := arc.CompileText(ctx, t, arc.WithResolver(resolver))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("functions in flow statements use {} not ()"))
		Expect(err.Error()).To(ContainSubstring("did you mean: check{}?"))
	})

	Describe("Stageless Sequences", func() {
		It("Should compile a stageless sequence with two writes", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	1 -> valve_a
	1 -> valve_b
}
`,
				symbol.MapResolver{
					"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
					"valve_a":   symbol.Symbol{Name: "valve_a", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
					"valve_b":   symbol.Symbol{Name: "valve_b", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 3},
				},
			)

			main := findTopLevelScope(mod, "main")
			Expect(main.Mode).To(Equal(ir.ScopeModeSequential))
			Expect(main.Steps).To(HaveLen(2))
			Expect(isFlowMember(main.Steps[0])).To(BeTrue())
			Expect(isFlowMember(main.Steps[1])).To(BeTrue())
			// Auto-wired transitions connect the flow steps.
			Expect(main.Transitions).ToNot(BeEmpty())
		})

		It("Should compile a stageless sequence with a function node", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	1 -> valve_cmd
	wait{duration=2s}
	0 -> valve_cmd
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"valve_cmd": symbol.Symbol{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
					},
					stl.SymbolResolver,
				},
			)

			main := findTopLevelScope(mod, "main")
			Expect(main.Steps).To(HaveLen(3))
			Expect(isFlowMember(main.Steps[0])).To(BeTrue())
			Expect(isFlowMember(main.Steps[1])).To(BeTrue())
			Expect(isFlowMember(main.Steps[2])).To(BeTrue())
		})

		It("Should compile a mixed stage and flow sequence", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	stage press {
		1 -> press_cmd,
		press_pt > 50 => next
	}
	0 -> press_cmd
	1 -> vent_cmd
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"press_cmd": symbol.Symbol{Name: "press_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
						"press_pt":  symbol.Symbol{Name: "press_pt", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 3},
						"vent_cmd":  symbol.Symbol{Name: "vent_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 4},
					},
					stl.SymbolResolver,
				},
			)

			main := findTopLevelScope(mod, "main")
			Expect(main.Steps).To(HaveLen(3))
			Expect(main.Steps[0].Key()).To(Equal("press"))
			Expect(isStageMember(main.Steps[0])).To(BeTrue())
			Expect(isFlowMember(main.Steps[1])).To(BeTrue())
			Expect(isFlowMember(main.Steps[2])).To(BeTrue())
			Expect(main.Transitions).ToNot(BeEmpty())
		})
	})

	Describe("Top-Level Stages", func() {
		It("Should compile a top-level stage as a single-step sequence", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => abort

stage abort {
	0 -> all_valves,
	1 -> vent_cmd,
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd":  symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"all_valves": symbol.Symbol{Name: "all_valves", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
						"vent_cmd":   symbol.Symbol{Name: "vent_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 3},
					},
					stl.SymbolResolver,
				},
			)

			abort := findTopLevelScope(mod, "abort")
			// Top-level stages are now parallel scopes directly under root,
			// not wrapped in a single-step sequence.
			Expect(abort.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(abort.Key).To(Equal("abort"))
			Expect(scopeNodeRefs(abort)).ToNot(BeEmpty())
		})

		It("Should allow => name from a sequence stage to a top-level stage", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	stage fire {
		1 -> engine_cmd,
		abort_btn => abort,
	}
}

stage abort {
	0 -> engine_cmd,
	1 -> vent_cmd,
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd":  symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"engine_cmd": symbol.Symbol{Name: "engine_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
						"abort_btn":  symbol.Symbol{Name: "abort_btn", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 3},
						"vent_cmd":   symbol.Symbol{Name: "vent_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 4},
					},
					stl.SymbolResolver,
				},
			)

			_ = findTopLevelScope(mod, "main")
			abort := findTopLevelScope(mod, "abort")
			Expect(abort.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(abort.Activation).ToNot(BeNil(), "abort should carry activation from abort_btn => abort")
		})
	})

	Describe("Proto Round-Trip", func() {
		It("Should round-trip a flow step program through proto", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	1 -> valve_cmd
	wait{duration=2s}
	0 -> valve_cmd
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"valve_cmd": symbol.Symbol{Name: "valve_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
					},
					stl.SymbolResolver,
				},
			)

			main := findTopLevelScope(mod, "main")
			Expect(main.Steps).To(HaveLen(3))

			pb := MustSucceed(programpb.ProgramToPB(mod))
			reconstructed := MustSucceed(programpb.ProgramFromPB(pb))

			rMain := findTopLevelScope(reconstructed, "main")
			Expect(rMain.Steps).To(HaveLen(3))
			Expect(isFlowMember(rMain.Steps[0])).To(BeTrue())
			Expect(isFlowMember(rMain.Steps[1])).To(BeTrue())
			Expect(isFlowMember(rMain.Steps[2])).To(BeTrue())
			Expect(reconstructed.Root.Strata).ToNot(BeEmpty())
		})

		It("Should round-trip a mixed stage and flow program through proto", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => main

sequence main {
	stage press {
		1 -> press_cmd,
		press_pt > 50 => next
	}
	0 -> press_cmd
	1 -> vent_cmd
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd": symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"press_cmd": symbol.Symbol{Name: "press_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
						"press_pt":  symbol.Symbol{Name: "press_pt", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 3},
						"vent_cmd":  symbol.Symbol{Name: "vent_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 4},
					},
					stl.SymbolResolver,
				},
			)

			pb := MustSucceed(programpb.ProgramToPB(mod))
			reconstructed := MustSucceed(programpb.ProgramFromPB(pb))

			rMain := findTopLevelScope(reconstructed, "main")
			Expect(rMain.Steps).To(HaveLen(3))
			Expect(isStageMember(rMain.Steps[0])).To(BeTrue())
			Expect(rMain.Steps[0].Key()).To(Equal("press"))
			Expect(isFlowMember(rMain.Steps[1])).To(BeTrue())
			Expect(isFlowMember(rMain.Steps[2])).To(BeTrue())
		})

		It("Should round-trip a top-level stage program through proto", func(ctx SpecContext) {
			mod := compile(ctx, `
start_cmd => abort

stage abort {
	0 -> all_valves,
	1 -> vent_cmd,
}
`,
				symbol.CompoundResolver{
					symbol.MapResolver{
						"start_cmd":  symbol.Symbol{Name: "start_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
						"all_valves": symbol.Symbol{Name: "all_valves", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 2},
						"vent_cmd":   symbol.Symbol{Name: "vent_cmd", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 3},
					},
					stl.SymbolResolver,
				},
			)

			pb := MustSucceed(programpb.ProgramToPB(mod))
			reconstructed := MustSucceed(programpb.ProgramFromPB(pb))

			rAbort := findTopLevelScope(reconstructed, "abort")
			Expect(rAbort.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(scopeNodeRefs(rAbort)).ToNot(BeEmpty())
		})
	})
})

// wasmImport represents a parsed WASM import entry.
type wasmImport struct {
	module string
	name   string
}

// parseWASMImports extracts import entries from raw WASM bytecode.
func parseWASMImports(wasm []byte) []wasmImport {
	if len(wasm) < 8 {
		return nil
	}
	// Skip magic (4 bytes) + version (4 bytes)
	pos := 8
	for pos < len(wasm) {
		sectionID := wasm[pos]
		pos++
		sectionSize, n := binary.Uvarint(wasm[pos:])
		pos += n
		if sectionID != 0x02 {
			// Not the import section — skip
			pos += int(sectionSize)
			continue
		}
		// Parse import section
		sectionEnd := pos + int(sectionSize)
		count, n := binary.Uvarint(wasm[pos:])
		pos += n
		var imports []wasmImport
		for i := 0; i < int(count) && pos < sectionEnd; i++ {
			modLen, n := binary.Uvarint(wasm[pos:])
			pos += n
			modName := string(wasm[pos : pos+int(modLen)])
			pos += int(modLen)
			nameLen, n := binary.Uvarint(wasm[pos:])
			pos += n
			funcName := string(wasm[pos : pos+int(nameLen)])
			pos += int(nameLen)
			// Skip import kind (1 byte) + type index (LEB128)
			pos++ // kind
			_, n = binary.Uvarint(wasm[pos:])
			pos += n
			imports = append(imports, wasmImport{module: modName, name: funcName})
		}
		return imports
	}
	return nil
}
