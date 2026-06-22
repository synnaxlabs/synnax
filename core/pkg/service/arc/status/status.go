// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"fmt"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/taskreporter"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
	"github.com/tetratelabs/wazero"
)

const (
	setMemberName = "set"
	moduleName    = "status"
)

// allowedVariantsList is the human-readable list used in compile-time and
// runtime diagnostics. Variant.IsValid is the source of truth for membership.
const allowedVariantsList = "success, info, warning, error, loading, disabled"

// memberDoc is the LSP hover body for status.set. The renderer prepends the
// title from the symbol name and kind, so it is omitted here.
var memberDoc = doc.New(
	doc.Paragraph("Sets a status notification on the cluster. Used to report alarms, warnings, or operational state."),
	doc.Divider(),
	doc.Code("arc", "status_key := sensor -> status.set{key_or_name=\"ox_alarm\", variant=\"error\", message=\"Overpressure\"}"),
	doc.Divider(),
	doc.Paragraph("Accepted variants: success, info, warning, error, loading, disabled."),
	doc.Divider(),
	doc.Paragraph("Outputs the resolved status key as a string."),
)

// moduleDoc is the LSP hover body for the status module itself.
var moduleDoc = doc.New(
	doc.Paragraph("Publishes status notifications (alarms, warnings, operational state) to the cluster."),
)

// newSetSymbolType returns a fresh function type for status.set per call so
// analysis never mutates a shared symbol. Empty defaults keep inputs optional.
func newSetSymbolType() types.Type {
	params := types.Params{
		{Name: "key_or_name", Type: types.String(), Value: ""},
		{Name: "message", Type: types.String(), Value: ""},
		{Name: "variant", Type: types.String(), Value: ""},
	}
	return types.Function(types.FunctionProperties{
		Inputs:  params,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the status module with its sole status.set member.
func NewSymbols() []*symbol.Symbol {
	member := &symbol.Symbol{
		Name:             setMemberName,
		Kind:             symbol.KindFunction,
		Exec:             symbol.ExecBoth,
		Type:             newSetSymbolType(),
		Trigger:          symbol.TriggerOnly,
		Doc:              memberDoc,
		AnalyzeArguments: analyzeStatusSetArguments,
	}
	mod := &symbol.Symbol{Name: moduleName, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(member)
	return []*symbol.Symbol{mod}
}

type Module struct {
	stat   *status.Service
	report taskreporter.Reporter
}

// ModuleConfig wires the Arc `status` module into a wazero runtime.
type ModuleConfig struct {
	Status   *status.Service
	Strings  *stlstrings.ProgramState
	Runtime  wazero.Runtime
	Reporter taskreporter.Reporter
}

func NewModule(ctx context.Context, cfg ModuleConfig) (*Module, error) {
	m := &Module{stat: cfg.Status, report: cfg.Reporter}
	if cfg.Runtime == nil {
		return m, nil
	}
	strings := cfg.Strings
	builder := cfg.Runtime.NewHostModuleBuilder(moduleName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH, msgH, variantH uint32) uint32 {
			keyOrName, kOK := strings.Get(keyOrNameH)
			msg, mOK := strings.Get(msgH)
			variant, vOK := strings.Get(variantH)
			if !kOK || !mOK || !vOK {
				m.report(ctx, status.VariantWarning,
					"status.set: invalid string handle from WASM runtime")
				return 0
			}
			return strings.Create(dispatchSet(ctx, m.stat, m.report, keyOrName, msg, variant))
		}).Export(setMemberName)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case setMemberName:
		inputs, err := node.ResolveInputs(cfg.State, cfg.Node)
		if err != nil {
			return nil, err
		}
		var in setInputs
		if err := setInputsSchema.Parse(inputs.ValidationMap(), &in); err != nil {
			return nil, errors.Wrap(err, "status.set inputs")
		}
		return &setNode{State: cfg.State, stat: m.stat, report: m.report, inputs: inputs}, nil
	default:
		return nil, query.ErrNotFound
	}
}

type setInputs struct {
	KeyOrName string `json:"key_or_name"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
}

var setInputsSchema = zyn.Object(map[string]zyn.Schema{
	"key_or_name": zyn.String(),
	"message":     zyn.String(),
	"variant":     zyn.String(),
})

type setNode struct {
	*node.State
	stat   *status.Service
	report taskreporter.Reporter
	inputs node.ResolvedInputs
}

func (s *setNode) Next(ctx node.Context) {
	if s.inputs.HasEdges() && !s.RefreshInputs() {
		return
	}
	var in setInputs
	if err := setInputsSchema.Parse(s.inputs.ValueMap(s.State), &in); err != nil {
		s.report(ctx, status.VariantWarning, fmt.Sprintf("status.set inputs: %v", err))
		return
	}
	key := dispatchSet(ctx, s.stat, s.report, in.KeyOrName, in.Message, in.Variant)
	*s.Output(0) = telem.NewSeriesV[string](key)
	*s.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	ctx.MarkChanged(0)
}

// dispatchSet upserts a status by key, by name, or creates a fresh row when
// neither matches. Failures use VariantWarning so the task continues running.
func dispatchSet(
	ctx context.Context,
	stat *status.Service,
	report taskreporter.Reporter,
	keyOrName, message, variantStr string,
) string {
	key, multi, err := stat.SetByKeyOrName(ctx, keyOrName, message, variantStr)
	if err != nil {
		report(ctx, status.VariantWarning, fmt.Sprintf("status.set: %v", err))
		return ""
	}
	if multi {
		report(ctx, status.VariantWarning, fmt.Sprintf(
			"status.set: multiple statuses named %q; updated first match (%s)",
			keyOrName, key,
		))
	}
	return key
}

const variantIndex = 2

// analyzeStatusSetArguments validates the variant argument across both call
// forms: it binds by name ("variant") or, when positional, by index.
func analyzeStatusSetArguments(diags *diagnostics.Diagnostics, args []symbol.Argument) {
	for _, arg := range args {
		if arg.Name == "variant" || (arg.Name == "" && arg.Index == variantIndex) {
			checkVariantLiteral(diags, arg.Expr)
			return
		}
	}
}

func checkVariantLiteral(diags *diagnostics.Diagnostics, expr parser.IExpressionContext) {
	lit := parser.GetLiteral(expr)
	if lit == nil {
		return
	}
	strNode := lit.STR_LITERAL()
	if strNode == nil {
		return
	}
	parsed, err := literal.ParseString(strNode.GetText(), types.String())
	if err != nil {
		return
	}
	value, ok := parsed.Value.(string)
	if !ok || status.Variant(value).IsValid() {
		return
	}
	diags.Add(diagnostics.Errorf(expr,
		"%q is not a valid status variant: [%s]",
		value, allowedVariantsList,
	))
}
