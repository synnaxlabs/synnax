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

	"github.com/synnaxlabs/alamos"
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
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
	"github.com/tetratelabs/wazero"
	"go.uber.org/zap"
)

const (
	setMemberName    = "set"
	deleteMemberName = "delete"
	moduleName       = "status"
)

// allowedVariantsList is the human-readable list used in compile-time and
// runtime diagnostics. Variant.IsValid is the source of truth for membership.
const allowedVariantsList = "success, info, warning, error, loading, disabled"

// setParams is the shared Inputs/Config list. Empty-string defaults mark the
// inputs optional so flow-form usage (Config-fulfilled, no edges) analyzes.
var setParams = types.Params{
	{Name: "key_or_name", Type: types.String(), Value: ""},
	{Name: "message", Type: types.String(), Value: ""},
	{Name: "variant", Type: types.String(), Value: ""},
}

var setSymbolProps = types.Function(types.FunctionProperties{
	Inputs:  setParams,
	Config:  setParams,
	Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
})

var deleteParams = types.Params{
	{Name: "key_or_name", Type: types.String(), Value: ""},
}

var deleteSymbolProps = types.Function(types.FunctionProperties{
	Inputs: deleteParams,
	Config: deleteParams,
})

var Symbols = []*symbol.Symbol{
	symbol.NewModule(moduleName,
		symbol.Symbol{
			Name:              setMemberName,
			Kind:              symbol.KindFunction,
			Exec:              symbol.ExecBoth,
			Type:              setSymbolProps,
			AnalyzeCall:       analyzeStatusSetCall,
			AnalyzeFlowConfig: analyzeStatusSetFlowConfig,
		},
		symbol.Symbol{
			Name: deleteMemberName,
			Kind: symbol.KindFunction,
			Exec: symbol.ExecBoth,
			Type: deleteSymbolProps,
		},
	),
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
	alamos.Instrumentation
}

func NewModule(ctx context.Context, cfg ModuleConfig) (*Module, error) {
	m := &Module{stat: cfg.Status, report: cfg.Reporter}
	if cfg.Runtime == nil {
		return m, nil
	}
	strings := cfg.Strings
	ins := cfg.Instrumentation
	builder := cfg.Runtime.NewHostModuleBuilder(moduleName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH, msgH, variantH uint32) uint32 {
			keyOrName, kOK := strings.Get(keyOrNameH)
			msg, mOK := strings.Get(msgH)
			variant, vOK := strings.Get(variantH)
			if !kOK || !mOK || !vOK {
				m.report(ctx, xstatus.VariantWarning,
					"status.set: invalid string handle from WASM runtime")
				return 0
			}
			return strings.Create(dispatchSet(ctx, m.stat, ins, m.report, keyOrName, msg, variant))
		}).Export(setMemberName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH uint32) {
			keyOrName, ok := strings.Get(keyOrNameH)
			if !ok {
				m.report(ctx, xstatus.VariantWarning,
					"status.delete: invalid string handle from WASM runtime")
				return
			}
			dispatchDelete(ctx, m.stat, ins, m.report, keyOrName)
		}).Export(deleteMemberName)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case setMemberName:
		var sc setConfig
		if err := setConfigSchema.Parse(cfg.Node.Config.ValueMap(), &sc); err != nil {
			return nil, errors.Wrap(err, "status.set config")
		}
		return &setNode{
			State:     cfg.State,
			stat:      m.stat,
			ins:       cfg.Instrumentation,
			report:    m.report,
			keyOrName: sc.KeyOrName,
			message:   sc.Message,
			variant:   sc.Variant,
		}, nil
	case deleteMemberName:
		var dc deleteConfig
		if err := deleteConfigSchema.Parse(cfg.Node.Config.ValueMap(), &dc); err != nil {
			return nil, errors.Wrap(err, "status.delete config")
		}
		return &deleteNode{
			State:     cfg.State,
			stat:      m.stat,
			ins:       cfg.Instrumentation,
			report:    m.report,
			keyOrName: dc.KeyOrName,
		}, nil
	default:
		return nil, query.ErrNotFound
	}
}

type setConfig struct {
	KeyOrName string `json:"key_or_name"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
}

var setConfigSchema = zyn.Object(map[string]zyn.Schema{
	"key_or_name": zyn.String(),
	"message":     zyn.String(),
	"variant":     zyn.String(),
})

type setNode struct {
	*node.State
	stat      *status.Service
	ins       alamos.Instrumentation
	report    taskreporter.Reporter
	keyOrName string
	message   string
	variant   string
}

func (s *setNode) Next(ctx node.Context) {
	key := dispatchSet(ctx, s.stat, s.ins, s.report, s.keyOrName, s.message, s.variant)
	*s.Output(0) = telem.NewSeriesV[string](key)
	*s.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	ctx.MarkChanged(0)
}

// dispatchSet upserts a status by key, by name, or creates a fresh row when
// neither matches. Failures use VariantWarning so the task continues running.
func dispatchSet(
	ctx context.Context,
	stat *status.Service,
	ins alamos.Instrumentation,
	report taskreporter.Reporter,
	keyOrName, message, variantStr string,
) string {
	key, multi, err := stat.SetByKeyOrName(ctx, keyOrName, message, variantStr)
	if err != nil {
		ins.L.Error("status.set failed",
			zap.String("key_or_name", keyOrName), zap.Error(err))
		report(ctx, xstatus.VariantWarning, fmt.Sprintf("status.set: %v", err))
		return ""
	}
	if multi {
		report(ctx, xstatus.VariantWarning, fmt.Sprintf(
			"status.set: multiple statuses named %q; updated first match (%s)",
			keyOrName, key,
		))
	}
	return key
}

const variantIndex = 2

func analyzeStatusSetCall(
	diags *diagnostics.Diagnostics,
	funcCall parser.IFunctionCallSuffixContext,
) {
	if al := funcCall.ArgumentList(); al != nil {
		if args := al.AllExpression(); len(args) > variantIndex {
			checkVariantLiteral(diags, args[variantIndex])
		}
	}
}

func analyzeStatusSetFlowConfig(
	diags *diagnostics.Diagnostics,
	config parser.IConfigValuesContext,
) {
	if config == nil {
		return
	}
	if named := config.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			if cv.IDENTIFIER().GetText() == "variant" {
				if e := cv.Expression(); e != nil {
					checkVariantLiteral(diags, e)
				}
				return
			}
		}
	} else if anon := config.AnonymousConfigValues(); anon != nil {
		if exprs := anon.AllExpression(); len(exprs) > variantIndex {
			checkVariantLiteral(diags, exprs[variantIndex])
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
	if !ok || xstatus.Variant(value).IsValid() {
		return
	}
	diags.Add(diagnostics.Errorf(expr,
		"%q is not a valid status variant: [%s]",
		value, allowedVariantsList,
	))
}

type deleteConfig struct {
	KeyOrName string `json:"key_or_name"`
}

var deleteConfigSchema = zyn.Object(map[string]zyn.Schema{
	"key_or_name": zyn.String(),
})

type deleteNode struct {
	*node.State
	stat      *status.Service
	ins       alamos.Instrumentation
	report    taskreporter.Reporter
	keyOrName string
}

func (s *deleteNode) Next(ctx node.Context) {
	dispatchDelete(ctx, s.stat, s.ins, s.report, s.keyOrName)
}

// dispatchDelete deletes a status by key or by name, reporting warnings on
// not-found, multi-match (deletes all), or failure.
func dispatchDelete(
	ctx context.Context,
	stat *status.Service,
	ins alamos.Instrumentation,
	report taskreporter.Reporter,
	keyOrName string,
) {
	count, err := stat.DeleteByKeyOrName(ctx, keyOrName)
	if err != nil {
		ins.L.Error("status.delete failed", zap.String("key_or_name", keyOrName), zap.Error(err))
		report(ctx, xstatus.VariantWarning, fmt.Sprintf("status.delete: %v", err))
		return
	}
	if count == 0 {
		report(ctx, xstatus.VariantWarning, fmt.Sprintf("status.delete: no status found %q", keyOrName))
		return
	}
	if count > 1 {
		report(ctx, xstatus.VariantWarning, fmt.Sprintf(
			"status.delete: multiple statuses named %q; deleted all (%d)",
			keyOrName, count,
		))
	}
}
