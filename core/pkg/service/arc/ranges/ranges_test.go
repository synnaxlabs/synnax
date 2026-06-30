// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranges_test

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	. "github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	arcranges "github.com/synnaxlabs/synnax/pkg/service/arc/ranges"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

type reportCall struct {
	variant status.Variant
	message string
}

type recordingReporter struct {
	mu    sync.Mutex
	calls []reportCall
}

func (r *recordingReporter) report(_ context.Context, v status.Variant, m string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, reportCall{variant: v, message: m})
}

func (r *recordingReporter) get() []reportCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]reportCall, len(r.calls))
	copy(out, r.calls)
	return out
}

// newModule builds a Module without WASM wiring (WASM is covered separately).
func newModule(ctx context.Context, reporter *recordingReporter) node.Factory {
	return MustSucceed(arcranges.NewModule(ctx, arcranges.ModuleConfig{
		Ranger:   rangeSvc,
		Reporter: reporter.report,
	}))
}

var rangesOutputs = types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}}

// buildState builds an ir.IR + ProgramState directly (skipping graph.Analyze, which
// rejects unwired ExecBoth nodes) so node.Output(0) is usable.
func buildState(bareType string, inputs types.Params) (*node.ProgramState, ir.Node) {
	irNode := ir.Node{Key: "n", Type: bareType, Inputs: inputs, Outputs: rangesOutputs}
	prog := ir.IR{Nodes: ir.Nodes{irNode}}
	return node.New(prog), irNode
}

func createInputs(name, colorHex, parent string) types.Params {
	return types.Params{
		{Name: "name", Type: types.String(), Value: name},
		{Name: "color", Type: types.String(), Value: colorHex},
		{Name: "parent", Type: types.String(), Value: parent},
	}
}

func endInputs(key string) types.Params {
	return types.Params{
		{Name: "key", Type: types.String(), Value: key},
	}
}

func nodeCtx(ctx context.Context) node.Context {
	return node.Context{Context: ctx, MarkChanged: func(int) {}}
}

// matchesColor reports whether a diagnostic message is a color.FromCSS rejection.
func matchesColor(msg string) bool {
	return strings.Contains(msg, "hex value") || strings.Contains(msg, "invalid hex color")
}

func retrieveRange(ctx context.Context, key string) (ranger.Range, error) {
	uid, err := uuid.Parse(key)
	if err != nil {
		return ranger.Range{}, err
	}
	var r ranger.Range
	err = rangeSvc.NewRetrieve().Where(ranger.MatchKeys(uid)).Entry(&r).Exec(ctx, nil)
	return r, err
}

var _ = Describe("Symbols", func() {
	rangesModule := func() *symbol.Symbol {
		for _, s := range arcranges.NewSymbols() {
			if s.Kind == symbol.KindModule && s.Name == "ranges" {
				return s
			}
		}
		return nil
	}

	Describe("Module shape", func() {
		It("Should expose a single 'ranges' module", func() {
			mod := rangesModule()
			Expect(mod).ToNot(BeNil())
			Expect(mod.Name).To(Equal("ranges"))
			Expect(mod.Kind).To(Equal(symbol.KindModule))
		})

		It("Should expose create and end as the only members", func() {
			children := rangesModule().Children()
			names := make([]string, len(children))
			for i, c := range children {
				names[i] = c.Name
			}
			Expect(names).To(ConsistOf("create", "end"))
		})

		It("Should not expose deprecated start or label members", func() {
			children := rangesModule().Children()
			for _, c := range children {
				Expect(c.Name).ToNot(Equal("start"))
			}
		})
	})

	Describe("ranges.create type signature", func() {
		var sym *symbol.Symbol
		BeforeEach(func() { sym = rangesModule().FindChild("create") })

		It("Should be a KindFunction with ExecBoth", func() {
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
			Expect(sym.Exec).To(Equal(symbol.ExecBoth))
		})

		It("Should be TriggerOnly", func() {
			Expect(sym.Trigger).To(Equal(symbol.TriggerOnly))
		})

		It("Should have name (required), color and parent (optional) string inputs", func() {
			Expect(sym.Type.Inputs).To(HaveLen(3))
			Expect(sym.Type.Inputs[0].Name).To(Equal("name"))
			Expect(sym.Type.Inputs[0].Type).To(Equal(types.String()))
			Expect(sym.Type.Inputs[0].Value).To(BeNil())
			Expect(sym.Type.Inputs[1].Name).To(Equal("color"))
			Expect(sym.Type.Inputs[1].Value).To(Equal(""))
			Expect(sym.Type.Inputs[2].Name).To(Equal("parent"))
			Expect(sym.Type.Inputs[2].Value).To(Equal(""))
		})

		It("Should have a single string output", func() {
			Expect(sym.Type.Outputs).To(HaveLen(1))
			Expect(sym.Type.Outputs[0].Name).To(Equal(ir.DefaultOutputParam))
			Expect(sym.Type.Outputs[0].Type).To(Equal(types.String()))
		})

		It("Should register an argument analyzer for color validation", func() {
			Expect(sym.AnalyzeArguments).ToNot(BeNil())
		})
	})

	Describe("ranges.end type signature", func() {
		var sym *symbol.Symbol
		BeforeEach(func() { sym = rangesModule().FindChild("end") })

		It("Should be a TriggerOnly KindFunction with ExecBoth", func() {
			Expect(sym.Kind).To(Equal(symbol.KindFunction))
			Expect(sym.Exec).To(Equal(symbol.ExecBoth))
			Expect(sym.Trigger).To(Equal(symbol.TriggerOnly))
		})

		It("Should have key (required string) as its only input", func() {
			Expect(sym.Type.Inputs).To(HaveLen(1))
			Expect(sym.Type.Inputs[0].Name).To(Equal("key"))
			Expect(sym.Type.Inputs[0].Type).To(Equal(types.String()))
			Expect(sym.Type.Inputs[0].Value).To(BeNil())
		})
	})
})

var _ = Describe("Module", func() {
	var (
		mod node.Factory
		rep *recordingReporter
	)
	BeforeEach(func(ctx SpecContext) {
		rep = &recordingReporter{}
		mod = newModule(ctx, rep)
	})

	Describe("Construction", func() {
		It("Should construct without WASM wiring when the runtime is nil", func() {
			Expect(mod).ToNot(BeNil())
		})

		It("Should wire host functions when a wazero runtime is provided", func(ctx SpecContext) {
			rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
			DeferCleanup(rt.Close)
			wired := MustSucceed(arcranges.NewModule(ctx, arcranges.ModuleConfig{
				Ranger:   rangeSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			}))
			Expect(wired).ToNot(BeNil())
		})

		It("Should error when the runtime can't re-instantiate the host module", func(ctx SpecContext) {
			rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
			DeferCleanup(rt.Close)
			MustSucceed(arcranges.NewModule(ctx, arcranges.ModuleConfig{
				Ranger:   rangeSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			}))
			Expect(arcranges.NewModule(ctx, arcranges.ModuleConfig{
				Ranger:   rangeSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			})).Error().To(MatchError(ContainSubstring("ranges")))
		})
	})

	Describe("Create", func() {
		It("Should return ErrNotFound for an unrecognized type", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "wrong_type"}}
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should construct a create node from valid inputs", func(ctx SpecContext) {
			state, irNode := buildState("create", createInputs("rng", "", ""))
			n := MustSucceed(mod.Create(ctx, node.Config{Node: irNode, State: state.Node(irNode.Key)}))
			Expect(n).ToNot(BeNil())
			Expect(func() { n.Reset() }).ToNot(Panic())
			Expect(n.IsOutputTruthy(0)).To(BeFalse())
		})

		It("Should construct an end node from valid inputs", func(ctx SpecContext) {
			state, irNode := buildState("end", endInputs(uuid.NewString()))
			n := MustSucceed(mod.Create(ctx, node.Config{Node: irNode, State: state.Node(irNode.Key)}))
			Expect(n).ToNot(BeNil())
		})

		It("Should return a clean error when create is missing name", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "create", Inputs: types.Params{
				{Name: "color", Type: types.String(), Value: ""},
				{Name: "parent", Type: types.String(), Value: ""},
			}, Outputs: rangesOutputs}}
			state := node.New(ir.IR{Nodes: ir.Nodes{cfg.Node}})
			cfg.State = state.Node("")
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(ContainSubstring("ranges.create inputs")))
		})

		It("Should return a clean error when end is missing key", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "end", Inputs: types.Params{}, Outputs: rangesOutputs}}
			state := node.New(ir.IR{Nodes: ir.Nodes{cfg.Node}})
			cfg.State = state.Node("")
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(ContainSubstring("ranges.end inputs")))
		})
	})
})

var _ = Describe("createNode.Next", func() {
	var (
		mod node.Factory
		rep *recordingReporter
	)
	BeforeEach(func(ctx SpecContext) {
		rep = &recordingReporter{}
		mod = newModule(ctx, rep)
	})

	build := func(ctx context.Context, name, colorHex, parent string) (node.Node, *node.State) {
		state, irNode := buildState("create", createInputs(name, colorHex, parent))
		s := state.Node(irNode.Key)
		return MustSucceed(mod.Create(ctx, node.Config{Node: irNode, State: s})), s
	}

	It("Should create an open range that starts now and ends at max", func(ctx SpecContext) {
		name := "create_open_" + uuid.NewString()
		before := telem.Now()
		n, state := build(ctx, name, "", "")
		n.Next(nodeCtx(ctx))

		keys := telem.UnmarshalSeries[string](*state.Output(0))
		Expect(keys).To(HaveLen(1))
		newKey := keys[0]
		MustSucceed(uuid.Parse(newKey))

		r := MustSucceed(retrieveRange(ctx, newKey))
		Expect(r.Name).To(Equal(name))
		Expect(r.TimeRange.End).To(Equal(telem.TimeStampMax))
		Expect(r.TimeRange.Start).To(BeNumerically(">=", before))
		Expect(r.TimeRange.Start).To(BeNumerically("<=", telem.Now()))
		Expect(rep.get()).To(BeEmpty())

		Expect((*state.OutputTime(0)).Len()).To(Equal(int64(1)))
	})

	It("Should store a parsed color when a valid hex is provided", func(ctx SpecContext) {
		name := "create_color_" + uuid.NewString()
		n, state := build(ctx, name, "#DF6D38", "")
		n.Next(nodeCtx(ctx))

		newKey := telem.UnmarshalSeries[string](*state.Output(0))[0]
		r := MustSucceed(retrieveRange(ctx, newKey))
		Expect(*r.Color).To(Equal(MustSucceed(color.FromCSS("#DF6D38"))))
		Expect(rep.get()).To(BeEmpty())
	})

	It("Should accept an rgb() color", func(ctx SpecContext) {
		name := "create_rgb_" + uuid.NewString()
		n, state := build(ctx, name, "rgb(223, 109, 56)", "")
		n.Next(nodeCtx(ctx))
		newKey := telem.UnmarshalSeries[string](*state.Output(0))[0]
		r := MustSucceed(retrieveRange(ctx, newKey))
		Expect(*r.Color).To(Equal(MustSucceed(color.FromCSS("rgb(223, 109, 56)"))))
		Expect(rep.get()).To(BeEmpty())
	})

	It("Should warn but still create the range when the color is invalid", func(ctx SpecContext) {
		name := "create_badcolor_" + uuid.NewString()
		n, state := build(ctx, name, "not-a-color", "")
		n.Next(nodeCtx(ctx))

		newKey := telem.UnmarshalSeries[string](*state.Output(0))[0]
		r := MustSucceed(retrieveRange(ctx, newKey))
		Expect(r.Name).To(Equal(name))
		Expect(r.Color).To(BeNil())

		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(status.VariantWarning))
		Expect(calls[0].message).To(ContainSubstring("ranges.create: invalid color"))
	})

	It("Should parent the new range under an existing parent range", func(ctx SpecContext) {
		parent := ranger.Range{
			Name:      "parent_" + uuid.NewString(),
			TimeRange: telem.TimeRange{Start: telem.Now(), End: telem.TimeStampMax},
		}
		Expect(rangeSvc.NewWriter(nil).Create(ctx, &parent)).To(Succeed())

		n, state := build(ctx, "child_"+uuid.NewString(), "", parent.Key.String())
		n.Next(nodeCtx(ctx))

		newKey := telem.UnmarshalSeries[string](*state.Output(0))[0]
		Expect(newKey).ToNot(BeEmpty())
		MustSucceed(retrieveRange(ctx, newKey))
		Expect(rep.get()).To(BeEmpty())
	})

	It("Should warn and emit an empty key when the parent key is not a UUID", func(ctx SpecContext) {
		n, state := build(ctx, "bad_parent_"+uuid.NewString(), "", "not-a-uuid")
		n.Next(nodeCtx(ctx))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{""}))
		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(status.VariantWarning))
		Expect(calls[0].message).To(ContainSubstring("ranges.create: invalid parent key"))
	})
})

var _ = Describe("endNode.Next", func() {
	var (
		mod node.Factory
		rep *recordingReporter
	)
	BeforeEach(func(ctx SpecContext) {
		rep = &recordingReporter{}
		mod = newModule(ctx, rep)
	})

	build := func(ctx context.Context, key string) (node.Node, *node.State) {
		state, irNode := buildState("end", endInputs(key))
		s := state.Node(irNode.Key)
		return MustSucceed(mod.Create(ctx, node.Config{Node: irNode, State: s})), s
	}

	openRange := func(ctx context.Context) ranger.Range {
		r := ranger.Range{
			Name:      "end_target_" + uuid.NewString(),
			TimeRange: telem.TimeRange{Start: telem.Now(), End: telem.TimeStampMax},
		}
		Expect(rangeSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		return r
	}

	It("Should set the end bound to now", func(ctx SpecContext) {
		r := openRange(ctx)
		before := telem.Now()
		n, state := build(ctx, r.Key.String())
		n.Next(nodeCtx(ctx))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{r.Key.String()}))
		updated := MustSucceed(retrieveRange(ctx, r.Key.String()))
		Expect(updated.TimeRange.End).To(BeNumerically(">=", before))
		Expect(updated.TimeRange.End).To(BeNumerically("<=", telem.Now()))
		Expect(updated.TimeRange.End).ToNot(Equal(telem.TimeStampMax))
		Expect(rep.get()).To(BeEmpty())
	})

	It("Should warn and emit an empty key when the key is not a UUID", func(ctx SpecContext) {
		n, state := build(ctx, "not-a-uuid")
		n.Next(nodeCtx(ctx))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{""}))
		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(status.VariantWarning))
		Expect(calls[0].message).To(ContainSubstring("ranges.end: invalid range key"))
	})

	It("Should warn when the range does not exist", func(ctx SpecContext) {
		n, state := build(ctx, uuid.NewString())
		n.Next(nodeCtx(ctx))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{""}))
		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(status.VariantWarning))
		Expect(calls[0].message).To(HavePrefix("ranges.end:"))
	})
})

var _ = Describe("Analyzer hooks", func() {
	channels := []symbol.Symbol{
		{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
	}
	buildRoot := func() *symbol.Symbol {
		root := NewRoot(nil, channels...)
		for _, s := range arcranges.NewSymbols() {
			root.Parent.AddChild(s)
		}
		return root
	}
	hasColorError := func(ctx context.Context, src string) bool {
		parsed := MustSucceed(text.Parse(text.Text{Raw: src}))
		_, diags := text.Analyze(ctx, parsed, buildRoot())
		for _, e := range diags.Errors() {
			if matchesColor(e.Message) {
				return true
			}
		}
		return false
	}

	It("Should flag an invalid color literal in a named argument", func(ctx SpecContext) {
		Expect(hasColorError(ctx,
			"import ranges\nsensor -> ranges.create{name=\"r\", color=\"not-a-color\"}")).To(BeTrue())
	})

	It("Should flag an invalid color literal in a positional argument", func(ctx SpecContext) {
		Expect(hasColorError(ctx,
			"import ranges\nsensor -> ranges.create{\"r\", \"not-a-color\"}")).To(BeTrue())
	})

	It("Should accept a valid color literal", func(ctx SpecContext) {
		Expect(hasColorError(ctx,
			"import ranges\nsensor -> ranges.create{name=\"r\", color=\"#DF6D38\"}")).To(BeFalse())
	})

	It("Should not flag a missing color argument", func(ctx SpecContext) {
		Expect(hasColorError(ctx,
			"import ranges\nsensor -> ranges.create{name=\"r\"}")).To(BeFalse())
	})
})

var _ = Describe("WASM host functions", func() {
	var (
		rt   *Runtime
		strs *stlstrings.ProgramState
		rep  *recordingReporter
	)

	BeforeEach(func(ctx SpecContext) {
		rt = NewRuntime(ctx)
		strs = stlstrings.NewProgramState()
		rep = &recordingReporter{}
		MustSucceed(arcranges.NewModule(ctx, arcranges.ModuleConfig{
			Ranger:   rangeSvc,
			Strings:  strs,
			Runtime:  rt.Underlying(),
			Reporter: rep.report,
		}))
		rt.Passthrough(ctx, "ranges")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("create", func() {
		It("Should create a range and return a handle resolving to its key", func(ctx SpecContext) {
			name := "wasm_create_" + uuid.NewString()
			nameH := strs.Create(name)
			colorH := strs.Create("")
			parentH := strs.Create("")

			res := rt.Call(ctx, "ranges", "create",
				U32(nameH), U32(colorH), U32(parentH))
			out := AsU32(res[0])
			Expect(out).ToNot(BeZero())
			newKey := MustBeOk(strs.Get(out))
			r := MustSucceed(retrieveRange(ctx, newKey))
			Expect(r.Name).To(Equal(name))
			Expect(r.TimeRange.End).To(Equal(telem.TimeStampMax))
		})

		It("Should warn and return 0 on an invalid name handle", func(ctx SpecContext) {
			colorH := strs.Create("")
			parentH := strs.Create("")
			res := rt.Call(ctx, "ranges", "create",
				U32(9999), U32(colorH), U32(parentH))
			Expect(AsU32(res[0])).To(Equal(uint32(0)))
			calls := rep.get()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].message).To(ContainSubstring("ranges.create: invalid string handle"))
		})
	})

	Describe("end", func() {
		It("Should set the end bound to now and return a handle resolving to the key", func(ctx SpecContext) {
			r := ranger.Range{
				Name:      "wasm_end_" + uuid.NewString(),
				TimeRange: telem.TimeRange{Start: telem.Now(), End: telem.TimeStampMax},
			}
			Expect(rangeSvc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			keyH := strs.Create(r.Key.String())
			before := telem.Now()

			res := rt.Call(ctx, "ranges", "end", U32(keyH))
			out := AsU32(res[0])
			Expect(MustBeOk(strs.Get(out))).To(Equal(r.Key.String()))
			updated := MustSucceed(retrieveRange(ctx, r.Key.String()))
			Expect(updated.TimeRange.End).To(BeNumerically(">=", before))
			Expect(updated.TimeRange.End).To(BeNumerically("<=", telem.Now()))
			Expect(updated.TimeRange.End).ToNot(Equal(telem.TimeStampMax))
		})

		It("Should warn and return 0 on an invalid key handle", func(ctx SpecContext) {
			res := rt.Call(ctx, "ranges", "end", U32(9999))
			Expect(AsU32(res[0])).To(Equal(uint32(0)))
			calls := rep.get()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].message).To(ContainSubstring("ranges.end: invalid string handle"))
		})
	})
})
