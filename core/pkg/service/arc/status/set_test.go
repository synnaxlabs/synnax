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
	"context"
	"regexp"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	arctest "github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/arc/symbol"
	symboltestutil "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero"
)

type reportCall struct {
	variant xstatus.Variant
	message string
}

type recordingReporter struct {
	mu    sync.Mutex
	calls []reportCall
}

func (r *recordingReporter) report(_ context.Context, v xstatus.Variant, m string) {
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

// newModule builds a Module without WASM wiring (covered C++-side).
func newModule(ctx context.Context, reporter *recordingReporter) *arcstatus.Module {
	return MustSucceed(arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
		Status:   statSvc,
		Reporter: reporter.report,
	}))
}

// buildState builds an ir.IR + ProgramState directly (skipping graph.Analyze,
// which rejects unwired ExecBoth nodes) so node.Output(0) is usable.
func buildState(_ context.Context, bareType string, config types.Params) (*node.ProgramState, ir.Node) {
	irNode := ir.Node{Key: "n", Type: bareType, Config: config, Outputs: setOutputs}
	prog := ir.IR{Nodes: ir.Nodes{irNode}}
	return node.New(prog), irNode
}

var setOutputs = types.Params{
	{Name: ir.DefaultOutputParam, Type: types.String()},
}

func setConfig(keyOrName, message, variant string) types.Params {
	return types.Params{
		{Name: "key_or_name", Type: types.String(), Value: keyOrName},
		{Name: "message", Type: types.String(), Value: message},
		{Name: "variant", Type: types.String(), Value: variant},
	}
}

var _ = Describe("Symbols", func() {
	statusModule := func() *symbol.Symbol {
		for _, s := range arcstatus.NewSymbols() {
			if s.Kind == symbol.KindModule && s.Name == "status" {
				return s
			}
		}
		return nil
	}

	Describe("Module shape", func() {
		It("Should expose a single 'status' module", func() {
			mod := statusModule()
			Expect(mod).ToNot(BeNil())
			Expect(mod.Name).To(Equal("status"))
			Expect(mod.Kind).To(Equal(symbol.KindModule))
		})

		It("Should expose set as the only member", func() {
			children := statusModule().Children()
			names := make([]string, len(children))
			for i, c := range children {
				names[i] = c.Name
			}
			Expect(names).To(ConsistOf("set"))
		})

		It("Should not expose a deprecated bare set_status symbol", func() {
			for _, s := range arcstatus.NewSymbols() {
				Expect(s.Name).ToNot(Equal("set_status"))
			}
		})
	})

	Describe("Type Signature", func() {
		Describe("status.set", func() {
			var sym *symbol.Symbol
			BeforeEach(func() { sym = statusModule().FindChild("set") })

			It("Should be a KindFunction with ExecBoth", func() {
				Expect(sym).ToNot(BeNil())
				Expect(sym.Kind).To(Equal(symbol.KindFunction))
				Expect(sym.Exec).To(Equal(symbol.ExecBoth))
			})

			It("Should be a function type", func() {
				Expect(sym.Type.Kind).To(Equal(types.KindFunction))
			})

			It("Should have three string config parameters", func() {
				Expect(sym.Type.Config).To(HaveLen(3))
				Expect(sym.Type.Config[0].Name).To(Equal("key_or_name"))
				Expect(sym.Type.Config[0].Type).To(Equal(types.String()))
				Expect(sym.Type.Config[1].Name).To(Equal("message"))
				Expect(sym.Type.Config[1].Type).To(Equal(types.String()))
				Expect(sym.Type.Config[2].Name).To(Equal("variant"))
				Expect(sym.Type.Config[2].Type).To(Equal(types.String()))
			})

			It("Should mirror Config in Inputs (ExecBoth contract)", func() {
				Expect(sym.Type.Inputs).To(Equal(sym.Type.Config))
			})

			It("Should have one string output named ir.DefaultOutputParam", func() {
				Expect(sym.Type.Outputs).To(HaveLen(1))
				Expect(sym.Type.Outputs[0].Name).To(Equal(ir.DefaultOutputParam))
				Expect(sym.Type.Outputs[0].Type).To(Equal(types.String()))
			})
		})

	})
})

var _ = Describe("Module", func() {
	var (
		mod *arcstatus.Module
		rep *recordingReporter
	)
	BeforeEach(func(ctx SpecContext) {
		rep = &recordingReporter{}
		mod = newModule(ctx, rep)
	})

	Describe("Construction", func() {
		It("Should construct without WASM wiring when rat is nil", func(ctx SpecContext) {
			Expect(mod).ToNot(BeNil())
			Expect(mod.ModuleName()).To(Equal("status"))
		})

		It("Should wire host functions when a wazero runtime is provided", func(ctx SpecContext) {
			rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
			defer func() { Expect(rt.Close(ctx)).To(Succeed()) }()
			wired := MustSucceed(arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
				Status:   statSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			}))
			Expect(wired).ToNot(BeNil())
			Expect(wired.ModuleName()).To(Equal("status"))
		})

		It("Should error when the runtime can't instantiate the host module", func(ctx SpecContext) {
			rt := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
			defer func() { Expect(rt.Close(ctx)).To(Succeed()) }()
			MustSucceed(arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
				Status:   statSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			}))
			Expect(arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
				Status:   statSvc,
				Strings:  stlstrings.NewProgramState(),
				Runtime:  rt,
				Reporter: rep.report,
			})).Error().To(MatchError(ContainSubstring("status")))
		})
	})

	Describe("Create", func() {
		It("Should return ErrNotFound for an unrecognized type", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "wrong_type"}}
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should construct a set node from valid config", func(ctx SpecContext) {
			state, irNode := buildState(ctx, "set", setConfig("alarm", "msg", "info"))
			n := MustSucceed(mod.Create(ctx, node.Config{
				Node:  irNode,
				State: state.Node(irNode.Key),
			}))
			Expect(n).ToNot(BeNil())
			Expect(func() { n.Reset() }).ToNot(Panic())
			// Output(0) hasn't been written yet; truthiness reads the empty cache.
			Expect(n.IsOutputTruthy(0)).To(BeFalse())
		})

		// Missing config returns an error instead of panicking on unchecked assertion.
		It("Should return a clean error when set config is missing key_or_name", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{Type: "set", Config: types.Params{
					{Name: "message", Type: types.String(), Value: "x"},
					{Name: "variant", Type: types.String(), Value: "info"},
				}},
			}
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(ContainSubstring("status.set config")))
		})

		It("Should return a clean error when set config is missing variant", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{Type: "set", Config: types.Params{
					{Name: "key_or_name", Type: types.String(), Value: "x"},
					{Name: "message", Type: types.String(), Value: "y"},
				}},
			}
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(ContainSubstring("status.set config")))
		})
	})
})

var _ = Describe("setNode.Next", func() {
	var (
		mod *arcstatus.Module
		rep *recordingReporter
	)
	BeforeEach(func(ctx SpecContext) {
		rep = &recordingReporter{}
		mod = newModule(ctx, rep)
	})

	build := func(ctx context.Context, keyOrName, message, variant string) (node.Node, *node.State) {
		state, irNode := buildState(ctx, "set", setConfig(keyOrName, message, variant))
		s := state.Node(irNode.Key)
		n := MustSucceed(mod.Create(ctx, node.Config{Node: irNode, State: s}))
		return n, s
	}

	nodeCtx := func(ctx context.Context) node.Context {
		return node.Context{Context: ctx, MarkChanged: func(int) {}}
	}

	It("Should upsert a new UUID-keyed row by name when none exists", func(ctx SpecContext) {
		name := "next_new_" + uuid.NewString()
		n, state := build(ctx, name, "All good", "success")
		n.Next(nodeCtx(ctx))

		out := *state.Output(0)
		keys := telem.UnmarshalSeries[string](out)
		Expect(keys).To(HaveLen(1))
		newKey := keys[0]
		MustSucceed(uuid.Parse(newKey))
		Expect(newKey).ToNot(Equal(name))

		var s status.Status[any]
		Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](newKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
		Expect(s.Key).To(Equal(newKey))
		Expect(s.Name).To(Equal(name))
		Expect(s.Variant).To(Equal(xstatus.VariantSuccess))
		Expect(s.Message).To(Equal("All good"))
		Expect(s.Time).ToNot(BeZero())

		outTime := *state.OutputTime(0)
		Expect(outTime.Len()).To(Equal(int64(1)))
	})

	It("Should update an existing row by name (single match)", func(ctx SpecContext) {
		name := "next_single_" + uuid.NewString()
		existingKey := uuid.NewString()
		Expect(statSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
			Key: existingKey, Name: name, Variant: xstatus.VariantInfo, Message: "orig", Time: telem.Now(),
		})).To(Succeed())

		n, _ := build(ctx, name, "updated", "warning")
		n.Next(nodeCtx(ctx))

		var s status.Status[any]
		Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](existingKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
		Expect(s.Variant).To(Equal(xstatus.VariantWarning))
		Expect(s.Message).To(Equal("updated"))
	})

	It("Should update an existing row by UUID key", func(ctx SpecContext) {
		key := uuid.NewString()
		Expect(statSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
			Key: key, Name: "by_uuid", Variant: xstatus.VariantInfo, Message: "orig", Time: telem.Now(),
		})).To(Succeed())

		n, state := build(ctx, key, "via uuid", "error")
		n.Next(nodeCtx(ctx))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{key}))

		var s status.Status[any]
		Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](key)).Entry(&s).Exec(ctx, nil)).To(Succeed())
		Expect(s.Variant).To(Equal(xstatus.VariantError))
		Expect(s.Message).To(Equal("via uuid"))
	})

	It("Should produce non-decreasing timestamps on successive Next calls", func(ctx SpecContext) {
		name := "next_time_" + uuid.NewString()
		n, state := build(ctx, name, "msg", "info")
		nctx := nodeCtx(ctx)
		n.Next(nctx)
		first := telem.ValueAt[telem.TimeStamp](*state.OutputTime(0), 0)
		n.Next(nctx)
		second := telem.ValueAt[telem.TimeStamp](*state.OutputTime(0), 0)
		Expect(second).To(BeNumerically(">=", first))
	})

	It("Should warn and not write when the variant is unknown", func(ctx SpecContext) {
		name := "next_iv_" + uuid.NewString()
		n, state := build(ctx, name, "msg", "not_a_real_variant")
		n.Next(nodeCtx(ctx))

		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(xstatus.VariantWarning))
		Expect(calls[0].message).To(HavePrefix("status.set:"))
		Expect(calls[0].message).To(ContainSubstring("invalid status variant"))

		Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](name)).
			Entry(&status.Status[any]{}).Exec(ctx, nil)).To(MatchError(query.ErrNotFound))

		// On invalid variant, Output(0) carries a single empty-string sample.
		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{""}))
	})

	It("Should reject upper-cased variants as case-sensitive", func(ctx SpecContext) {
		name := "next_iv_case_" + uuid.NewString()
		n, _ := build(ctx, name, "msg", "SUCCESS")
		n.Next(nodeCtx(ctx))
		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].message).To(ContainSubstring("invalid status variant"))
	})

	It("Should warn on multi-match, update the first match, and report the resolved key", func(ctx SpecContext) {
		name := "next_multi_" + uuid.NewString()
		k1, k2 := uuid.NewString(), uuid.NewString()
		Expect(statSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
			Key: k1, Name: name, Variant: xstatus.VariantInfo, Message: "first", Time: telem.Now(),
		})).To(Succeed())
		Expect(statSvc.NewWriter(nil).Set(ctx, &status.Status[any]{
			Key: k2, Name: name, Variant: xstatus.VariantInfo, Message: "second", Time: telem.Now(),
		})).To(Succeed())

		n, state := build(ctx, name, "now updated", "warning")
		n.Next(nodeCtx(ctx))

		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(xstatus.VariantWarning))
		// Exact format: `status.set: multiple statuses named "name"; updated first match (<uuid>)`.
		re := regexp.MustCompile(`^status\.set: multiple statuses named "[^"]+"; updated first match \([0-9a-f-]+\)$`)
		Expect(re.MatchString(calls[0].message)).To(BeTrue(), "got: %q", calls[0].message)

		resolvedRows := telem.UnmarshalSeries[string](*state.Output(0))
		Expect(resolvedRows).To(HaveLen(1))
		Expect(resolvedRows[0]).To(SatisfyAny(Equal(k1), Equal(k2)))

		// Only one row picked up the new variant.
		var rows []status.Status[any]
		Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](k1, k2)).Entries(&rows).Exec(ctx, nil)).To(Succeed())
		warning, info := 0, 0
		for _, r := range rows {
			switch r.Variant {
			case xstatus.VariantWarning:
				warning++
			case xstatus.VariantInfo:
				info++
			}
		}
		Expect(warning).To(Equal(1))
		Expect(info).To(Equal(1))
	})

	It("Should warn with a 'status.set:' prefix when the service returns an error", func(ctx SpecContext) {
		// Empty keyOrName trips the service-side required-input validation guard.
		n, state := build(ctx, "", "msg", "info")
		n.Next(nodeCtx(ctx))

		calls := rep.get()
		Expect(calls).To(HaveLen(1))
		Expect(calls[0].variant).To(Equal(xstatus.VariantWarning))
		Expect(calls[0].message).To(HavePrefix("status.set:"))

		Expect(telem.UnmarshalSeries[string](*state.Output(0))).To(Equal([]string{""}))
	})
})

var _ = Describe("Analyzer hooks", func() {
	// Exercise compile-time variant validation through text.Analyze.
	channels := []symbol.Symbol{
		{Name: "sensor", Kind: symbol.KindChannel, Type: types.Chan(types.U8()), ID: 1},
		{Name: "output", Kind: symbol.KindChannel, Type: types.WriteChan(types.U8()), ID: 2},
	}
	buildRoot := func() *symbol.Symbol {
		root := symboltestutil.NewRoot(nil, channels...)
		for _, s := range arcstatus.NewSymbols() {
			root.Parent.AddChild(s)
		}
		return root
	}

	analyzeOK := func(ctx context.Context, src string) bool {
		parsed := MustSucceed(text.Parse(text.Text{Raw: src}))
		_, diags := text.Analyze(ctx, parsed, buildRoot())
		return diags.Ok()
	}

	expectInvalidVariantError := func(ctx context.Context, src, badVariant string) {
		parsed := MustSucceed(text.Parse(text.Text{Raw: src}))
		_, diags := text.Analyze(ctx, parsed, buildRoot())
		Expect(diags.Ok()).To(BeFalse())
		errs := diags.Errors()
		Expect(errs).ToNot(BeEmpty())
		found := false
		for _, e := range errs {
			if regexp.MustCompile(regexp.QuoteMeta(badVariant) + `.*not a valid status variant`).MatchString(e.Message) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(), "expected an invalid-variant diagnostic mentioning %q, got: %+v", badVariant, errs)
	}

	Describe("flow form (analyzeStatusSetFlowConfig)", func() {
		It("Should flag an invalid variant in named config", func(ctx SpecContext) {
			expectInvalidVariantError(ctx,
				"import status\nsensor -> status.set{key_or_name=\"alarm\", message=\"bad\", variant=\"bogus\"}",
				"bogus")
		})

		It("Should reject upper-cased variants (case-sensitive)", func(ctx SpecContext) {
			expectInvalidVariantError(ctx,
				"import status\nsensor -> status.set{key_or_name=\"alarm\", message=\"bad\", variant=\"SUCCESS\"}",
				"SUCCESS")
		})

		It("Should accept a valid variant", func(ctx SpecContext) {
			// Variant hook only; trigger type wiring is orthogonal (not on this branch).
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nsensor -> status.set{key_or_name=\"alarm\", message=\"ok\", variant=\"success\"}"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})

		It("Should not flag a missing variant entry", func(ctx SpecContext) {
			// No variant key set; analyzer skips silently. We only assert that no
			// "not a valid variant" diagnostic fires.
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nsensor -> status.set{key_or_name=\"a\", message=\"b\"}"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})

		It("Should flag an invalid variant in anonymous (positional) config", func(ctx SpecContext) {
			expectInvalidVariantError(ctx,
				"import status\nsensor -> status.set{\"alarm\", \"bad\", \"bogus\"}",
				"bogus")
		})

		It("Should accept a valid variant in anonymous (positional) config", func(ctx SpecContext) {
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nsensor -> status.set{\"alarm\", \"ok\", \"success\"}"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})

		It("Should not flag positional config that omits the variant slot", func(ctx SpecContext) {
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nsensor -> status.set{\"a\", \"b\"}"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})
	})

	Describe("func form (analyzeStatusSetCall)", func() {
		It("Should flag an invalid variant in a call", func(ctx SpecContext) {
			expectInvalidVariantError(ctx,
				"import status\nfunc body() { status.set(\"alarm\", \"bad\", \"bogus\") }",
				"bogus")
		})

		It("Should accept a valid variant in a call", func(ctx SpecContext) {
			Expect(analyzeOK(ctx,
				"import status\nfunc body() { status.set(\"alarm\", \"ok\", \"success\") }",
			)).To(BeTrue())
		})

		It("Should not flag a call with fewer than 3 args", func(ctx SpecContext) {
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nfunc body() { status.set(\"a\", \"b\") }"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})
	})

	Describe("checkVariantLiteral (non-literal variant expressions)", func() {
		It("Should not flag a non-string-literal variant", func(ctx SpecContext) {
			parsed := MustSucceed(text.Parse(text.Text{Raw: "import status\nsensor -> status.set{key_or_name=\"a\", message=\"b\", variant=42}"}))
			_, diags := text.Analyze(ctx, parsed, buildRoot())
			for _, e := range diags.Errors() {
				Expect(e.Message).ToNot(ContainSubstring("not a valid status variant"))
			}
		})
	})
})

var _ = Describe("WASM host functions", func() {
	var (
		rt   *arctest.Runtime
		strs *stlstrings.ProgramState
		rep  *recordingReporter
	)

	BeforeEach(func(ctx SpecContext) {
		rt = arctest.NewRuntime(ctx)
		strs = stlstrings.NewProgramState()
		rep = &recordingReporter{}
		MustSucceed(arcstatus.NewModule(ctx, arcstatus.ModuleConfig{
			Status:   statSvc,
			Strings:  strs,
			Runtime:  rt.Underlying(),
			Reporter: rep.report,
		}))
		rt.Passthrough(ctx, "status")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("set", func() {
		It("Should upsert a status and return a non-zero handle resolving to the key", func(ctx SpecContext) {
			name := "wasm_set_" + uuid.NewString()
			keyH := strs.Create(name)
			msgH := strs.Create("from wasm")
			varH := strs.Create("info")

			res := rt.Call(ctx, "status", "set",
				arctest.U32(keyH), arctest.U32(msgH), arctest.U32(varH))
			out := arctest.AsU32(res[0])
			Expect(out).ToNot(BeZero())
			newKey := MustBeOk(strs.Get(out))
			MustSucceed(uuid.Parse(newKey))
			Expect(newKey).ToNot(Equal(name))

			var s status.Status[any]
			Expect(statSvc.NewRetrieve().Where(status.MatchKeys[any](newKey)).Entry(&s).Exec(ctx, nil)).To(Succeed())
			Expect(s.Name).To(Equal(name))
			Expect(s.Variant).To(Equal(xstatus.VariantInfo))
		})

		It("Should warn and return 0 on an invalid key_or_name handle", func(ctx SpecContext) {
			msgH := strs.Create("m")
			varH := strs.Create("info")
			res := rt.Call(ctx, "status", "set",
				arctest.U32(9999), arctest.U32(msgH), arctest.U32(varH))
			Expect(arctest.AsU32(res[0])).To(Equal(uint32(0)))
			calls := rep.get()
			Expect(calls).To(HaveLen(1))
			Expect(calls[0].variant).To(Equal(xstatus.VariantWarning))
			Expect(calls[0].message).To(ContainSubstring("status.set: invalid string handle"))
		})

		It("Should warn and return 0 on an invalid message handle", func(ctx SpecContext) {
			keyH := strs.Create("wasm_msg_h_" + uuid.NewString())
			varH := strs.Create("info")
			res := rt.Call(ctx, "status", "set",
				arctest.U32(keyH), arctest.U32(9999), arctest.U32(varH))
			Expect(arctest.AsU32(res[0])).To(Equal(uint32(0)))
			Expect(rep.get()).To(HaveLen(1))
		})

		It("Should warn and return 0 on an invalid variant handle", func(ctx SpecContext) {
			keyH := strs.Create("wasm_var_h_" + uuid.NewString())
			msgH := strs.Create("m")
			res := rt.Call(ctx, "status", "set",
				arctest.U32(keyH), arctest.U32(msgH), arctest.U32(9999))
			Expect(arctest.AsU32(res[0])).To(Equal(uint32(0)))
			Expect(rep.get()).To(HaveLen(1))
		})
	})
})
