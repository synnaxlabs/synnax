// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// findNodeByKey finds a node by key and asserts it exists
func findNodeByKey(nodes ir.Nodes, key string) ir.Node {
	node, found := nodes.Find(key)
	ExpectWithOffset(1, found).To(BeTrue(), "expected node '%s' to exist", key)
	return node
}

// findNodeByType finds the first node by type and asserts it exists
func findNodeByType(nodes ir.Nodes, nodeType string) ir.Node {
	for _, n := range nodes {
		if n.Type == nodeType {
			return n
		}
	}
	Fail("expected node with type '" + nodeType + "' to exist")
	return ir.Node{}
}

// findEdgeBySourceParam finds an edge by source parameter name
func findEdgeBySourceParam(edges []ir.Edge, param string) ir.Edge {
	for _, e := range edges {
		if e.Source.Param == param {
			return e
		}
	}
	Fail("expected edge with source param '" + param + "' to exist")
	return ir.Edge{}
}

// countNodesByType counts nodes of a specific type
func countNodesByType(nodes ir.Nodes, nodeType string) int {
	count := 0
	for _, n := range nodes {
		if n.Type == nodeType {
			count++
		}
	}
	return count
}

// findTopLevelScope returns the top-level Scope member whose key matches.
// Fails the spec if no such member exists. Top-level scopes are always
// members of the root scope's first stratum.
func findTopLevelScope(prog ir.IR, key string) ir.Scope {
	for _, stratum := range prog.Root.Strata {
		for _, m := range stratum {
			if m.Scope != nil && m.Scope.Key == key {
				return *m.Scope
			}
		}
	}
	Fail("expected top-level scope '" + key + "' to exist")
	return ir.Scope{}
}

// findMember returns the first direct member of a scope with the matching key.
// Searches both Steps (sequential scopes) and Strata (parallel scopes).
// Fails the spec if no such member exists.
func findMember(scope ir.Scope, key string) ir.Member {
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
// (across all strata and steps). Used to assert that a set of synthesized
// node keys belongs to a particular scope.
func scopeNodeRefs(scope ir.Scope) []string {
	var keys []string
	for _, stratum := range scope.Strata {
		for _, m := range stratum {
			if m.NodeKey != nil {
				keys = append(keys, *m.NodeKey)
			}
		}
	}
	for _, m := range scope.Steps {
		if m.NodeKey != nil {
			keys = append(keys, *m.NodeKey)
		}
	}
	return keys
}

var _ = Describe("Text", func() {
	Describe("Variables", func() {
		varResolver := []symbol.Symbol{
			{Name: "count_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 901},
			{Name: "out_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 902},
			{Name: "flag_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 903},
			{Name: "sink_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 904},
		}

		variableNodes := func(inter ir.IR) []ir.Node {
			var out []ir.Node
			for _, n := range inter.Nodes {
				if n.Type == "variable" || n.Type == "stateful_variable" {
					out = append(out, n)
				}
			}
			return out
		}
		hasEdge := func(inter ir.IR, src, tgt string) bool {
			for _, e := range inter.Edges {
				if e.Source.Node == src && e.Target.Node == tgt {
					return true
				}
			}
			return false
		}
		nodeReading := func(inter ir.IR, key uint32) string {
			for _, n := range inter.Nodes {
				if _, ok := n.Channels.Read[key]; ok {
					return n.Key
				}
			}
			return ""
		}
		nodeWriting := func(inter ir.IR, key uint32) string {
			for _, n := range inter.Nodes {
				if _, ok := n.Channels.Write[key]; ok {
					return n.Key
				}
			}
			return ""
		}

		It("Should lower a written variable to a node wired between channel reads and writes", func(ctx SpecContext) {
			source := `
			sequence main {
				counter i64 := 0
				stage s1 {
					count_ch -> counter
					counter -> out_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(1))
			counter := vars[0]
			Expect(counter.Inputs[0].Value).To(Equal(int64(0)))
			onKey := nodeReading(inter, 901)
			writeKey := nodeWriting(inter, 902)
			Expect(onKey).ToNot(BeEmpty(), "expected an on-node reading count_ch")
			Expect(writeKey).ToNot(BeEmpty(), "expected a write-node writing out_ch")
			Expect(hasEdge(inter, onKey, counter.Key)).To(BeTrue(),
				"the channel read must feed the variable node")
			Expect(hasEdge(inter, counter.Key, writeKey)).To(BeTrue(),
				"the variable node must feed the channel write")
		})

		It("Should reject a flow write to a channel-read variable", func(ctx SpecContext) {
			source := `
			sequence main {
				r i64 := count_ch + 1
				stage s1 {
					count_ch -> r
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.String()).To(ContainSubstring("channel-read variable"))
		})

		It("Should inline a never-reassigned constant flow read as a constant node", func(ctx SpecContext) {
			source := `
			sequence main {
				k i64 := 5
				stage s1 {
					k -> out_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(variableNodes(inter)).To(BeEmpty())
			writeKey := nodeWriting(inter, 902)
			Expect(writeKey).ToNot(BeEmpty())
			constKey := ""
			for _, n := range inter.Nodes {
				if n.Type == "constant" && len(n.Inputs) > 0 &&
					n.Inputs[0].Value == int64(5) {
					constKey = n.Key
				}
			}
			Expect(constKey).ToNot(BeEmpty(), "expected a constant node seeded with 5")
			Expect(hasEdge(inter, constKey, writeKey)).To(BeTrue())
		})

		It("Should lower a written stateful variable to a stateful_variable node", func(ctx SpecContext) {
			source := `
			sequence main {
				total i64 $= 0
				stage s1 {
					count_ch -> total
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(1))
			Expect(vars[0].Type).To(Equal("stateful_variable"))
		})

		It("Should bind a derivation's trigger read to the alias's register", func(ctx SpecContext) {
			source := `
			sequence main {
				p := count_ch
				r i64 := p + 1
				stage s1 {
					p = out_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			found := false
			for _, e := range inter.Edges {
				if e.Target.Param == "channel" && strings.HasPrefix(e.Source.Node, "bind_p") {
					found = true
				}
			}
			Expect(found).To(BeTrue(),
				"expected the alias register to feed a read node's channel param")
		})

		It("Should accept a channel alias as a chained flow head", func(ctx SpecContext) {
			resolver := append([]symbol.Symbol{
				{Name: "sensor_f", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 905},
				{Name: "out_f", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 906},
			}, varResolver...)
			source := `import math
			cpu := sensor_f
			cpu -> math.avg{} -> out_f`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(nodeReading(inter, 905)).ToNot(BeEmpty(),
				"expected an on-node reading the aliased channel")
			Expect(nodeWriting(inter, 906)).ToNot(BeEmpty())
		})

		It("Should route a chained flow's head read through the alias register", func(ctx SpecContext) {
			resolver := append([]symbol.Symbol{
				{Name: "sensor_f", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 905},
				{Name: "backup_f", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 906},
				{Name: "out_f", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 907},
			}, varResolver...)
			source := `import math
			sequence main {
				p := sensor_f
				stage s1 {
					p -> math.avg{} -> out_f
				}
				stage s2 {
					p = backup_f
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			found := false
			for _, e := range inter.Edges {
				if e.Target.Param == "channel" && strings.HasPrefix(e.Source.Node, "bind_p") {
					found = true
				}
			}
			Expect(found).To(BeTrue(),
				"expected the chained head read to bind to the alias register")
		})

		It("Should reject a non-literal config input value", func(ctx SpecContext) {
			source := `count_ch -> wait{duration=1s+1s} -> sink_ch`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.String()).To(ContainSubstring("must be a literal"))
		})

		DescribeTable("Should lower reactive re-expression and alias rebind",
			func(ctx SpecContext, source string) {
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Nodes).ToNot(BeEmpty())
			},
			Entry("reactive variable reassigned to a new expression", `
				sequence main {
					r i64 := count_ch + 1
					stage s1 {
						r = count_ch + 2
					}
				}`),
			Entry("channel read/write alias rebound to another channel", `
				sequence main {
					p := count_ch
					stage s1 {
						count_ch > 0 => s2
					}
					stage s2 {
						p = out_ch
					}
				}`),
			Entry("alias drives a flow after rebind", `
				sequence main {
					p := count_ch
					stage s1 {
						p -> sink_ch
					}
				}`),
			Entry("reactive variable reassigned across two stages", `
				sequence main {
					r i64 := count_ch + 1
					stage s1 {
						count_ch > 5 => s2
						r = count_ch + 2
					}
					stage s2 {
						r = count_ch + 3
					}
				}`),
			Entry("alias rebound in two stages", `
				sequence main {
					p := count_ch
					stage s1 {
						count_ch > 0 => s2
						p = out_ch
					}
					stage s2 {
						p = sink_ch
					}
				}`),
			Entry("reactive variable feeds a flow and is reassigned", `
				sequence main {
					r i64 := count_ch + 1
					stage s1 {
						r -> out_ch
						r = count_ch + 2
					}
				}`),
			Entry("alias is written by a flow", `
				sequence main {
					p := count_ch
					stage s1 {
						out_ch -> p
					}
				}`),
			Entry("alias written by a flow then rebound", `
				sequence main {
					p := count_ch
					stage s1 {
						out_ch -> p
						count_ch > 0 => s2
					}
					stage s2 {
						p = sink_ch
					}
				}`),
			Entry("value variable written then read in a flow", `
				sequence main {
					v i64 := 0
					stage s1 {
						count_ch -> v
						v -> out_ch
					}
				}`),
			Entry("reactive variable reassigned across three stages", `
				sequence main {
					r i64 := count_ch + 1
					stage s1 {
						count_ch > 1 => s2
						r = count_ch + 2
					}
					stage s2 {
						count_ch > 2 => s3
						r = count_ch + 3
					}
					stage s3 {
						r = count_ch + 4
					}
				}`),
			Entry("alias read in a transition then written", `
				sequence main {
					p := count_ch
					stage s1 {
						p > 0 => s2
						out_ch -> p
					}
					stage s2 {
					}
				}`),
			Entry("value variable seeded then reassigned to a literal", `
				sequence main {
					level i64 := 5
					stage s1 {
						count_ch > 0 => s2
					}
					stage s2 {
						level = 10
					}
				}`),
			Entry("value variable copied from another value variable", `
				sequence main {
					v i64 := 0
					w i64 := 0
					stage s1 {
						count_ch -> v
						w = v
					}
				}`),
			Entry("value variable initialized from a reactive variable", `
				sequence main {
					a i64 := count_ch + 1
					b i64 := a + 2
					stage s1 {
						a -> out_ch
						b -> sink_ch
					}
				}`),
			Entry("alias read in an expression in a later stage", `
				sequence main {
					p := count_ch
					stage s1 {
						count_ch > 0 => s2
					}
					stage s2 {
						p + 1 -> out_ch
					}
				}`),
			Entry("reactive variable copied to a value variable", `
				sequence main {
					r i64 := count_ch + 1
					m i64 := 0
					stage s1 {
						m = r
					}
				}`),
		)

		It("Should lower a format-string flow node", func(ctx SpecContext) {
			res := []symbol.Symbol{
				{Name: "count_ch", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 901},
				{Name: "msg_ch", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 906},
			}
			source := `
			sequence main {
				stage s1 {
					f"count={count_ch}" -> msg_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, res...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		DescribeTable("Should seed a variable node with its literal value",
			func(ctx SpecContext, decl string, expected any) {
				// A never-reassigned variable lowers to no node; reassign with
				// the same literal so the register exists.
				lit := strings.SplitN(decl, ":= ", 2)[1]
				source := "sequence main {\n" + decl + "\na = " + lit + "\n}"
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				vars := variableNodes(inter)
				Expect(vars).To(HaveLen(1))
				Expect(vars[0].Inputs[0].Value).To(Equal(expected))
			},
			Entry("i8", `a i8 := -5`, int8(-5)),
			Entry("i16", `a i16 := -5`, int16(-5)),
			Entry("i32", `a i32 := -5`, int32(-5)),
			Entry("i64", `a i64 := -5`, int64(-5)),
			Entry("f32", `a f32 := -2.5`, float32(-2.5)),
			Entry("f64", `a f64 := -2.5`, float64(-2.5)),
			Entry("i8 type minimum", `a i8 := -128`, int8(-128)),
		)

		It("Should reject a write to a channel-read variable", func(ctx SpecContext) {
			source := `
			sequence main {
				r := count_ch + 1
				count_ch -> r
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"cannot write to channel-read variable r"))
		})

		It("Should lower a literal reassignment to a feeder into the variable node", func(ctx SpecContext) {
			source := `
			sequence main {
				msg := "hello"
				msg = "world"
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			var msg *ir.Node
			for _, n := range variableNodes(inter) {
				if n.Inputs[0].Value == "hello" {
					msg = &n
				}
			}
			Expect(msg).ToNot(BeNil(), "expected msg's register seeded with its literal")
			fed := false
			for _, e := range inter.Edges {
				if e.Target.Node == msg.Key {
					fed = true
				}
			}
			Expect(fed).To(BeTrue(), "expected the reassignment to feed the variable node")
		})

		It("Should lower a re-expressed variable to a sel-switched demux", func(ctx SpecContext) {
			source := `
			sequence main {
				r := count_ch + 1
				r = count_ch + 2
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			// r's demux reader and its sel register.
			Expect(vars).To(HaveLen(2))
			var deref, bind ir.Node
			for _, n := range vars {
				if _, ok := n.Inputs.Get("sel"); ok {
					deref = n
				} else {
					bind = n
				}
			}
			Expect(deref.Key).ToNot(BeEmpty(), "expected a demux with a sel input")
			Expect(bind.Inputs[0].Value).To(Equal(uint32(0)),
				"the sel register seeds the declared derivation's index")
			Expect(hasEdge(inter, bind.Key, deref.Key)).To(BeTrue(),
				"the sel register must drive the demux")
			feeders := 0
			for _, e := range inter.Edges {
				if e.Target.Node == deref.Key && e.Source.Node != bind.Key {
					feeders++
				}
			}
			Expect(feeders).To(Equal(2), "both derivations must feed the demux")
		})

		It("seeds the literal register and leaves the derivation reader edge-fed", func(ctx SpecContext) {
			source := `
			sequence main {
				k := 5
				r := count_ch + 1
				k = 6
				r = count_ch + 2
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			// k's register, r's derivation reader, and r's sel register. k needs
			// a reassignment to lower to a register at all.
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(3))
			literalSeeds, unseeded := 0, 0
			for _, n := range vars {
				switch n.Inputs[0].Value {
				case int64(5):
					literalSeeds++
				case nil:
					unseeded++
				}
			}
			Expect(literalSeeds).To(Equal(1), "only k's register carries a literal seed")
			Expect(unseeded).To(Equal(1), "r's derivation reader is edge-fed, not seeded")
		})

		It("DUMP", func(ctx SpecContext) {
			source := `
			sequence s {
				r := count_ch + 1
				r = count_ch + 2
				r -> out_ch
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			var dumpScope func(s *ir.Scope, indent string)
			dumpScope = func(s *ir.Scope, indent string) {
				GinkgoWriter.Printf("%sSCOPE key=%s mode=%v live=%v\n", indent, s.Key, s.Mode, s.Liveness)
				if s.Activation != nil {
					GinkgoWriter.Printf("%s  activation=%s.%s\n", indent, s.Activation.Node, s.Activation.Param)
				}
				for i, st := range s.Strata {
					GinkgoWriter.Printf("%s  stratum %d:\n", indent, i)
					for _, m := range st {
						if m.NodeKey != nil {
							GinkgoWriter.Printf("%s    node %s\n", indent, *m.NodeKey)
						} else if m.Scope != nil {
							dumpScope(m.Scope, indent+"    ")
						}
					}
				}
				for si, m := range s.Steps {
					GinkgoWriter.Printf("%s  step %d:\n", indent, si)
					if m.NodeKey != nil {
						GinkgoWriter.Printf("%s    node %s\n", indent, *m.NodeKey)
					} else if m.Scope != nil {
						dumpScope(m.Scope, indent+"    ")
					}
				}
				for _, t := range s.Transitions {
					tk := "<exit>"
					if t.TargetKey != nil {
						tk = *t.TargetKey
					}
					GinkgoWriter.Printf("%s  transition on=%s.%s -> %s\n", indent, t.On.Node, t.On.Param, tk)
				}
			}
			dumpScope(&inter.Root, "")
			GinkgoWriter.Printf("EDGES:\n")
			for _, e := range inter.Edges {
				GinkgoWriter.Printf("  %s.%s -> %s.%s\n", e.Source.Node, e.Source.Param, e.Target.Node, e.Target.Param)
			}
			GinkgoWriter.Printf("NODES:\n")
			for _, n := range inter.Nodes {
				GinkgoWriter.Printf("  %s type=%s read=%v write=%v\n", n.Key, n.Type, n.Channels.Read, n.Channels.Write)
			}
		})

		It("Should register both channels as rebind candidates on the alias read", func(ctx SpecContext) {
			source := `
			sequence main {
				a := count_ch
				a = out_ch
				a -> sink_ch
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			// Every channel the alias can point at is opened up front; the bind
			// register selects between them at runtime.
			var readNode ir.Node
			for _, n := range inter.Nodes {
				if _, ok := n.Channels.Read[901]; ok {
					readNode = n
				}
			}
			Expect(readNode.Key).ToNot(BeEmpty(),
				"the alias read must register the declared channel")
			Expect(readNode.Channels.Read).To(HaveKey(uint32(902)),
				"the rebound channel must be a read candidate too")
			seeded := false
			for _, n := range variableNodes(inter) {
				if n.Inputs[0].Value == uint32(901) {
					seeded = true
				}
			}
			Expect(seeded).To(BeTrue(),
				"the bind register seeds the declared channel key")
		})

		It("Should reject rebinding an alias to a nonexistent channel", func(ctx SpecContext) {
			source := `
			sequence main {
				a := count_ch
				a = missing_ch
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"cannot rebind channel read/write variable a; the right-hand side must be a channel"))
		})

		It("Should reject rebinding an alias to a non-channel value", func(ctx SpecContext) {
			source := `
			sequence main {
				a := count_ch
				k := 5
				a = k
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"cannot rebind channel read/write variable a; the right-hand side must be a channel"))
		})

		It("Should reject rebinding an alias to a channel of a different type", func(ctx SpecContext) {
			source := `
			sequence main {
				a := count_ch
				a = flag_ch
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring("cannot rebind channel read/write variable a of type"))
		})

		It("Should reject reassigning a variable at the top level", func(ctx SpecContext) {
			source := `
			a := count_ch
			a = out_ch`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"cannot reassign a top-level variable; assignment is only valid " +
					"inside a sequence, stage, or function"))
		})

		It("Should reject compound reassignment of a variable", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 := 0
				c += 1
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"compound and indexed assignment to a variable are not yet supported"))
		})

		It("Should reject redeclaring a variable in the same scope", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 := 0
				c i64 := 1
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"name c conflicts with existing variable"))
		})

		It("Should reject reassigning a variable with an incompatible value type", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 := 0
				c = "hello"
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"cannot assign str to 'c' (type i64)"))
		})

		It("Should reject reading a variable outside its declaring scope", func(ctx SpecContext) {
			source := `
			sequence main {
				stage s1 {
					x i64 := 0
				}
				stage s2 {
					x -> out_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring("undefined symbol: x"))
		})

		It("Should reject assigning a channel read/write to a stateful variable", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 $= count_ch
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"channels and channel-read expressions cannot be assigned to stateful variables"))
		})

		It("Should reject assigning a reactive expression to a stateful variable", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 $= count_ch + 1
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"channels and channel-read expressions cannot be assigned to stateful variables"))
		})

		It("Should reject a computed stateful variable initializer", func(ctx SpecContext) {
			source := `
			sequence main {
				c i64 $= 2 + 3
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"stateful variable initializer must be a literal value"))
		})

		It("Should reject initializing a ':=' variable from a stateful variable", func(ctx SpecContext) {
			source := `
			sequence main {
				s i64 $= 0
				y := s
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
			Expect(diagnostics.String()).To(ContainSubstring(
				"stateful variables cannot be assigned to ':=' variables"))
		})

		It("Should assign distinct keys to variables in sibling sequences", func(ctx SpecContext) {
			source := `
			sequence a {
				x i64 := 0
				stage s1 {
					count_ch -> x
				}
			}
			sequence b {
				y i64 := 0
				stage s1 {
					count_ch -> y
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(2))
			Expect(vars[0].Key).ToNot(Equal(vars[1].Key))
		})

		It("Should compile a variable read inside a transition condition", func(ctx SpecContext) {
			source := `
			sequence main {
				counter i64 := 0
				stage s1 {
					count_ch -> counter
					counter > 5 => s2
				}
				stage s2 {
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			prog := MustSucceed(text.Compile(ctx, inter))
			Expect(prog.WASM).ToNot(BeEmpty())
		})

		It("Should lower a stage-local variable to a node wired between channel reads and writes", func(ctx SpecContext) {
			source := `
			sequence main {
				stage s1 {
					counter i64 := 0
					count_ch -> counter
					counter -> out_ch
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(1))
			counter := vars[0]
			Expect(counter.Inputs[0].Value).To(Equal(int64(0)))
			onKey := nodeReading(inter, 901)
			writeKey := nodeWriting(inter, 902)
			Expect(onKey).ToNot(BeEmpty(), "expected an on-node reading count_ch")
			Expect(writeKey).ToNot(BeEmpty(), "expected a write-node writing out_ch")
			Expect(hasEdge(inter, onKey, counter.Key)).To(BeTrue(),
				"the channel read must feed the variable node")
			Expect(hasEdge(inter, counter.Key, writeKey)).To(BeTrue(),
				"the variable node must feed the channel write")
		})

		It("Should assign distinct keys to stage-local variables in sibling stages", func(ctx SpecContext) {
			source := `
			sequence main {
				stage s1 {
					x i64 := 0
					count_ch -> x
				}
				stage s2 {
					y i64 := 0
					count_ch -> y
				}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, varResolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			vars := variableNodes(inter)
			Expect(vars).To(HaveLen(2))
			Expect(vars[0].Key).ToNot(Equal(vars[1].Key))
		})
	})

	Describe("Parse", func() {
		It("Should correctly parse a text-based arc program", func() {
			source := `
			func add(a i64, b i64) i64 {
			    return a + b
			}

			func adder{} (a i64, b i64) i64 {
			    return add(a, b)
			}

			func print{} () {}

			adder{} -> print{}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
		})
	})

	Describe("Analyze", func() {
		It("Should correctly analyze a text-based arc program", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "in", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 100},
			}
			source := `
			func add(a i64, b i64) i64 {
			    return a + b
			}

			func adder{} (a i64) i64 {
			    return a
			}

			func print{} () {}

			in -> adder{} -> print{}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			Expect(parsedText.AST).ToNot(BeNil())
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Functions).To(HaveLen(3))
			Expect(inter.Nodes).To(HaveLen(3))
			Expect(inter.Edges).To(HaveLen(2))

			f := inter.Functions[0]
			Expect(f.Key).To(Equal("add"))
			Expect(f.Inputs).To(HaveLen(2))
			Expect(f.Inputs[0].Type).To(Equal(types.I64()))
			Expect(f.Inputs[1].Type).To(Equal(types.I64()))

			s := inter.Functions[1]
			Expect(s.Key).To(Equal("adder"))
			Expect(s.Inputs).To(HaveLen(1))
			Expect(s.Inputs[0].Type).To(Equal(types.I64()))

			n1 := findNodeByKey(inter.Nodes, "adder_0")
			Expect(n1.Type).To(Equal("adder"))
			Expect(n1.Inputs).To(HaveLen(1))
			Expect(n1.Inputs[0].Name).To(Equal("a"))
			Expect(n1.Channels.Read).ToNot(BeNil())
			Expect(n1.Channels.Read).To(BeEmpty())
			Expect(n1.Channels.Write).ToNot(BeNil())
			Expect(n1.Channels.Write).To(BeEmpty())
		})

		Context("Channel Flow Analysis", func() {
			It("Should analyze flow with channel identifier", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10042},
				}
				source := `
				func print{} () {}

				sensor -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Edges).To(HaveLen(1))

				channelNode := findNodeByKey(inter.Nodes, "on_sensor_0")
				Expect(channelNode.Type).To(Equal("on"))
				Expect(channelNode.Inputs).To(HaveLen(1))
				Expect(channelNode.Inputs[0].Name).To(Equal("channel"))
				Expect(channelNode.Inputs[0].Type).To(Equal(types.Chan(types.I32())))
				Expect(channelNode.Channels.Read).To(HaveKey(uint32(10042)))

				printNode := findNodeByKey(inter.Nodes, "print_0")
				Expect(printNode.Type).To(Equal("print"))

				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal("on_sensor_0"))
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			It("Should report error for unresolved channel", func(ctx SpecContext) {
				source := `
				func print{} () {}

				unknown_channel -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("unknown_channel"))
			})
		})

		Context("Expression Flow Analysis", func() {
			It("Should analyze flow with expression nodes", func(ctx SpecContext) {
				source := `
				func add(a i64, b i64) i64 {
				    return a + b
				}

				func print{} () {}

				add(1, 2) -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Edges).To(HaveLen(1))

				exprNode := inter.Nodes[0]
				Expect(exprNode.Key).ToNot(BeEmpty())
				Expect(exprNode.Type).ToNot(BeEmpty())

				printNode := findNodeByKey(inter.Nodes, "print_0")
				Expect(printNode.Type).To(Equal("print"))

				edge := inter.Edges[0]
				Expect(edge.Target.Node).To(Equal(printNode.Key))
			})

			DescribeTable("Literal constant generation",
				func(ctx SpecContext, source string, chans []symbol.Symbol, expectConstant bool, expectedType types.Type) {
					parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
					root := symbol.NewRoot(nil, stl.NewSymbols())
					for i := range chans {
						s := chans[i]
						root.Parent.AddChild(&s)
					}
					inter, diagnostics := text.Analyze(ctx, parsedText, root)
					Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

					constantCount := countNodesByType(inter.Nodes, "constant")
					if expectConstant {
						Expect(constantCount).To(Equal(1), "expected exactly one constant node")
						constantNode := findNodeByType(inter.Nodes, "constant")
						Expect(constantNode.Inputs).To(HaveLen(1))
						Expect(constantNode.Inputs[0].Name).To(Equal("value"))
						Expect(constantNode.Inputs[0].Type).To(Equal(expectedType))
					} else {
						Expect(constantCount).To(Equal(0), "expected no constant nodes for complex expressions")
					}
				},
				Entry("integer literal",
					`1 -> output`,
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10001},
					},
					true, types.F32(),
				),
				Entry("float literal",
					`3.14 -> output`,
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10002},
					},
					true, types.F64(),
				),
				Entry("string literal",
					`"hello" -> output`,
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 10004},
					},
					true, types.String(),
				),
				Entry("multi-line string literal",
					"`hello\nworld` -> output",
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 10005},
					},
					true, types.String(),
				),
				Entry("raw string literal",
					`r"C:\path" -> output`,
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 10006},
					},
					true, types.String(),
				),
				Entry("raw multi-line string literal",
					"r`line1\\n\nline2` -> output",
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 10007},
					},
					true, types.String(),
				),
				Entry("complex expression (should not generate constant)",
					`1 + 2 -> output`,
					[]symbol.Symbol{
						{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10003},
					},
					false, types.Type{}, // Type ignored when expectConstant is false
				),
			)

			It("Should reject a negative constant that is out of range for its target", func(ctx SpecContext) {
				parsedText := MustSucceed(text.Parse(text.Text{Raw: `-5 -> output`}))
				root := symbol.NewRoot(nil, stl.NewSymbols())
				root.Parent.AddChild(&symbol.Symbol{
					Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10098,
				})
				_, diagnostics := text.Analyze(ctx, parsedText, root)
				Expect(diagnostics.Ok()).To(BeFalse(), diagnostics.String())
				Expect(diagnostics.String()).To(ContainSubstring("out of range for u8"))
			})
		})

		Context("Input Values", func() {
			It("Should extract named input values", func(ctx SpecContext) {
				source := `
				func processor{threshold i64, scale f64} () i64 {
				    return threshold
				}

				func print{} () {}

				processor{threshold=100, scale=2.5} -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				node := findNodeByKey(inter.Nodes, "processor_0")
				Expect(node.Type).To(Equal("processor"))
				Expect(node.Inputs).To(HaveLen(2))
				Expect(node.Inputs[0].Name).To(Equal("threshold"))
				Expect(node.Inputs[0].Type).To(Equal(types.I64()))
				Expect(node.Inputs[0].Value).To(Equal(int64(100)))
				Expect(node.Inputs[1].Name).To(Equal("scale"))
				Expect(node.Inputs[1].Type).To(Equal(types.F64()))
				Expect(node.Inputs[1].Value).To(Equal(2.5))
			})

			It("Should handle simple input with multiple values", func(ctx SpecContext) {
				source := `
				func calculator{a i64, b i64, c i64} () i64 {
				    return a + b + c
				}

				func print{} () {}

				calculator{a=10, b=20, c=30} -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "calculator_0")
				Expect(node.Type).To(Equal("calculator"))
				Expect(node.Inputs).To(HaveLen(3))

				inputValues := map[string]int64{
					"a": 10, "b": 20, "c": 30,
				}
				for i, cfg := range node.Inputs {
					Expect(cfg.Type).To(Equal(types.I64()))
					Expect(cfg.Value).To(Equal(inputValues[cfg.Name]), "input[%d] '%s' value mismatch", i, cfg.Name)
				}
			})

			It("Should handle negated integer input value", func(ctx SpecContext) {
				source := `
				func processor{
					threshold i64
				} () i64 {
					return threshold
				}

				func print{} () {
				}

				processor{threshold=-100} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "processor_0")
				Expect(node.Inputs).To(HaveLen(1))
				Expect(node.Inputs[0].Name).To(Equal("threshold"))
				Expect(node.Inputs[0].Value).To(Equal(int64(-100)))
			})

			It("Should handle a type-minimum input value whose magnitude overflows the target", func(ctx SpecContext) {
				source := `
				func processor{
					threshold i8
				} () i8 {
					return threshold
				}

				func print{} () {
				}

				processor{threshold=-128} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "processor_0")
				Expect(node.Inputs).To(HaveLen(1))
				Expect(node.Inputs[0].Name).To(Equal("threshold"))
				Expect(node.Inputs[0].Value).To(Equal(int8(-128)))
			})

			It("Should handle negated float input value", func(ctx SpecContext) {
				source := `
				func scaler{
					factor f64
				} () f64 {
					return factor
				}

				func print{} () {
				}

				scaler{factor=-2.5} -> print{}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "scaler_0")
				Expect(node.Inputs).To(HaveLen(1))
				Expect(node.Inputs[0].Name).To(Equal("factor"))
				Expect(node.Inputs[0].Value).To(Equal(-2.5))
			})

			It("Should handle negated time unit input value", func(ctx SpecContext) {
				source := `
				import time
				time_trigger -> time.wait{duration=-3h} -> wait_out
				`
				resolver := []symbol.Symbol{
					{Name: "time_trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10042},
					{Name: "wait_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10043},
				}
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				waitNode := findNodeByType(inter.Nodes, "time.wait")
				Expect(waitNode).ToNot(BeNil())
				Expect(waitNode.Inputs).To(HaveLen(1))
				Expect(waitNode.Inputs[0].Name).To(Equal("duration"))
				threeHoursNanos := int64(3*60*60) * int64(telem.Second)
				Expect(waitNode.Inputs[0].Value).To(Equal(telem.TimeSpan(-threeHoursNanos)))
			})

			It("Should resolve channel name to channel ID in input parameter", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "temp_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10042},
				}
				source := `
				func reader{channel chan f64} () f64 {
				    return channel
				}

				func display{} (value f64) {}

				reader{channel=temp_sensor} -> display{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				readerNode := findNodeByKey(inter.Nodes, "reader_0")
				Expect(readerNode.Inputs).To(HaveLen(1))
				Expect(readerNode.Inputs[0].Name).To(Equal("channel"))
				Expect(readerNode.Inputs[0].Type).To(Equal(types.Chan(types.F64())))
				Expect(readerNode.Inputs[0].Value).To(Equal(uint32(10042)))
				Expect(readerNode.Channels.Read).To(HaveKey(uint32(10042)))
			})

			It("Should produce diagnostic error when channel input type mismatches", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "temp_sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10043},
				}
				source := `
				func reader{channel chan f64} () f64 {
				    return channel
				}

				func display{} (value f64) {}

				reader{channel=temp_sensor} -> display{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse())
				diagStr := diagnostics.String()
				Expect(diagStr).To(ContainSubstring("type mismatch"))
				Expect(diagStr).To(ContainSubstring("channel"))
				Expect(diagStr).To(ContainSubstring("expected f64"))
				Expect(diagStr).To(ContainSubstring("got i32"))
			})

			It("Should produce diagnostic error when channel name is not found in resolver", func(ctx SpecContext) {
				resolver := []symbol.Symbol{}
				source := `
				func reader{channel chan f64} () f64 {
				    return channel
				}

				func display{} (value f64) {}

				reader{channel=unknown_sensor} -> display{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse())
				diagStr := diagnostics.String()
				Expect(diagStr).To(ContainSubstring("undefined symbol"))
				Expect(diagStr).To(ContainSubstring("unknown_sensor"))
			})

			It("Should reject read channel for input param requiring write channel", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "read_sensor",
						Kind: symbol.KindChannel,
						Type: types.ReadChan(types.F64()),
						ID:   10056,
					},
				}
				source := `
				func source{} () u8 {
				    return 1
				}

				source{} -> set_authority{value=200, channel=read_sensor}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse())
			})

			It("Should reject read channel for qualified control.set_authority input param", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "read_sensor",
						Kind: symbol.KindChannel,
						Type: types.ReadChan(types.F64()),
						ID:   10056,
					},
				}
				source := `
				func source{} () u8 {
					return 1
				}

				source{} -> control.set_authority{value=200, channel=read_sensor}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse())
			})

			It("Should emit deprecation warning for deprecated bare function name", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "write_ch",
						Kind: symbol.KindChannel,
						Type: types.WriteChan(types.U8()),
						ID:   10060,
					},
				}
				source := `
				func source{} () u8 {
					return 1
				}

				source{} -> set_authority{value=200, channel=write_ch}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("control.set_authority"))
			})

			It("Should not emit deprecation warning for qualified function name", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "write_ch",
						Kind: symbol.KindChannel,
						Type: types.WriteChan(types.U8()),
						ID:   10060,
					},
				}
				source := `
				import control
				func source{} () u8 {
					return 1
				}

				source{} -> control.set_authority{value=200, channel=write_ch}
				`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should emit deprecation warning for bare avg", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.F64()), ID: 200},
				}
				source := `sensor -> avg{} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("math.avg"))
			})

			It("Should not emit deprecation warning for qualified math.avg", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.F64()), ID: 200},
				}
				source := `import math
sensor -> math.avg{} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should not emit deprecation warning for bare select", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `flag -> select{} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should not resolve qualified selector.select", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `flag -> selector.select{} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeFalse())
			})

			It("Should reject a flow node whose required input is never connected", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
					{Name: "log", Kind: symbol.KindChannel, Type: types.WriteChan(types.String()), ID: 200},
				}
				// The juxtaposition `(1 > 0) select{}` never wires the comparison
				// into select's selector, leaving select's required u8 input with
				// no source. Previously this compiled and panicked at runtime when
				// the node state tried to materialize a series from a nil value.
				source := `trigger -> (1 > 0) select{} -> { true: "hello" + "!" -> log,}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeFalse())
				Expect(diags.String()).To(ContainSubstring("missing required argument"))
			})

			It("Should emit deprecation warning for bare stable_for", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `sensor -> stable_for{duration=1s} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("stable.for"))
			})

			It("Should not emit deprecation warning for qualified stable.for", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `import stable
sensor -> stable.for{duration=1s} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should emit deprecation warning for bare set_status", func(ctx SpecContext) {
				statusFnType := types.Function(types.FunctionProperties{
					Inputs: types.Params{
						{Name: "status_key", Type: types.String()},
						{Name: "variant", Type: types.String()},
						{Name: "message", Type: types.String()},
					},
				})
				statusModule := &symbol.Symbol{Name: "status", Kind: symbol.KindModule}
				statusModule.AddChild(&symbol.Symbol{
					Name: "set",
					Kind: symbol.KindFunction,
					Exec: symbol.ExecFlow,
					Type: statusFnType,
				})
				statusResolver := []symbol.Symbol{
					{
						Name:       "set_status",
						Kind:       symbol.KindFunction,
						Exec:       symbol.ExecFlow,
						Deprecated: statusModule.FindChild("set"),
						Type:       statusFnType,
					},
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
				}
				source := `sensor -> set_status{status_key="alarm", variant="error", message="Bad"}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, statusResolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("status.set"))
			})

			It("Should not emit deprecation warning for qualified status.set", func(ctx SpecContext) {
				statusFnType := types.Function(types.FunctionProperties{
					Inputs: types.Params{
						{Name: "status_key", Type: types.String()},
						{Name: "variant", Type: types.String()},
						{Name: "message", Type: types.String()},
					},
				})
				statusModule := &symbol.Symbol{Name: "status", Kind: symbol.KindModule}
				statusModule.AddChild(&symbol.Symbol{
					Name: "set",
					Kind: symbol.KindFunction,
					Exec: symbol.ExecFlow,
					Type: statusFnType,
				})
				root := NewRoot(nil, symbol.Symbol{
					Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100,
				})
				// Attach the status module to the ambient prelude so import resolves.
				root.Parent.AddChild(statusModule)
				source := `import status
sensor -> status.set{status_key="alarm", variant="error", message="Bad"}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, root)
				Expect(diags.Ok()).To(BeTrue(), diags.String())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should emit deprecation warning for bare derivative", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.F64()), ID: 200},
				}
				source := `sensor -> derivative{} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("math.derivative"))
			})

			It("Should emit deprecation warning for bare interval", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `interval{period=100ms} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("time.interval"))
			})

			It("Should not emit deprecation warning for qualified time.interval", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `import time
time.interval{period=100ms} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should emit deprecation warning for bare wait", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `wait{duration=500ms} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(HaveLen(1))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("deprecated"))
				Expect(diags.Warnings()[0].Message).To(ContainSubstring("time.wait"))
			})

			It("Should not emit deprecation warning for qualified time.wait", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 200},
				}
				source := `import time
time.wait{duration=500ms} -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diags.Ok()).To(BeTrue())
				Expect(diags.Warnings()).To(BeEmpty())
			})

			It("Should resolve channel name for write operations and add to Channels.Write", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output_channel", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10055},
				}
				source := `
				func writer{channel chan f64} (value f64) {
				    channel = value
				}

				func source{} () f64 {
				    return 1.0
				}

				source{} -> writer{channel=output_channel}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				writerNode := findNodeByKey(inter.Nodes, "writer_0")
				Expect(writerNode.Inputs).To(HaveLen(2))
				Expect(writerNode.Inputs[0].Name).To(Equal("channel"))
				Expect(writerNode.Inputs[0].Type).To(Equal(types.Chan(types.F64())))
				Expect(writerNode.Inputs[0].Value).To(Equal(uint32(10055)))
				Expect(writerNode.Channels.Write).To(HaveKey(uint32(10055)))
				Expect(writerNode.Channels.Read).NotTo(HaveKey(uint32(10055)))
			})

			It("Should register separate write channels when function with channel input is used multiple times", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "toggle_1", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10011},
					{Name: "toggle_2", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10012},
					{Name: "counter_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10013},
					{Name: "counter_2", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10014},
				}
				source := `
				func count_rising{counter chan f32} (input u8) {
				    prev $= input
				    if input != 0 and prev == 0 {
				        counter = counter + 1.0
				    }
				    prev = input
				}

				toggle_1 -> count_rising{counter=counter_1}
				toggle_2 -> count_rising{counter=counter_2}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Find the two count_rising nodes
				node1 := findNodeByKey(inter.Nodes, "count_rising_0")
				node2 := findNodeByKey(inter.Nodes, "count_rising_1")

				// Each node should have its own write channel
				Expect(node1.Channels.Write).To(HaveKey(uint32(10013)), "first node should write to counter_1")
				Expect(node2.Channels.Write).To(HaveKey(uint32(10014)), "second node should write to counter_2")

				Expect(node1.Channels.Read).To(HaveKey(uint32(10013)), "first node should read from counter_1")
				Expect(node2.Channels.Read).To(HaveKey(uint32(10014)), "second node should read from counter_2")
			})

			It("Should not add stateful variable to write channels when initialized from global channel", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "toggle_1", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10101},
					{Name: "counter_1", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10102},
				}
				source := `
				func count_rising(input u8) {
				    counter $= counter_1
				    prev $= input
				    if input != 0 and prev == 0 {
				        counter = counter + 1.0
				    }
				    prev = input
				}

				toggle_1 -> count_rising{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Find the count_rising node
				node := findNodeByKey(inter.Nodes, "count_rising_0")

				// counter_1 should be in Read (stateful var is initialized from channel value)
				Expect(node.Channels.Read).To(HaveKey(uint32(10102)), "should read from counter_1")
				// Write channels should be empty - we write to a stateful variable, not a channel
				Expect(node.Channels.Write).To(BeEmpty(), "should not have any write channels")
			})

			It("Should resolve read-only input param channel in Channels.Read", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "do_0_state", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10201},
					{Name: "do_0_counter", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10202},
					{Name: "do_0_counter_max", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10203},
				}
				source := `
				func count_rising_test{counter_ch chan f32, max_ch chan f32} (input u8) {
				    prev $= input
				    counter f32 $= 0
				    read_val := max_ch + f32(0.0)

				    if counter < read_val {
				        counter = read_val
				    }

				    if input and not prev {
				        counter = counter + 1.0
				    }

				    counter_ch = counter
				    prev = input
				}

				do_0_state -> count_rising_test{counter_ch=do_0_counter, max_ch=do_0_counter_max}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "count_rising_test_0")
				Expect(node.Channels.Write).To(HaveKey(uint32(10202)), "should write to do_0_counter")
				Expect(node.Channels.Read).To(HaveKey(uint32(10203)), "should read from do_0_counter_max")
				Expect(node.Inputs).To(HaveLen(3))
				Expect(node.Inputs[0].Value).To(Equal(uint32(10202)))
				Expect(node.Inputs[1].Value).To(Equal(uint32(10203)))
			})

			It("Should handle input values using global constants", func(ctx SpecContext) {
				source := `
				A := 10
				B := 20
				C := 30

				func calculator{a i64, b i64, c i64} () i64 {
				    return a + b + c
				}

				func print{} () {}

				calculator{a=A, b=B, c=C} -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "calculator_0")
				Expect(node.Type).To(Equal("calculator"))
				Expect(node.Inputs).To(HaveLen(3))

				inputValues := map[string]int64{
					"a": 10, "b": 20, "c": 30,
				}
				for i, cfg := range node.Inputs {
					Expect(cfg.Type).To(Equal(types.I64()))
					Expect(cfg.Value).To(Equal(inputValues[cfg.Name]), "input[%d] '%s' value mismatch", i, cfg.Name)
				}
			})

			It("Should handle f64 global constants in input", func(ctx SpecContext) {
				source := `
				SCALE := 2.5
				OFFSET := 0.1

				func transform{scale f64, offset f64} (x f64) f64 {
				    return x * scale + offset
				}

				func sink{} () {}

				0.0 -> transform{scale=SCALE, offset=OFFSET} -> sink{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "transform_0")
				Expect(node.Inputs).To(HaveLen(3))

				inputValues := map[string]float64{
					"scale": 2.5, "offset": 0.1,
				}
				for _, cfg := range node.Inputs {
					if cfg.Value == nil {
						continue
					}
					Expect(cfg.Type).To(Equal(types.F64()))
					Expect(cfg.Value).To(Equal(inputValues[cfg.Name]))
				}
			})

			It("Should handle mixed literal and constant input values", func(ctx SpecContext) {
				source := `
				THRESHOLD := 100

				func filter{threshold i64, enabled i64} (x i64) i64 {
				    return x
				}

				func sink{} () {}

				0 -> filter{threshold=THRESHOLD, enabled=1} -> sink{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "filter_0")
				Expect(node.Inputs).To(HaveLen(3))

				for _, cfg := range node.Inputs {
					switch cfg.Name {
					case "threshold":
						Expect(cfg.Value).To(Equal(int64(100)))
					case "enabled":
						Expect(cfg.Value).To(Equal(int64(1)))
					}
				}
			})

			It("Should handle typed global constants in input", func(ctx SpecContext) {
				source := `
				MAX_VALUE i32 := 255

				func clamp{max i32} (x i32) i32 {
				    return x
				}

				func sink{} () {}

				i32(0) -> clamp{max=MAX_VALUE} -> sink{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "clamp_0")
				Expect(node.Inputs).To(HaveLen(2))
				Expect(node.Inputs[0].Name).To(Equal("max"))
				Expect(node.Inputs[0].Type).To(Equal(types.I32()))
				Expect(node.Inputs[0].Value).To(Equal(int32(255)))
			})

			It("Should resolve anonymous input values by position into IR nodes", func(ctx SpecContext) {
				source := `
				func transform{scale f64, offset f64} (x f64) f64 {
				    return x * scale + offset
				}

				func sink{} () {}

				0.0 -> transform{2.5, 0.1} -> sink{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "transform_0")
				Expect(node.Inputs).To(HaveLen(3))
				Expect(node.Inputs[0].Name).To(Equal("scale"))
				Expect(node.Inputs[0].Value).To(Equal(2.5))
				Expect(node.Inputs[1].Name).To(Equal("offset"))
				Expect(node.Inputs[1].Value).To(Equal(0.1))
			})

			It("Should resolve partial anonymous input with defaults into IR nodes", func(ctx SpecContext) {
				source := `
				func controller{setpoint f64, gain f64 = 1.0} (x f64) f64 {
				    return x
				}

				func sink{} () {}

				0.0 -> controller{100.0} -> sink{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "controller_0")
				Expect(node.Inputs).To(HaveLen(3))
				Expect(node.Inputs[0].Name).To(Equal("setpoint"))
				Expect(node.Inputs[0].Value).To(Equal(100.0))
				Expect(node.Inputs[1].Name).To(Equal("gain"))
				Expect(node.Inputs[1].Value).To(Equal(1.0))
			})

			It("Should resolve channel identifier as anonymous input into IR", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10001},
				}
				source := `
				func controller{sensor chan f64, setpoint f64} () {
				    v := sensor
				}

				sensor_chan -> controller{sensor_chan, 100.0}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				node := findNodeByKey(inter.Nodes, "controller_0")
				Expect(node.Inputs).To(HaveLen(2))
				Expect(node.Inputs[0].Name).To(Equal("sensor"))
				Expect(node.Inputs[1].Name).To(Equal("setpoint"))
				Expect(node.Inputs[1].Value).To(Equal(100.0))
			})
		})

		Context("Edge Parameter Validation", func() {
			It("Should create edges with parameters that exist in node definitions", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10001},
				}
				source := `
				func filter{} (data i64) i64 {
				    return data
				}

				func transform{} (value i64) i64 {
				    return value * 2
				}

				sensor -> filter{} -> transform{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))
				Expect(inter.Edges).To(HaveLen(2))

				// Verify Edge 0: sensor -> filter
				edge0 := inter.Edges[0]
				srcNode0 := findNodeByKey(inter.Nodes, edge0.Source.Node)
				tgtNode0 := findNodeByKey(inter.Nodes, edge0.Target.Node)

				Expect(srcNode0.Key).To(Equal("on_sensor_0"))
				Expect(edge0.Source.Param).To(Equal("output"))
				Expect(srcNode0.Outputs.Has(edge0.Source.Param)).To(BeTrue())

				Expect(tgtNode0.Key).To(Equal("filter_0"))
				Expect(edge0.Target.Param).To(Equal("data"))
				Expect(tgtNode0.Inputs.Has(edge0.Target.Param)).To(BeTrue())

				// Verify Edge 1: filter -> transform
				edge1 := inter.Edges[1]
				srcNode1 := findNodeByKey(inter.Nodes, edge1.Source.Node)
				tgtNode1 := findNodeByKey(inter.Nodes, edge1.Target.Node)

				Expect(srcNode1.Key).To(Equal("filter_0"))
				Expect(edge1.Source.Param).To(Equal("output"))
				Expect(srcNode1.Outputs.Has(edge1.Source.Param)).To(BeTrue())

				Expect(tgtNode1.Key).To(Equal("transform_0"))
				Expect(edge1.Target.Param).To(Equal("value"))
				Expect(tgtNode1.Inputs.Has(edge1.Target.Param)).To(BeTrue())
			})

			It("Should handle functions with custom input parameter names", func(ctx SpecContext) {
				source := `
				func generator{} () i64 {
				    return 42
				}

				func processor{} (inputValue i64) i64 {
				    return inputValue * 2
				}

				generator{} -> processor{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				srcNode := findNodeByKey(inter.Nodes, edge.Source.Node)
				tgtNode := findNodeByKey(inter.Nodes, edge.Target.Node)

				Expect(edge.Source.Param).To(Equal("output"))
				Expect(srcNode.Outputs.Has("output")).To(BeTrue())

				Expect(edge.Target.Param).To(Equal("inputValue"))
				Expect(tgtNode.Inputs.Has("inputValue")).To(BeTrue())
			})

			It("Should verify channel node outputs are defined", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "temp", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10044},
				}
				source := `
				func display{} (value f64) {}

				temp -> display{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				channelNode := findNodeByKey(inter.Nodes, "on_temp_0")
				Expect(channelNode.Outputs).To(HaveLen(1))
				Expect(channelNode.Outputs[0].Name).To(Equal("output"))
				Expect(channelNode.Outputs[0].Type).To(Equal(types.F64()))

				edge := inter.Edges[0]
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(channelNode.Outputs.Has(edge.Source.Param)).To(BeTrue())
			})

			It("Should reject a multi-input function used as a bare flow node", func(ctx SpecContext) {
				// A flow node receives a single value from its upstream, so a
				// function with more than one required input cannot have all of
				// its inputs sourced in flow form. The unconnected inputs are
				// reported as diagnostics rather than producing IR that panics
				// at runtime when materializing a series from a nil value.
				source := `
				func add{} (a i64, b i64) i64 {
				    return a + b
				}

				func print{} (value i64) {}

				add{} -> print{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(SatisfyAll(
					ContainSubstring("missing required argument for parameter 'a'"),
					ContainSubstring("missing required argument for parameter 'b'"),
				))
			})
		})

		Context("Output Routing Tables", func() {
			It("Should analyze simple output routing with multiple targets", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10101},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
				    if (value > threshold) {
				        high = value
				    } else {
				        low = value
				    }
				}

				func alarm{} (value f64) {}

				func logger{} (value f64) {}

				signal -> demux{threshold=100.0} -> {
				    high: alarm{},
				    low: logger{}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))
				Expect(inter.Edges).To(HaveLen(3))

				demuxNode := findNodeByKey(inter.Nodes, "demux_0")
				alarmNode := findNodeByKey(inter.Nodes, "alarm_0")
				loggerNode := findNodeByKey(inter.Nodes, "logger_0")

				Expect(demuxNode.Outputs).To(HaveLen(2))
				Expect(demuxNode.Outputs.Has("high")).To(BeTrue())
				Expect(demuxNode.Outputs.Has("low")).To(BeTrue())

				highEdge := findEdgeBySourceParam(inter.Edges, "high")
				Expect(highEdge.Source.Node).To(Equal("demux_0"))
				Expect(highEdge.Target.Node).To(Equal(alarmNode.Key))
				Expect(highEdge.Target.Param).To(Equal("value"))

				lowEdge := findEdgeBySourceParam(inter.Edges, "low")
				Expect(lowEdge.Source.Node).To(Equal("demux_0"))
				Expect(lowEdge.Target.Node).To(Equal(loggerNode.Key))
				Expect(lowEdge.Target.Param).To(Equal("value"))
			})

			It("Should handle routing with chained processing", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10102},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
				    if (value > threshold) {
				        high = value
				    } else {
				        low = value
				    }
				}

				func amplify{} (signal f64) f64 {
				    return signal * 2
				}

				func display{} (value f64) {}

				signal -> demux{threshold=100.0} -> {
				    high: amplify{} -> display{}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))
				Expect(inter.Edges).To(HaveLen(3))

				demuxToAmplify := findEdgeBySourceParam(inter.Edges, "high")
				Expect(demuxToAmplify.Source.Node).To(Equal("demux_0"))
				Expect(demuxToAmplify.Target.Node).To(Equal("amplify_0"))

				var amplifyToDisplay ir.Edge
				for _, e := range inter.Edges {
					if e.Source.Node == "amplify_0" {
						amplifyToDisplay = e
					}
				}
				Expect(amplifyToDisplay.Target.Node).To(Equal("display_0"))
			})

			It("Should not create phantom output edges for void functions in routing branches", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "counter", Kind: symbol.KindChannel, Type: types.Chan(types.U32()), ID: 10301},
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10302},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
				    if (value > threshold) {
				        high = value
				    } else {
				        low = value
				    }
				}

				func increment{ch chan u32} () {
				    ch = ch + 1
				}

				signal -> demux{threshold=100.0} -> {
				    high: increment{ch=counter}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				outputSet := make(set.Set[ir.Handle])
				for _, n := range inter.Nodes {
					for _, p := range n.Outputs {
						outputSet.Add(ir.Handle{Node: n.Key, Param: p.Name})
					}
				}
				for _, edge := range inter.Edges {
					Expect(outputSet).To(HaveKey(edge.Source),
						"edge source %v references a non-existent node output", edge.Source)
				}
			})

			It("Should report error for non-existent output parameter", func(ctx SpecContext) {
				source := `
				func simple{} () (bob i64) {
				    bob = 42
				}

				func display{} (value i64) {}

				simple{} -> {
				    nonexistent: display{}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("nonexistent"))
			})
		})

		Context("Stratification", func() {
			It("Should calculate strata for simple flow chain", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10001},
				}
				source := `
				func filter{} (data i64) i64 {
				    return data
				}

				func transform{} (value i64) i64 {
				    return value * 2
				}

				sensor -> filter{} -> transform{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Root.Strata).To(HaveLen(3))
				Expect(inter.Root.Strata[0][0].Key()).To(Equal("on_sensor_0"))
				Expect(inter.Root.Strata[1][0].Key()).To(Equal("filter_0"))
				Expect(inter.Root.Strata[2][0].Key()).To(Equal("transform_0"))
			})

			It("Should calculate strata for output routing tables", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10103},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
				    if (value > threshold) {
				        high = value
				    } else {
				        low = value
				    }
				}

				func alarm{} (value f64) {}

				func logger{} (value f64) {}

				signal -> demux{threshold=100.0} -> {
				    high: alarm{},
				    low: logger{}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Root.Strata).To(HaveLen(3))
				Expect(inter.Root.Strata[0][0].Key()).To(Equal("on_signal_0"))
				Expect(inter.Root.Strata[1][0].Key()).To(Equal("demux_0"))
				keys := lo.Map(inter.Root.Strata[2], func(m ir.Member, _ int) string {
					return m.Key()
				})
				Expect(keys).To(ContainElements("alarm_0", "logger_0"))
			})
		})

		Context("Channel Sink Detection", func() {
			It("Should create write node for channel at end of flow", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "input_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10021},
					{Name: "output_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10022},
				}
				source := `
				func double{} (x f32) f32 {
				    return x * 2
				}

				input_chan -> double{} -> output_chan`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				inputNode := inter.Nodes[0]
				Expect(inputNode.Type).To(Equal("on"))
				Expect(inputNode.Channels.Read).To(HaveKey(uint32(10021)))
				Expect(inputNode.Outputs).To(HaveLen(1))

				outputNode := inter.Nodes[2]
				Expect(outputNode.Type).To(Equal("write"))
				Expect(outputNode.Channels.Write).To(HaveKey(uint32(10022)))
				Expect(outputNode.Inputs).To(HaveLen(2))
				Expect(outputNode.Inputs[0].Name).To(Equal("input"))
				Expect(outputNode.Outputs).To(HaveLen(1))
				Expect(outputNode.Outputs[0].Type).To(Equal(types.U8()))
			})

			It("Should handle channel-to-channel flow", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "chan1", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10031},
					{Name: "chan2", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10032},
				}
				source := `chan1 -> chan2`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))
				Expect(inter.Nodes[0].Type).To(Equal("on"))
				Expect(inter.Nodes[0].Channels.Read).To(HaveKey(uint32(10031)))
				Expect(inter.Nodes[1].Type).To(Equal("write"))
				Expect(inter.Nodes[1].Channels.Write).To(HaveKey(uint32(10032)))
			})

			It("Should handle channel sinks in routing tables", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "high_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10041},
					{Name: "low_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10045},
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10046},
				}
				source := `
				func demux{threshold f64} (value f64) (high f64, low f64) {
				    if (value > threshold) {
				        high = value
				    } else {
				        low = value
				    }
				}

				signal -> demux{threshold=100.0} -> {
				    high: high_chan,
				    low: low_chan
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))

				writeCount := countNodesByType(inter.Nodes, "write")
				Expect(writeCount).To(Equal(2))

				for _, node := range inter.Nodes {
					if node.Type == "write" {
						Expect(node.Inputs).To(HaveLen(2))
					}
				}
			})
		})

		Context("Single Node Flow Validation", func() {
			DescribeTable("Should reject single-node flows at parse time",
				func(source string) {
					_, diagnostics := text.Parse(text.Text{Raw: source})
					Expect(diagnostics).To(HaveOccurred())
					Expect(diagnostics.Ok()).To(BeFalse())
				},
				Entry("single function node", `
					func print{} () {}

					print{}`),
				Entry("single channel identifier", `sensor`),
			)
		})

		Context("Sequence Targeting", func() {
			It("Should connect one-shot edge to sequence's first stage entry node", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10051},
				}
				source := `
				sequence main {
				    stage run {}
				}

				trigger => main`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The channel read is the only node; the scope activation is
				// stamped directly on main's Scope (no synthesized entry
				// node, no edge).
				Expect(inter.Nodes).To(HaveLen(1))
				triggerNode := findNodeByKey(inter.Nodes, "on_trigger_0")
				Expect(triggerNode.Type).To(Equal("on"))

				// Sequence activation: trigger.output fires main's gated
				// scope.
				main := findTopLevelScope(inter, "main")
				Expect(main.Mode).To(Equal(ir.ScopeModeSequential))
				Expect(main.Liveness).To(Equal(ir.LivenessGated))
				Expect(main.Activation).ToNot(BeNil())
				Expect(main.Activation.Node).To(Equal("on_trigger_0"))
				Expect(main.Activation.Param).To(Equal(ir.DefaultOutputParam))

				// No dataflow edges and no write node for the sequence name.
				Expect(inter.Edges).To(BeEmpty())
				for _, node := range inter.Nodes {
					Expect(node.Key).ToNot(HavePrefix("write_main"))
				}
			})

			It("Should handle continuous flow to sequence", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10061},
				}
				source := `
				sequence main {
				    stage run {}
				}

				sensor -> main`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Continuous flow into a sequence activates the sequence
				// scope just like a conditional transition would; the
				// difference in source syntax doesn't change the IR shape.
				Expect(inter.Edges).To(BeEmpty())
				main := findTopLevelScope(inter, "main")
				Expect(main.Activation).ToNot(BeNil())
				Expect(main.Activation.Node).To(Equal("on_sensor_0"))
			})

			It("Should handle sequence with multiple stages - activates first stage", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10051},
				}
				source := `
				sequence main {
				    stage first {}
				    stage second {}
				}

				trigger => main`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Only the trigger read exists; stages are Scope members,
				// not IR nodes.
				Expect(inter.Nodes).To(HaveLen(1))
				Expect(inter.Edges).To(BeEmpty())

				main := findTopLevelScope(inter, "main")
				Expect(main.Steps).To(HaveLen(2))
				Expect(main.Steps[0].Key()).To(Equal("first"))
				Expect(main.Activation).ToNot(BeNil())
				Expect(main.Activation.Node).To(Equal("on_trigger_0"))
			})

			It("Should error when targeting empty sequence (no stages)", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10051},
				}
				source := `
				sequence empty {}

				trigger => empty`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse())
				Expect(diagnostics.String()).To(ContainSubstring("no steps"))
			})

			It("Should emit an inline stage case body as a gated synth nested in its enclosing scope with Activation bound", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30001},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30002},
					{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 30003},
				}
				source := `
				func gate{} (value u8) (yes u8, no u8) {
				    if (value > 0) {
				        yes = 1
				    } else {
				        no = 1
				    }
				}

				sequence main {
				    stage hold {
				        trigger -> gate{} -> {
				            yes: stage {
				                1 -> ch_a
				                "fired" -> log
				            }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				hold := findMember(main, "hold").Scope
				synth := findMember(*hold, "__inline_0").Scope
				Expect(synth.Liveness).To(Equal(ir.LivenessGated))
				Expect(synth.Activation).ToNot(BeNil())
				Expect(synth.Activation.Param).To(Equal("yes"))
			})

			It("Should emit an inline sequence case body as a sequential synth nested in its enclosing scope", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30011},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30012},
					{Name: "ch_b", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30013},
				}
				source := `
				func gate{} (value u8) (yes u8, no u8) {
				    if (value > 0) {
				        yes = 1
				    } else {
				        no = 1
				    }
				}

				sequence main {
				    stage hold {
				        trigger -> gate{} -> {
				            yes: sequence {
				                1 -> ch_a
				                2 -> ch_b
				            }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				hold := findMember(main, "hold").Scope
				synth := findMember(*hold, "__inline_0").Scope
				Expect(synth.Mode).To(Equal(ir.ScopeModeSequential))
				Expect(synth.Steps).To(HaveLen(2))
			})

			It("Should emit two distinct synths for two inline case bodies in the same table", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30021},
					{Name: "yes_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30022},
					{Name: "no_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30023},
				}
				source := `
				func gate{} (value u8) (yes u8, no u8) {
				    if (value > 0) {
				        yes = 1
				    } else {
				        no = 1
				    }
				}

				sequence main {
				    stage hold {
				        trigger -> gate{} -> {
				            yes: stage { 1 -> yes_ch },
				            no: stage { 1 -> no_ch }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				hold := findMember(main, "hold").Scope
				yesBody := findMember(*hold, "__inline_0").Scope
				noBody := findMember(*hold, "__inline_1").Scope
				params := []string{yesBody.Activation.Param, noBody.Activation.Param}
				Expect(params).To(ConsistOf("yes", "no"))
			})

			It("Should emit only one synth when one entry is inline and another is a chain", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30031},
					{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 30032},
				}
				source := `
				func gate{} (value u8) (yes u8, no u8) {
				    if (value > 0) {
				        yes = 1
				    } else {
				        no = 1
				    }
				}

				sequence main {
				    stage hold {
				        trigger -> gate{} -> {
				            yes: stage { "yes_inline" -> log },
				            no: "no_chain" -> log
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				hold := findMember(main, "hold").Scope
				synth := findMember(*hold, "__inline_0").Scope
				Expect(synth.Activation.Param).To(Equal("yes"))
				inlineCount := 0
				for _, stratum := range hold.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, ir.InlinePrefix) {
							inlineCount++
						}
					}
				}
				Expect(inlineCount).To(Equal(1))
			})

			It("Should not wrap a top-level stage with inline routing in a synthetic Sequential scope", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11020},
					{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 11021},
				}
				source := `
				stage main {
				    flag -> select{} -> {
				        true: stage { "true_branch" -> log_str },
				        false: stage { "false_branch" -> log_str }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				Expect(main.Mode).To(Equal(ir.ScopeModeParallel))
				Expect(main.Steps).To(BeEmpty())
				Expect(findMember(main, "__inline_0").Scope.Activation).ToNot(BeNil())
				Expect(findMember(main, "__inline_1").Scope.Activation).ToNot(BeNil())
			})

			It("Should not emit a duplicate transition when inline routing sits directly in a sequence body", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30061},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30062},
				}
				source := `
				func gate{} (value u8) (yes u8, no u8) {
				    if (value > 0) {
				        yes = 1
				    } else {
				        no = 1
				    }
				}

				sequence main {
				    trigger -> gate{} -> {
				        yes: stage { 1 -> ch_a }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				perHandle := map[ir.Handle]int{}
				for _, t := range main.Transitions {
					perHandle[t.On]++
				}
				for h, n := range perHandle {
					Expect(n).To(Equal(1),
						"handle %+v must carry exactly one transition, found %d", h, n)
				}
			})

			It("Should not auto-advance the enclosing sequence when inline routing fires", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30071},
					{Name: "inline_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30072},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30073},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> select{} -> {
				            true: stage { 1 -> inline_out }
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				for _, t := range main.Transitions {
					if strings.HasPrefix(t.On.Node, "select") {
						Fail("inline routing must not register a transition on main; " +
							"completion of select must not auto-advance to 'second'")
					}
				}
			})

			It("Should preserve activation on a named top-level stage with an inline case body", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11025},
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11026},
					{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 11027},
				}
				source := `
				trigger => main

				stage main {
				    flag -> select{} -> {
				        true: stage { "true_branch" -> log_str },
				        false: stage { "false_branch" -> log_str }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				Expect(main.Activation).ToNot(BeNil(),
					"named top-level stage must keep its trigger activation")
				Expect(main.Liveness).To(Equal(ir.LivenessGated))
			})

			It("Should emit an inline stage flow target as a gated synth root sibling with Activation bound", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31001},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31002},
					{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 31003},
				}
				source := `
				trigger -> stage {
				    1 -> ch_a
				    "fired" -> log
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				var synth *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, "__inline_") {
							synth = m.Scope
						}
					}
				}
				Expect(synth).ToNot(BeNil(),
					"inline flow target must surface as a root-level synth sibling")
				Expect(synth.Liveness).To(Equal(ir.LivenessGated),
					"synth inline must be gated by its activation handle")
				Expect(synth.Activation).ToNot(BeNil(),
					"synth inline must have an Activation handle bound")
				Expect(synth.Activation.Node).ToNot(BeEmpty(),
					"activation must fire on the upstream source's output node")
			})

			It("Should emit an inline sequence flow target as a sequential synth sibling", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31011},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31012},
					{Name: "ch_b", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31013},
				}
				source := `
				trigger -> sequence {
				    1 -> ch_a
				    2 -> ch_b
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				var synth *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, "__inline_") {
							synth = m.Scope
						}
					}
				}
				Expect(synth).ToNot(BeNil())
				Expect(synth.Mode).To(Equal(ir.ScopeModeSequential))
				Expect(synth.Steps).To(HaveLen(2),
					"inline sequence with 2 flow statements must expose 2 sequential steps")
			})

			It("Should emit two distinct synth siblings for two inline flow targets in the same scope", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31021},
					{Name: "other", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31022},
					{Name: "yes_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31023},
					{Name: "no_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31024},
				}
				source := `
				trigger -> stage { 1 -> yes_ch }
				other -> stage { 1 -> no_ch }`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				var synths []*ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, "__inline_") {
							synths = append(synths, m.Scope)
						}
					}
				}
				Expect(synths).To(HaveLen(2),
					"each inline flow target must produce a distinct synth sibling")
				Expect(synths[0].Key).ToNot(Equal(synths[1].Key))
				Expect(synths[0].Activation).ToNot(BeNil())
				Expect(synths[1].Activation).ToNot(BeNil())
				Expect(synths[0].Activation.Node).ToNot(Equal(synths[1].Activation.Node),
					"each synth must be gated by its own source's output")
			})

			It("Should emit only one synth sibling when one flow statement is inline and another is a plain chain", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31031},
					{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31032},
					{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 31033},
				}
				source := `
				trigger -> stage { "yes_inline" -> log }
				trigger -> sink`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				var synths []*ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, "__inline_") {
							synths = append(synths, m.Scope)
						}
					}
				}
				Expect(synths).To(HaveLen(1),
					"only the inline flow target must produce a synth sibling")
				Expect(synths[0].Activation).ToNot(BeNil())
			})

			It("Should not wrap a top-level stage with inline flow targets in a synthetic Sequential scope", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31041},
					{Name: "other", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31042},
					{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 31043},
				}
				source := `
				stage main {
				    flag -> stage { "true_branch" -> log_str }
				    other -> stage { "false_branch" -> log_str }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				Expect(main.Mode).To(Equal(ir.ScopeModeParallel))
				Expect(main.Steps).To(BeEmpty())
				Expect(findMember(main, "__inline_0").Scope.Activation).ToNot(BeNil())
				Expect(findMember(main, "__inline_1").Scope.Activation).ToNot(BeNil())
			})

			It("Should not emit a duplicate transition when an inline flow target sits directly in a sequence body", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31051},
					{Name: "ch_a", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31052},
				}
				source := `
				sequence main {
				    trigger -> stage { 1 -> ch_a }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				perHandle := map[ir.Handle]int{}
				for _, t := range main.Transitions {
					perHandle[t.On]++
				}
				for h, n := range perHandle {
					Expect(n).To(Equal(1),
						"handle %+v must carry exactly one transition, found %d", h, n)
				}
			})

			It("Should not auto-advance the enclosing sequence when an inline flow target fires", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31061},
					{Name: "inline_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31062},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31063},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> stage { 1 -> inline_out }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				Expect(main.Transitions).To(BeEmpty(),
					"inline flow target completion must not register a transition on main; "+
						"'second' must not auto-advance")
			})

			It("Should preserve activation on a named top-level stage with an inline flow target", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31071},
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31072},
					{Name: "log_str", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 31073},
				}
				source := `
				trigger => main

				stage main {
				    flag -> stage { "true_branch" -> log_str }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				main := findTopLevelScope(inter, "main")
				Expect(main.Activation).ToNot(BeNil(),
					"named top-level stage must keep its trigger activation")
				Expect(main.Liveness).To(Equal(ir.LivenessGated))
			})

			It("Should emit a distinct gated synth for each body when inline flow targets are nested", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31081},
					{Name: "inner_ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31082},
				}
				source := `
				trigger -> stage {
				    1 -> stage { 1 -> inner_ch }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				outer := findMember(inter.Root, "__inline_0").Scope
				inner := findMember(*outer, "__inline_1").Scope
				for _, s := range []*ir.Scope{outer, inner} {
					Expect(s.Liveness).To(Equal(ir.LivenessGated))
					Expect(s.Activation).ToNot(BeNil())
				}
			})

			It("Should reject `=> next` inside an inline routing case body", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30081},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30082},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> select{} -> {
				            true: stage {
				                flag > 0 => next
				            }
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse(),
					"`=> next` from an inline routing case body must be rejected")
				Expect(diagnostics.String()).To(ContainSubstring(
					"'next' is not valid inside an inline routing case body"))
			})

			It("Should reject `=> next` that escapes an inline sequence routing case body", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30091},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 30092},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> select{} -> {
				            true: sequence {
				                flag > 0 => next
				            }
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse(),
					"`=> next` escaping an inline sequence body must be rejected")
				Expect(diagnostics.String()).To(ContainSubstring(
					"'next' is not valid inside an inline routing case body"))
			})

			It("Should reject `=> next` inside an inline stage flow target", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31081},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31082},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> stage {
				            flag > 0 => next
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse(),
					"`=> next` from an inline stage flow target must be rejected")
				Expect(diagnostics.String()).To(ContainSubstring(
					"'next' is not valid inside an inline routing case body"))
			})

			It("Should reject `=> next` that escapes an inline sequence flow target", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31091},
					{Name: "second_out", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 31092},
				}
				source := `
				sequence main {
				    stage first {
				        flag -> sequence {
				            flag > 0 => next
				        }
				    }
				    stage second {
				        1 -> second_out
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeFalse(),
					"`=> next` escaping an inline sequence flow target must be rejected")
				Expect(diagnostics.String()).To(ContainSubstring(
					"'next' is not valid inside an inline routing case body"))
			})

			It("Should handle sequence in routing table as sink", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "high_chan", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10071},
					{Name: "signal", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 10072},
				}
				source := `
				sequence alarm {
				    stage active {}
				}

				func demux{threshold f64} (value f64) (high u8, low f64) {
				    if (value > threshold) {
				        high = 1
				    } else {
				        low = value
				    }
				}

				signal -> demux{threshold=100.0} -> {
				    high: alarm,
				    low: high_chan
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// on node for signal + demux node + write node for the `low`
				// branch. The `high` branch is a scope reference that becomes
				// an activation.
				Expect(inter.Nodes).To(HaveLen(3))

				demuxNode := findNodeByType(inter.Nodes, "demux")
				writeNode := findNodeByType(inter.Nodes, "write")
				Expect(writeNode.Channels.Write).To(HaveKey(uint32(10071)))

				// The signal->demux edge and the write edge exist; the `high`
				// branch lands on alarm's scope activation.
				Expect(inter.Edges).To(HaveLen(2))
				alarm := findTopLevelScope(inter, "alarm")
				Expect(alarm.Activation).ToNot(BeNil())
				Expect(alarm.Activation.Node).To(Equal(demuxNode.Key))
				Expect(alarm.Activation.Param).To(Equal("high"))
			})

		})

		Context("Direct Stage Targeting", func() {
			It("Should emit a member-key transition when targeting a sibling stage", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10081},
				}
				source := `
				sequence main {
				    stage first {
				        input > 10 => second
				    }
				    stage second {}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The `=> second` edge becomes a Transition on main whose
				// on-handle fires from the comparison node.
				main := findTopLevelScope(inter, "main")
				Expect(main.Transitions).To(HaveLen(1))
				t := main.Transitions[0]
				Expect(t.TargetKey).ToNot(BeNil())
				Expect(*t.TargetKey).To(Equal("second"))
				Expect(t.On.Node).To(HavePrefix("expression_"))
			})
		})

		Context("Cross-scope Transitions", func() {
			// These tests assert the IR shape produced when `=> X` resolves
			// to a frame further up the shell stack than the innermost
			// enclosing sequence. The resolver must (1) place the transition
			// on the correct owning frame, (2) suppress the redundant
			// auto-wire that would otherwise clear markedFlags before the
			// outer frame's evaluator sees it, and (3) preserve the
			// root-level activation path for top-level targets.

			It("Places => X on the enclosing sequence's frame when X is two levels up through a stage and a nested sequence", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20001},
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20002},
				}
				source := `
				sequence main {
				    stage first {
				        sequence {
				            trigger => second
				        }
				    }
				    stage second {
				        1 -> ch
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The transition must live on `main` (the frame whose
				// memberKeys contain `second`), not on the anonymous nested
				// sequence inside `first`.
				main := findTopLevelScope(inter, "main")
				Expect(main.Transitions).To(HaveLen(1),
					"=> second must be placed on main's frame")
				t := main.Transitions[0]
				Expect(t.TargetKey).ToNot(BeNil())
				Expect(*t.TargetKey).To(Equal("second"))
				Expect(t.On.Node).To(HavePrefix("on_trigger"))

				// The nested sequence inside first must NOT carry an exit
				// transition on the same handle — otherwise its evaluator
				// would clear markedFlags before main's evaluator sees it.
				first := findMember(main, "first").Scope
				Expect(first).ToNot(BeNil())
				nestedSeqMember := first.Strata[0][0]
				Expect(nestedSeqMember.Scope).ToNot(BeNil())
				Expect(nestedSeqMember.Scope.Transitions).To(BeEmpty(),
					"the inner nested sequence must not auto-wire an exit transition when the flow ends in an explicit cross-level => target")
			})

			It("Prefers the innermost sequence's member when the name exists at multiple levels", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20003},
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20004},
				}
				source := `
				sequence outer {
				    sequence inner {
				        stage a {
				            trigger => target
				        }
				        stage target {
				            1 -> ch
				        }
				    }
				    sequence target {
				        1 -> ch
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// The transition must land on `inner` (innermost sibling
				// that owns `target`), not on `outer`.
				outer := findTopLevelScope(inter, "outer")
				Expect(outer.Transitions).To(BeEmpty(),
					"outer must not carry the transition — it is shadowed by inner's target")

				inner := findMember(outer, "inner").Scope
				Expect(inner).ToNot(BeNil())
				Expect(inner.Transitions).To(HaveLen(1),
					"inner must carry the transition because its target shadows outer's")
				t := inner.Transitions[0]
				Expect(t.TargetKey).ToNot(BeNil())
				Expect(*t.TargetKey).To(Equal("target"))
			})

			It("Emits exactly one transition per flow step that ends in an explicit => X (no auto-wire duplicate)", func(ctx SpecContext) {
				// Regression guard: before the auto-wire-suppression fix,
				// analyzeSequence would emit a terminal exit transition on
				// the innermost frame in addition to the explicit cross-
				// level transition. That clobbered the outer transition's
				// markedFlags. The IR should have a single transition for
				// this flow step.
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20005},
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20006},
				}
				source := `
				sequence main {
				    stage first {
				        sequence {
				            trigger => second
				        }
				    }
				    stage second {
				        1 -> ch
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				// Count transitions across main's frame and the inner anon
				// sequence's frame — the total must be 1.
				main := findTopLevelScope(inter, "main")
				first := findMember(main, "first").Scope
				inner := first.Strata[0][0].Scope
				total := len(main.Transitions) + len(inner.Transitions)
				Expect(total).To(Equal(1),
					"a single explicit => X must produce exactly one transition across all frames")
			})

			It("Preserves root-level activation for => root_sibling from inside a top-level sequence", func(ctx SpecContext) {
				// Regression guard for the activation path. When the target
				// is not in any enclosing frame's memberKeys but is a root-
				// level scope, the resolver must register an activation
				// (not a transition) and the stamping loop must set
				// Activation on that scope.
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20007},
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 20008},
				}
				source := `
				sequence main {
				    trigger => other
				}
				sequence other {
				    1 -> ch
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				other := findTopLevelScope(inter, "other")
				Expect(other.Activation).ToNot(BeNil(),
					"=> other from inside main must stamp an Activation handle on other")
				Expect(other.Activation.Node).To(HavePrefix("on_trigger"))
			})
		})

		Context("Top-level anonymous scopes", func() {
			It("Should compile an anonymous top-level stage and produce a root member with an auto-generated key", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11001},
				}
				source := `
				stage {
				    1 -> ch
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Root.Strata).To(HaveLen(1))
				var scopeMembers []ir.Member
				for _, m := range inter.Root.Strata[0] {
					if m.Scope != nil {
						scopeMembers = append(scopeMembers, m)
					}
				}
				Expect(scopeMembers).To(HaveLen(1))
				s := scopeMembers[0].Scope
				Expect(s.Key).To(HavePrefix("stage_"))
				Expect(s.Liveness).To(Equal(ir.LivenessAlways))
				Expect(s.Activation).To(BeNil())
			})

			It("Should compile an anonymous top-level sequence and produce a root member with an auto-generated key", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11002},
				}
				source := `
				sequence {
				    1 -> ch
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Root.Strata).To(HaveLen(1))
				var scopeMembers []ir.Member
				for _, m := range inter.Root.Strata[0] {
					if m.Scope != nil {
						scopeMembers = append(scopeMembers, m)
					}
				}
				Expect(scopeMembers).To(HaveLen(1))
				s := scopeMembers[0].Scope
				Expect(s.Key).To(HavePrefix("seq_"))
				Expect(s.Liveness).To(Equal(ir.LivenessAlways))
				Expect(s.Activation).To(BeNil())
			})

			It("Should compile a named top-level sequence as LivenessGated with no activation when nothing targets it", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11010},
				}
				source := `
				sequence main {
				    stage s {
				        1 -> ch
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Root.Strata).To(HaveLen(1))
				var scopeMembers []ir.Member
				for _, m := range inter.Root.Strata[0] {
					if m.Scope != nil {
						scopeMembers = append(scopeMembers, m)
					}
				}
				Expect(scopeMembers).To(HaveLen(1))
				s := scopeMembers[0].Scope
				Expect(s.Key).To(Equal("main"))
				Expect(s.Liveness).To(Equal(ir.LivenessGated))
				Expect(s.Activation).To(BeNil())
			})

			It("Should compile a named top-level stage as LivenessGated with no activation when nothing targets it", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11011},
				}
				source := `
				stage main {
				    1 -> ch
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				Expect(inter.Root.Strata).To(HaveLen(1))
				var scopeMembers []ir.Member
				for _, m := range inter.Root.Strata[0] {
					if m.Scope != nil {
						scopeMembers = append(scopeMembers, m)
					}
				}
				Expect(scopeMembers).To(HaveLen(1))
				s := scopeMembers[0].Scope
				Expect(s.Key).To(Equal("main"))
				Expect(s.Liveness).To(Equal(ir.LivenessGated))
				Expect(s.Activation).To(BeNil())
			})

			It("Should compile a named top-level sequence as LivenessGated with an activation handle when `trigger => name` targets it", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11012},
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11013},
				}
				source := `
				trigger => main

				sequence main {
				    stage s {
				        1 -> ch
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				var main *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && m.Scope.Key == "main" {
							main = m.Scope
						}
					}
				}
				Expect(main).ToNot(BeNil())
				Expect(main.Liveness).To(Equal(ir.LivenessGated))
				Expect(main.Activation).ToNot(BeNil())
			})

			It("Should compile named nested scopes as LivenessGated at every depth", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11014},
				}
				source := `
				sequence outer {
				    stage s {
				        sequence inner {
				            stage t {
				                1 -> ch
				            }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				var outer *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && m.Scope.Key == "outer" {
							outer = m.Scope
						}
					}
				}
				Expect(outer).ToNot(BeNil())
				Expect(outer.Liveness).To(Equal(ir.LivenessGated))
				Expect(outer.Steps).To(HaveLen(1))
				stageS := outer.Steps[0].Scope
				Expect(stageS).ToNot(BeNil())
				Expect(stageS.Key).To(Equal("s"))
				Expect(stageS.Liveness).To(Equal(ir.LivenessGated))
				var inner *ir.Scope
				for _, stratum := range stageS.Strata {
					for _, mem := range stratum {
						if mem.Scope != nil && mem.Scope.Key == "inner" {
							inner = mem.Scope
						}
					}
				}
				Expect(inner).ToNot(BeNil())
				Expect(inner.Liveness).To(Equal(ir.LivenessGated))
				Expect(inner.Steps).To(HaveLen(1))
				stageT := inner.Steps[0].Scope
				Expect(stageT).ToNot(BeNil())
				Expect(stageT.Key).To(Equal("t"))
				Expect(stageT.Liveness).To(Equal(ir.LivenessGated))
			})

			It("Should compile anonymous nested scopes as LivenessAlways at every depth", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11015},
				}
				source := `
				sequence {
				    stage {
				        sequence {
				            stage {
				                1 -> ch
				            }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				var outer *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && strings.HasPrefix(m.Scope.Key, "seq_") {
							outer = m.Scope
						}
					}
				}
				Expect(outer).ToNot(BeNil())
				Expect(outer.Liveness).To(Equal(ir.LivenessAlways))
				Expect(outer.Steps).To(HaveLen(1))
				stageA := outer.Steps[0].Scope
				Expect(stageA).ToNot(BeNil())
				Expect(stageA.Liveness).To(Equal(ir.LivenessAlways))
				var innerSeq *ir.Scope
				for _, stratum := range stageA.Strata {
					for _, mem := range stratum {
						if mem.Scope != nil {
							innerSeq = mem.Scope
						}
					}
				}
				Expect(innerSeq).ToNot(BeNil())
				Expect(innerSeq.Liveness).To(Equal(ir.LivenessAlways))
				Expect(innerSeq.Steps).To(HaveLen(1))
				stageB := innerSeq.Steps[0].Scope
				Expect(stageB).ToNot(BeNil())
				Expect(stageB.Liveness).To(Equal(ir.LivenessAlways))
			})

			It("Should compile mixed named/anonymous nesting per the uniform rule", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "ch", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 11016},
				}
				source := `
				sequence main {
				    stage first {
				        sequence {
				            stage {
				                1 -> ch
				            }
				        }
				    }
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
				var main *ir.Scope
				for _, stratum := range inter.Root.Strata {
					for _, m := range stratum {
						if m.Scope != nil && m.Scope.Key == "main" {
							main = m.Scope
						}
					}
				}
				Expect(main).ToNot(BeNil())
				Expect(main.Liveness).To(Equal(ir.LivenessGated))
				Expect(main.Steps).To(HaveLen(1))
				first := main.Steps[0].Scope
				Expect(first).ToNot(BeNil())
				Expect(first.Key).To(Equal("first"))
				Expect(first.Liveness).To(Equal(ir.LivenessGated))
				var anonSeq *ir.Scope
				for _, stratum := range first.Strata {
					for _, mem := range stratum {
						if mem.Scope != nil && strings.HasPrefix(mem.Scope.Key, "seq_") {
							anonSeq = mem.Scope
						}
					}
				}
				Expect(anonSeq).ToNot(BeNil())
				Expect(anonSeq.Liveness).To(Equal(ir.LivenessAlways))
				Expect(anonSeq.Steps).To(HaveLen(1))
				anonStage := anonSeq.Steps[0].Scope
				Expect(anonStage).ToNot(BeNil())
				Expect(anonStage.Liveness).To(Equal(ir.LivenessAlways))
			})
		})

		Context("next keyword", func() {
			It("Should emit a member-key transition targeting the following stage", func(ctx SpecContext) {
				source := `
				sequence main {
				    stage first {
				        1 -> output
				        input > 10 => next
				    }
				    stage second {
				        0 -> output
				    }
				}`
				resolver := []symbol.Symbol{
					{Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10091},
					{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10092},
				}
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diag := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diag.Ok()).To(BeTrue(), diag.String())

				main := findTopLevelScope(inter, "main")
				nextT, ok := lo.Find(main.Transitions, func(t ir.Transition) bool {
					return t.TargetKey != nil && *t.TargetKey == "second"
				})
				Expect(ok).To(BeTrue(), "expected a transition targeting 'second'")
				Expect(nextT.On.Node).To(HavePrefix("expression_"))
			})

			DescribeTable("next keyword error cases",
				func(ctx SpecContext, source string, chans []symbol.Symbol, expectedError string) {
					parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
					root := symbol.NewRoot(nil, stl.NewSymbols())
					for i := range chans {
						s := chans[i]
						root.Parent.AddChild(&s)
					}
					_, diag := text.Analyze(ctx, parsedText, root)
					Expect(diag).To(HaveOccurred())
					Expect(diag.Ok()).To(BeFalse())
					Expect(diag.String()).To(ContainSubstring(expectedError))
				},
				Entry("next in last stage",
					`
					sequence main {
					    stage only {
					        input > 10 => next
					    }
					}`,
					[]symbol.Symbol{
						{Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10081},
					},
					"no next stage",
				),
				Entry("next outside sequence",
					`input > 10 => next`,
					[]symbol.Symbol{
						{Name: "input", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10081},
					},
					"outside of a sequence",
				),
				Entry("next inside inline routing stage body",
					`
					sequence main {
					    stage first {
					        flag -> select{} -> {
					            true: stage { 1 => next }
					        }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
				Entry("next on last step of inline routing sequence body",
					`
					sequence main {
					    stage first {
					        flag -> select{} -> {
					            true: sequence {
					                1 -> sink,
					                2 => next
					            }
					        }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
				Entry("next inside doubly-nested inline routing stage body",
					`
					sequence main {
					    stage first {
					        flag -> select{} -> {
					            true: stage {
					                flag -> select{} -> {
					                    true: stage { 1 => next }
					                }
					            }
					        }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
				Entry("next inside inline stage flow target",
					`
					sequence main {
					    stage first {
					        flag -> stage { 1 => next }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
				Entry("next on last step of inline sequence flow target",
					`
					sequence main {
					    stage first {
					        flag -> sequence {
					            1 -> sink,
					            2 => next
					        }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
				Entry("next inside doubly-nested inline stage flow target",
					`
					sequence main {
					    stage first {
					        flag -> stage {
					            flag -> stage { 1 => next }
					        }
					    }
					    stage second { 1 -> sink }
					}`,
					[]symbol.Symbol{
						{Name: "flag", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10082},
						{Name: "sink", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10083},
					},
					"not valid inside an inline routing case body",
				),
			)
		})

		Context("Implicit Expression Triggers", func() {
			It("Should inject implicit trigger for expression as first flow node", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10142},
				}
				source := `
				func alarm{} (value u8) {}

				sensor > 20 => alarm{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				triggerNode := findNodeByKey(inter.Nodes, "on_sensor_0")
				Expect(triggerNode.Type).To(Equal("on"))
				Expect(triggerNode.Channels.Read).To(HaveKey(uint32(10142)))

				exprNode := inter.Nodes[1]
				Expect(exprNode.Type).To(HavePrefix("expression_"))
				Expect(exprNode.Channels.Read).To(HaveKey(uint32(10142)))

				Expect(inter.Edges).To(HaveLen(2))

				edge0 := inter.Edges[0]
				Expect(edge0.Source.Node).To(Equal("on_sensor_0"))
				Expect(edge0.Target.Node).To(Equal(exprNode.Key))
				Expect(edge0.Kind).To(Equal(ir.EdgeKindContinuous))

				// Second edge: expression -> alarm (Conditional from =>)
				edge1 := inter.Edges[1]
				Expect(edge1.Source.Node).To(Equal(exprNode.Key))
				Expect(edge1.Target.Node).To(Equal("alarm_0"))
				Expect(edge1.Kind).To(Equal(ir.EdgeKindConditional))
			})

			It("Should not suppress a trigger when the flow writes a different channel", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "a", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10191},
					{Name: "b", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10192},
				}
				parsedText := MustSucceed(text.Parse(text.Text{Raw: `a + 1 => b`}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(countNodesByType(inter.Nodes, "constant")).To(Equal(0),
					"a non-self-writing flow keeps its channel trigger and needs no pulse")
				Expect(countNodesByType(inter.Nodes, "on")).To(Equal(1))

				triggerNode := findNodeByType(inter.Nodes, "on")
				Expect(triggerNode.Channels.Read).To(HaveKey(uint32(10191)))
			})

			It("Should inject multiple triggers for multi-channel expression", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "temp", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10151},
					{Name: "pressure", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10152},
				}
				source := `
				func alarm{} (value u8) {}

				temp + pressure > 100 => alarm{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(4))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(2))

				var exprNode ir.Node
				for _, n := range inter.Nodes {
					if n.Type != "on" && n.Type != "alarm" {
						exprNode = n
						break
					}
				}
				Expect(exprNode.Channels.Read).To(HaveLen(2))
				Expect(exprNode.Channels.Read).To(HaveKey(uint32(10151)))
				Expect(exprNode.Channels.Read).To(HaveKey(uint32(10152)))

				Expect(inter.Edges).To(HaveLen(3))

				exprEdgeCount := 0
				for _, edge := range inter.Edges {
					if edge.Target.Node == exprNode.Key {
						exprEdgeCount++
						Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
					}
				}
				Expect(exprEdgeCount).To(Equal(2))
			})

			It("Should not inject trigger for constant expressions", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "output", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10161},
				}
				source := `1 + 2 -> output`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(0))
			})

			It("Should not inject trigger when expression is not first node", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10142},
				}
				source := `
				func alarm{} (value u8) {}

				sensor -> sensor > 20 => alarm{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(3))

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(1))

				Expect(inter.Edges).To(HaveLen(2))
			})

			It("Should inject trigger for expression in sequence stage", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 10142},
				}
				source := `
				sequence main {
				    stage monitoring {
				        sensor > 100 => next
				    }
				    stage alarm {}
				}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				triggerCount := countNodesByType(inter.Nodes, "on")
				Expect(triggerCount).To(Equal(1))

				triggerNode := findNodeByType(inter.Nodes, "on")
				Expect(triggerNode.Channels.Read).To(HaveKey(uint32(10142)))
			})
		})

		Context("Interval One-Shot Edge Generation", func() {
			It("Should generate one-shot edge for interval triggering function", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "interval",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Inputs:  types.Params{{Name: "period", Type: types.TimeSpan()}},
							Outputs: types.Params{{Name: "output", Type: types.U8()}},
						}),
					},
				}
				source := `
				func press{} () {}

				interval{period=50ms} => press{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Nodes).To(HaveLen(2))

				intervalNode := findNodeByType(inter.Nodes, "interval")
				Expect(intervalNode.Inputs).To(HaveLen(1))
				Expect(intervalNode.Inputs[0].Name).To(Equal("period"))

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Source.Node).To(Equal(intervalNode.Key))
				Expect(edge.Source.Param).To(Equal("output"))
				Expect(edge.Target.Node).To(Equal("press_0"))
				Expect(edge.Kind).To(Equal(ir.EdgeKindConditional))
			})

			It("Should generate continuous edge for interval with -> operator", func(ctx SpecContext) {
				resolver := []symbol.Symbol{
					{
						Name: "interval",
						Kind: symbol.KindFunction,
						Type: types.Function(types.FunctionProperties{
							Inputs:  types.Params{{Name: "period", Type: types.TimeSpan()}},
							Outputs: types.Params{{Name: "output", Type: types.U8()}},
						}),
					},
				}
				source := `
				func handler{} () {}

				interval{period=50ms} -> handler{}`
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
				Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

				Expect(inter.Edges).To(HaveLen(1))
				edge := inter.Edges[0]
				Expect(edge.Kind).To(Equal(ir.EdgeKindContinuous))
			})
		})
	})

	Describe("Synthesized Format-String Functions", func() {
		It("Registers a fmt$ function for a flow-form format string with a single placeholder", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := `sensor -> f"v={sensor}" -> log`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			f := synth[0]
			Expect(f.Body.Raw).To(Equal("v={sensor}"))
			Expect(f.Inputs).To(BeEmpty())
			Expect(f.Inputs).To(BeEmpty())
			Expect(f.Outputs).To(HaveLen(1))
			Expect(f.Outputs[0].Type).To(Equal(types.String()))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
			Expect(f.Channels.Write).To(BeEmpty())

			synthNode := findNodeByType(inter.Nodes, f.Key)
			Expect(synthNode.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
		})

		It("Registers a fmt$ function whose Channels.Read covers every placeholder channel", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "t", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 102},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := `sensor -> f"v={sensor} t={t}" -> log`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			f := synth[0]
			Expect(f.Body.Raw).To(Equal("v={sensor} t={t}"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(102), "t"))
		})

		It("Records the aliased channel key for a placeholder that reads a channel read/write", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := "cpu := sensor\nsensor -> " + `f"v={cpu}"` + " -> log"
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			Expect(synth[0].Channels.Read).To(HaveKey(uint32(100)))
		})

		It("Does not synthesize a fmt$ function for a literal format string with no placeholders", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := `trig -> f"static" -> log`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			for _, f := range inter.Functions {
				Expect(strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)).To(BeFalse(),
					"unexpected fmt$ synthetic %q for placeholder-free literal", f.Key)
			}
		})

		It("Registers a fmt$ function for an rf-prefixed multi-line format string preserving backslashes across newlines", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "t", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 102},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := "sensor -> rf`path\\to: {sensor}\nt={t}` -> log"
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			f := synth[0]
			Expect(f.Body.Raw).To(Equal("path\\to: {sensor}\nt={t}"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(102), "t"))
		})

		It("Registers a fmt$ function for an rf-prefixed format string preserving backslashes", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := `sensor -> rf"path\to: {sensor}" -> log`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			f := synth[0]
			Expect(f.Body.Raw).To(Equal(`path\to: {sensor}`))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
		})

		It("Registers a fmt$ function for a multi-line format string with placeholders across newlines", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "t", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 102},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := "sensor -> f`v={sensor}\nt={t}` -> log"
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			synth := lo.Filter(inter.Functions, func(f ir.Function, _ int) bool {
				return strings.HasPrefix(f.Key, compiler.FmtStrSyntheticPrefix)
			})
			Expect(synth).To(HaveLen(1))
			f := synth[0]
			Expect(f.Body.Raw).To(Equal("v={sensor}\nt={t}"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(100), "sensor"))
			Expect(f.Channels.Read).To(HaveKeyWithValue(uint32(102), "t"))
		})

		It("Surfaces analyzer diagnostics for an invalid format spec at this layer", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.F32()), ID: 100},
				{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String()), ID: 101},
			}
			source := `sensor -> f"v={sensor:d}" -> log`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeFalse(),
				"expected an analyzer diagnostic for :d on a float channel")
			Expect(diagnostics.String()).To(ContainSubstring("invalid format spec"))
		})
	})

	Describe("Unit Dimensional Analysis", func() {
		DescribeTable("dimension compatibility",
			func(ctx SpecContext, source string, expectOk bool, expectedErrorContains string) {
				parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
				_, diag := text.Analyze(ctx, parsedText, NewRoot(nil))
				if expectOk {
					Expect(diag.Ok()).To(BeTrue(), diag.String())
				} else {
					Expect(diag.Ok()).To(BeFalse())
					Expect(diag.String()).To(ContainSubstring(expectedErrorContains))
				}
			},
			Entry("error when adding incompatible dimensions",
				`func bad() f64 {
    return 5psi + 3s
}`,
				false, "incompatible dimensions:",
			),
			Entry("allow adding same dimensions",
				`func good() f64 {
    return 100psi + 50psi
}`,
				true, "",
			),
			Entry("allow multiplying different dimensions",
				`func velocity() f64 {
    return 100m / 10s
}`,
				true, "",
			),
		)
	})

	Describe("Single Invocations in Stages", func() {
		It("Should compile standalone function invocation to IR node", func(ctx SpecContext) {
			source := `
			func setup() {}

			sequence main {
			    stage start {
			        setup{}
			    }
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			setupNode := findNodeByType(inter.Nodes, "setup")
			Expect(setupNode.Inputs).To(BeEmpty())

			main := findTopLevelScope(inter, "main")
			start := findMember(main, "start")
			Expect(start.Scope).ToNot(BeNil())
			Expect(scopeNodeRefs(*start.Scope)).To(ContainElement(setupNode.Key))
		})

		It("Should compile standalone expression to IR node", func(ctx SpecContext) {
			source := `
			sequence main {
			    stage start {
			        1 + 2
			    }
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			exprNodes := lo.Filter(inter.Nodes, func(n ir.Node, _ int) bool {
				return strings.HasPrefix(n.Type, "expression_")
			})
			Expect(exprNodes).To(HaveLen(1))
			exprNode := exprNodes[0]
			Expect(exprNode.Outputs).To(HaveLen(1))
			Expect(exprNode.Outputs[0].Type.Kind).To(Equal(types.KindI64))

			main := findTopLevelScope(inter, "main")
			start := findMember(main, "start")
			Expect(start.Scope).ToNot(BeNil())
			Expect(scopeNodeRefs(*start.Scope)).To(ContainElement(exprNode.Key))
		})

		It("Should reject void functions mid-chain in flow statements", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "counter", Kind: symbol.KindChannel, Type: types.Chan(types.U32()), ID: 10201},
				{Name: "trigger", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10202},
			}
			source := `
			func increment{ch chan u32} () {
			    ch = ch + 1
			}

			sequence main {
			    stage first {
			        trigger => increment{ch=counter} => next
			    }
			    stage second {}
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.String()).To(ContainSubstring("has no output"))
		})

		It("Should place a single-invocation step's node in its stage's first phase", func(ctx SpecContext) {
			source := `
			func initialize() u8 {
			    return 1
			}

			sequence main {
			    stage start {
			        initialize{}
			    }
			}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			initNode := findNodeByType(inter.Nodes, "initialize")
			Expect(initNode.Inputs).To(BeEmpty())

			main := findTopLevelScope(inter, "main")
			start := findMember(main, "start")
			Expect(start.Scope).ToNot(BeNil())
			Expect(start.Scope.Mode).To(Equal(ir.ScopeModeParallel))
			Expect(start.Scope.Strata).ToNot(BeEmpty())
			Expect(scopeNodeRefs(*start.Scope)).To(ContainElement(initNode.Key))
		})
	})

	Describe("Authority Analysis", func() {
		It("Should include authority input in IR with simple form", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "valve", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
			}
			source := `
			authority 200

			func a{} () f64 {
			    return 0.0
			}
			a{} -> valve`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Authorities.Default).ToNot(BeNil())
			Expect(*inter.Authorities.Default).To(Equal(uint8(200)))
		})

		It("Should include per-channel authority overrides", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "valve", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 100},
				{Name: "vent", Kind: symbol.KindChannel, Type: types.Chan(types.F64()), ID: 200},
			}
			source := `
			authority (
			    200
			    valve 100
			    vent 150
			)

			func a{} () f64 {
			    return 0.0
			}
			a{} -> valve`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(inter.Authorities.Default).ToNot(BeNil())
			Expect(*inter.Authorities.Default).To(Equal(uint8(200)))
			Expect(inter.Authorities.Channels).To(HaveLen(2))
			Expect(inter.Authorities.Channels[100]).To(Equal(uint8(100)))
			Expect(inter.Authorities.Channels[200]).To(Equal(uint8(150)))
		})

		It("Should report error for authority after function", func(ctx SpecContext) {
			source := `
			func a{} () {}
			authority 200
			func b{} () {}
			a{} -> b{}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.String()).To(ContainSubstring("before"))
		})
	})

	Describe("Compile", func() {
		It("Should compile a simple arc program to WebAssembly", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "in", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 100},
			}
			source := `
			func adder{} (a i64) i64 {
			    return a
			}

			func print{} () {}

			in -> adder{} -> print{}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})

		It("Should compile a function-only program with no flow statement", func(ctx SpecContext) {
			// A function declaration compiles and exports to WASM on its own,
			// without being referenced by a flow node. This includes multi-input
			// functions, which cannot be expressed as a runnable flow node at all
			// (a flow node receives a single upstream value). Callers that invoke
			// the exported function directly (e.g. the C++ runtime test harness)
			// rely on this instead of scaffolding an unconnected flow node.
			source := `func pow_ff(base f64, exp f64) f64 { return base ^ exp }`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})

		It("Should compile function with channel input param assigned to intermediate variable", func(ctx SpecContext) {
			// This is the exact user pattern that was failing:
			// sp := set_point (where set_point is a chan f32 input param)
			resolver := []symbol.Symbol{
				{
					Name: "virt_1",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   10025,
				},
				{
					Name: "alarm_out",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   10026,
				},
			}
			source := `
			func tolerance_alarm{
			    tolerance_upper f32,
			    tolerance_lower f32,
			    set_point chan f32,
			    samples i64,
			} (value f32) u8 {
			    count i64 $= 0
			    sp := set_point

			    if value >= (sp + tolerance_upper) {
			        count = count + 1
			    } else if value <= (sp - tolerance_lower) {
			        count = count + 1
			    } else {
			        count = 0
			    }

			    if count >= samples {
			        return 1
			    }
			    return 0
			}

			virt_1 -> tolerance_alarm{
			    tolerance_upper = 200.0,
			    tolerance_lower = 0.0,
			    set_point = virt_1,
			    samples = 10
			} -> alarm_out`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(ir.Nodes).To(HaveLen(3))
			Expect(ir.Nodes[1].Channels.Read).To(HaveLen(1))
			Expect(ir.Nodes[1].Channels.Read).To(HaveKey(uint32(10025)))

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})

		It("Should compile function with channel input param assigned to intermediate variable and written to", func(ctx SpecContext) {
			// Test that writing to an intermediate variable correctly tracks the channel
			// out := output (input param with channel type)
			// out = value * 2.0 (write to channel through intermediate variable)
			resolver := []symbol.Symbol{
				{
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   10100,
				},
				{
					Name: "write_target",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   10200,
				},
				{
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   10300,
				},
			}
			source := `
			func writer{output chan f32} (value f32) u8 {
			    out := output
			    out = value * 2.0
			    return 0
			}

			input_ch -> writer{output=write_target} -> sink_ch`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(ir.Nodes).To(HaveLen(3))

			// The writer function node should have write_target (200) in Channels.Write
			// NOT the intermediate variable's ID
			writerNode := ir.Nodes[1]
			Expect(writerNode.Type).To(Equal("writer"))
			Expect(writerNode.Channels.Write).To(HaveLen(1))
			Expect(writerNode.Channels.Write).To(HaveKey(uint32(10200)))

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})

		It("Should compile function with global channel assigned to intermediate variable and written to", func(ctx SpecContext) {
			// Test that writing through an alias of a global channel correctly tracks the channel
			// out := output (global channel)
			// out = value * 3.0 (write to channel through alias)
			resolver := []symbol.Symbol{
				{
					Name: "input_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   10110,
				},
				{
					Name: "output_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.F32()),
					ID:   10210,
				},
				{
					Name: "sink_ch",
					Kind: symbol.KindChannel,
					Type: types.Chan(types.U8()),
					ID:   10310,
				},
			}
			source := `
			func writer{} (value f32) u8 {
			    out := output_ch
			    out = value * 3.0
			    return 0
			}

			input_ch -> writer{} -> sink_ch`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			ir, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
			Expect(ir.Nodes).To(HaveLen(3))

			writerNode := ir.Nodes[1]
			Expect(writerNode.Type).To(Equal("writer"))
			Expect(writerNode.Channels.Write).To(HaveLen(1))
			Expect(writerNode.Channels.Write).To(HaveKey(uint32(10210)))

			module := MustSucceed(text.Compile(ctx, ir))
			Expect(module.Output.WASM).ToNot(BeEmpty())
		})
	})

	Describe("ExecContext", func() {
		It("Should reject a WASM-only function in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10042},
			}
			source := `
			func print{} () {
			}

			sensor -> len{} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.String()).To(ContainSubstring("cannot be used as a flow statement"))
		})

		It("Should allow a flow function in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10042},
			}
			source := `
			func print{} () {
			}

			sensor -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		It("Should allow an ExecBoth user function in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.I32()), ID: 10042},
			}
			source := `
			func handler{} () {
			}

			sensor -> handler{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		It("Should allow a flow-only STL function in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10042},
			}
			source := `
			func print{} () {
			}

			interval{100ms} -> print{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		It("Should allow an ExecBoth STL function (time.now) in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10042},
				{Name: "ts_out", Kind: symbol.KindChannel, Type: types.Chan(types.TimeStamp()), ID: 10043},
			}
			source := `
			import time
			interval{100ms} -> time.now{} -> ts_out
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		It("Should omit inputs on an STL ExecBoth node whose upstream is a trigger", func(ctx SpecContext) {
			bothType := types.Function(types.FunctionProperties{
				Inputs: types.Params{
					{Name: "key_or_name", Type: types.String()},
					{Name: "message", Type: types.String()},
					{Name: "variant", Type: types.String()},
				},
			})
			mod := &symbol.Symbol{Name: "stub", Kind: symbol.KindModule}
			mod.AddChild(&symbol.Symbol{
				Name: "both",
				Kind: symbol.KindFunction,
				Exec: symbol.ExecBoth,
				Type: bothType,
			})
			root := NewRoot(nil, symbol.Symbol{
				Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100,
			})
			root.Parent.AddChild(mod)
			source := `import stub
sensor -> stub.both{key_or_name="x", message="y", variant="info"}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diags := text.Analyze(ctx, parsedText, root)
			Expect(diags.Ok()).To(BeTrue(), diags.String())
			n := findNodeByType(inter.Nodes, "stub.both")
			Expect(n.Inputs).To(HaveLen(3))
		})

		It("Should keep inputs on a user-defined function in a flow statement", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100},
			}
			source := `
			func handler{} (x u8) {
			}
			sensor -> handler{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diags := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diags.Ok()).To(BeTrue(), diags.String())
			n := findNodeByType(inter.Nodes, "handler")
			Expect(n.Inputs).To(HaveLen(1))
			Expect(n.Inputs[0].Name).To(Equal("x"))
		})

		It("Should keep inputs on an ExecFlow STL node in a flow statement", func(ctx SpecContext) {
			sinkType := types.Function(types.FunctionProperties{
				Inputs: types.Params{{Name: "value", Type: types.U8()}},
			})
			mod := &symbol.Symbol{Name: "stub", Kind: symbol.KindModule}
			mod.AddChild(&symbol.Symbol{
				Name:    "sink",
				Kind:    symbol.KindFunction,
				Exec:    symbol.ExecFlow,
				Type:    sinkType,
				Trigger: symbol.TriggerInput("value"),
			})
			root := NewRoot(nil, symbol.Symbol{
				Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 100,
			})
			root.Parent.AddChild(mod)
			source := `import stub
sensor -> stub.sink{}`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			inter, diags := text.Analyze(ctx, parsedText, root)
			Expect(diags.Ok()).To(BeTrue(), diags.String())
			n := findNodeByType(inter.Nodes, "stub.sink")
			Expect(n.Inputs).To(HaveLen(1))
			Expect(n.Inputs[0].Name).To(Equal("value"))
		})

		It("Should allow time.now{} to write into a plain i64 channel", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "int_out", Kind: symbol.KindChannel, Type: types.Chan(types.I64()), ID: 10044},
			}
			source := `
			import time
			interval{100ms} -> time.now{} -> int_out
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeTrue(), diagnostics.String())
		})

		It("Should reject a flow-only function called in a func body at analysis time", func(ctx SpecContext) {
			resolver := []symbol.Symbol{
				{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 10042},
			}
			source := `
			func test() {
				interval()
			}

			sensor -> test{}
			`
			parsedText := MustSucceed(text.Parse(text.Text{Raw: source}))
			_, diagnostics := text.Analyze(ctx, parsedText, NewRoot(nil, resolver...))
			Expect(diagnostics.Ok()).To(BeFalse())
			Expect(diagnostics.Errors()[0].Message).To(ContainSubstring("cannot be called inside a func block"))
		})
	})
})
