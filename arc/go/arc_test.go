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
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/authority"
	stlchannel "github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stage"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	stltelem "github.com/synnaxlabs/arc/stl/telem"
	"github.com/synnaxlabs/arc/stl/time"
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

	findNodeByType := func(nodes ir.Nodes, nodeType string) ir.Node {
		for _, n := range nodes {
			if n.Type == nodeType {
				return n
			}
		}
		Fail("node with type " + nodeType + " not found")
		return ir.Node{}
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
		Expect(onNode.Channels.Read.Contains(uint32(1))).To(BeTrue())
		Expect(onNode.Outputs).To(HaveLen(1))
		Expect(onNode.Outputs.Has("output")).To(BeTrue())

		calcNode := findNodeByType(mod.Nodes, "calc")
		Expect(calcNode.Inputs).To(HaveLen(1))
		Expect(calcNode.Inputs.Has("val")).To(BeTrue())
		Expect(calcNode.Outputs).To(HaveLen(1))
		Expect(calcNode.Outputs.Has("output")).To(BeTrue())

		writeNode := findNodeByType(mod.Nodes, "write")
		Expect(writeNode.Channels.Write.Contains(uint32(2))).To(BeTrue())
		Expect(writeNode.Inputs).To(HaveLen(1))

		Expect(mod.Edges).To(HaveLen(2))

		edge1 := MustBeOk(mod.Edges.FindByTarget(ir.Handle{Node: calcNode.Key, Param: "val"}))
		Expect(edge1.Source.Node).To(Equal(onNode.Key))
		Expect(edge1.Source.Param).To(Equal("output"))
		Expect(edge1.Kind).To(Equal(ir.EdgeKindContinuous))

		edge2 := MustBeOk(mod.Edges.FindBySource(ir.Handle{Node: calcNode.Key, Param: "output"}))
		Expect(edge2.Target.Node).To(Equal(writeNode.Key))
		Expect(edge2.Kind).To(Equal(ir.EdgeKindContinuous))

		Expect(mod.Strata).To(HaveLen(3))
		Expect(mod.Strata[0]).To(ContainElement(onNode.Key))
		Expect(mod.Strata[1]).To(ContainElement(calcNode.Key))
		Expect(mod.Strata[2]).To(ContainElement(writeNode.Key))
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

		Expect(mod.Sequences).To(HaveLen(1))
		seq := MustBeOk(mod.Sequences.Find("seg"))
		Expect(seq.Key).To(Equal("seg"))
		Expect(seq.Stages).To(HaveLen(1))
		Expect(seq.Entry().Key).To(Equal("init"))

		initStage := MustBeOk(seq.FindStage("init"))
		Expect(initStage.Nodes).To(HaveLen(2))

		Expect(mod.Nodes).To(HaveLen(3))
		entryNode := findNodeByType(mod.Nodes, "stage_entry")
		Expect(entryNode.Inputs).To(HaveLen(1))
		Expect(entryNode.Inputs.Has("activate")).To(BeTrue())

		constNode := findNodeByType(mod.Nodes, "constant")
		Expect(constNode.Config).To(HaveLen(1))

		writeNode := findNodeByType(mod.Nodes, "write")
		Expect(writeNode.Channels.Write.Contains(uint32(1))).To(BeTrue())

		Expect(mod.Edges).To(HaveLen(1))
		edge := MustBeOk(mod.Edges.FindByTarget(ir.Handle{Node: writeNode.Key, Param: "input"}))
		Expect(edge.Source.Node).To(Equal(constNode.Key))
		Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))

		Expect(initStage.Strata).To(HaveLen(2))
		Expect(initStage.Strata[0]).To(ContainElement(constNode.Key))
		Expect(initStage.Strata[1]).To(ContainElement(writeNode.Key))
	})

	It("Should compile a three stage sequence", func() {
		mod := compile(`
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

		Expect(mod.Sequences).To(HaveLen(1))
		seq := MustBeOk(mod.Sequences.Find("main"))
		Expect(seq.Stages).To(HaveLen(2))

		pressStage := MustBeOk(seq.FindStage("press"))
		Expect(pressStage.Nodes).ToNot(BeEmpty())

		stopStage := MustBeOk(seq.FindStage("stop"))
		Expect(stopStage.Nodes).ToNot(BeEmpty())

		nextStage := MustBeOk(seq.NextStage("press"))
		Expect(nextStage.Key).To(Equal("stop"))

		_, ok := seq.NextStage("stop")
		Expect(ok).To(BeFalse())

		oneShotEdges := mod.Edges.GetByKind(ir.EdgeKindOneShot)
		Expect(oneShotEdges).ToNot(BeEmpty())

		continuousEdges := mod.Edges.GetByKind(ir.EdgeKindContinuous)
		Expect(continuousEdges).ToNot(BeEmpty())
	})

	It("Should correctly generate strata for a loop", func() {
		mod := compile(`
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

		Expect(mod.Sequences).To(HaveLen(1))
		seq := MustBeOk(mod.Sequences.Find("main"))
		Expect(seq.Stages).To(HaveLen(2))

		pressStage := MustBeOk(seq.FindStage("press"))
		Expect(pressStage.Strata).ToNot(BeEmpty())

		ventStage := MustBeOk(seq.FindStage("vent"))
		Expect(ventStage.Strata).ToNot(BeEmpty())

		nextFromPress := MustBeOk(seq.NextStage("press"))
		Expect(nextFromPress.Key).To(Equal("vent"))

		_, ok := seq.NextStage("vent")
		Expect(ok).To(BeFalse())

		oneShotEdges := mod.Edges.GetByKind(ir.EdgeKindOneShot)
		Expect(len(oneShotEdges)).To(BeNumerically(">=", 2))
	})

	It("Should correctly compile a node with a unit literal", func() {
		mod := compile(`
			sequence main {
				stage initial {
					wait{duration=5s} => next
				}
				stage end {
				}
			}
		`, time.SymbolResolver)
		Expect(mod.Nodes).To(HaveLen(3))
	})

	It("Should generate typed state imports for stateful variables", func() {
		// Regression test: stateful variables must produce typed WASM imports
		// like "state::load_i64", not bare "state::load". This mirrors the
		// exact program used in the C++ NodeTest.StatefulVariablesAreIsolatedBetweenNodeInstances.
		modules := []stl.Module{
			stlchannel.NewModule(nil, nil),
			stateful.NewModule(nil, nil),
			series.NewModule(nil),
			stlstrings.NewModule(nil),
			stlmath.NewModule(),
			stlerrors.NewModule(),
			constant.NewModule(),
			stlop.NewModule(),
			selector.NewModule(),
			stable.NewModule(),
			authority.NewModule(nil),
			stltelem.NewModule(),
			stat.NewModule(),
			time.NewModule(),
			stage.NewModule(),
		}
		resolver := stl.CompoundResolver(modules...)
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
		fullResolver := append(resolver, channelResolver)

		mod := compile(`
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
