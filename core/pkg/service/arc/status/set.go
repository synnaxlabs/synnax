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
	"slices"
	"strings"

	"github.com/synnaxlabs/alamos"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
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
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

const (
	setMemberName = "set"
	moduleName    = "status"
)

// setParams is the shared parameter list for the ExecBoth form. Inputs and
// Config reference it directly to satisfy the dual-shape mirror contract.
var setParams = types.Params{
	{Name: "key_or_name", Type: types.String()},
	{Name: "message", Type: types.String()},
	{Name: "variant", Type: types.String()},
}

var setSymbolProps = types.Function(types.FunctionProperties{
	Inputs:  setParams,
	Config:  setParams,
	Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
})

var SymbolResolver = &symbol.ModuleResolver{
	Name: moduleName,
	Members: symbol.MapResolver{
		setMemberName: {
			Name:              setMemberName,
			Kind:              symbol.KindFunction,
			Exec:              symbol.ExecBoth,
			Type:              setSymbolProps,
			AnalyzeCall:       analyzeStatusSetCall,
			AnalyzeFlowConfig: analyzeStatusSetFlowConfig,
		},
		deleteMemberName: deleteResolverEntry,
	},
}

type Module struct {
	stat    *status.Service
	strings *stlstrings.ProgramState
	memory  api.Memory
	ins     alamos.Instrumentation
	report  taskreporter.Reporter
}

func (m *Module) SetMemory(memory api.Memory) { m.memory = memory }

func NewModule(
	ctx context.Context,
	stat *status.Service,
	strings *stlstrings.ProgramState,
	rat wazero.Runtime,
	memory api.Memory,
	ins alamos.Instrumentation,
	report taskreporter.Reporter,
) (*Module, error) {
	m := &Module{stat: stat, strings: strings, memory: memory, ins: ins, report: report}
	if rat == nil {
		return m, nil
	}
	builder := rat.NewHostModuleBuilder(moduleName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH, msgH, variantH uint32) uint32 {
			keyOrName, _ := m.strings.Get(keyOrNameH)
			msg, _ := m.strings.Get(msgH)
			variant, _ := m.strings.Get(variantH)
			return m.strings.Create(dispatchSet(ctx, m.stat, m.ins, m.report, keyOrName, msg, variant))
		}).Export(setMemberName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH uint32) uint32 {
			keyOrName, _ := m.strings.Get(keyOrNameH)
			if dispatchDelete(ctx, m.stat, m.ins, m.report, keyOrName) {
				return 1
			}
			return 0
		}).Export(deleteMemberName)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Module) Resolve(ctx context.Context, name string) (symbol.Symbol, error) {
	return SymbolResolver.Resolve(ctx, name)
}

func (m *Module) Search(ctx context.Context, term string) ([]symbol.Symbol, error) {
	return SymbolResolver.Search(ctx, term)
}

func (m *Module) ModuleName() string { return moduleName }

func (m *Module) Create(ctx context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case setMemberName:
		vm := cfg.Node.Config.ValueMap()
		return &setNode{
			State:     cfg.State,
			stat:      m.stat,
			ins:       cfg.Instrumentation,
			report:    m.report,
			keyOrName: vm["key_or_name"].(string),
			message:   vm["message"].(string),
			variant:   vm["variant"].(string),
		}, nil
	case deleteMemberName:
		vm := cfg.Node.Config.ValueMap()
		return &deleteNode{
			State:     cfg.State,
			stat:      m.stat,
			ins:       cfg.Instrumentation,
			report:    m.report,
			keyOrName: vm["key_or_name"].(string),
		}, nil
	default:
		return nil, query.ErrNotFound
	}
}

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

// dispatchSet upserts a status by key (UUID) or by name. Failures use
// VariantWarning so the task continues running.
func dispatchSet(
	ctx context.Context,
	stat *status.Service,
	ins alamos.Instrumentation,
	report taskreporter.Reporter,
	keyOrName, message, variantStr string,
) string {
	key, multi, err := stat.SetByKeyOrName(ctx, keyOrName, message, variantStr)
	if errors.Is(err, status.ErrInvalidVariant) {
		ins.L.Error("invalid status variant",
			zap.String("variant", variantStr),
			zap.Strings("allowed", status.AllowedVariants))
		report(ctx, xstatus.VariantWarning,
			fmt.Sprintf("status.set: %q is not a valid variant", variantStr))
		return ""
	}
	if err == nil {
		if multi {
			report(ctx, xstatus.VariantWarning,
				fmt.Sprintf("status.set: multiple statuses named %q; updated first match (%s)", keyOrName, key))
		}
		return key
	}
	ins.L.Error("status.set failed",
		zap.String("key_or_name", keyOrName), zap.Error(err))
	report(ctx, xstatus.VariantWarning,
		fmt.Sprintf("status.set: %v", err))
	return ""
}

const variantIndex = 2

func analyzeStatusSetCall(ctx any, funcCall parser.IFunctionCallSuffixContext) {
	c, ok := ctx.(acontext.Context[parser.IPostfixExpressionContext])
	if !ok {
		return
	}
	if al := funcCall.ArgumentList(); al != nil {
		if args := al.AllExpression(); len(args) > variantIndex {
			checkVariantLiteral(c.Diagnostics, args[variantIndex])
		}
	}
}

func analyzeStatusSetFlowConfig(ctx any, config parser.IConfigValuesContext) {
	c, ok := ctx.(acontext.Context[parser.IFunctionContext])
	if !ok || config == nil {
		return
	}
	if named := config.NamedConfigValues(); named != nil {
		for _, cv := range named.AllNamedConfigValue() {
			if cv.IDENTIFIER().GetText() == "variant" {
				if e := cv.Expression(); e != nil {
					checkVariantLiteral(c.Diagnostics, e)
				}
				return
			}
		}
	} else if anon := config.AnonymousConfigValues(); anon != nil {
		if exprs := anon.AllExpression(); len(exprs) > variantIndex {
			checkVariantLiteral(c.Diagnostics, exprs[variantIndex])
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
	if !ok || slices.Contains(status.AllowedVariants, value) {
		return
	}
	diags.Add(diagnostics.Errorf(expr,
		"%q is not a valid status variant: [%s]",
		value, strings.Join(status.AllowedVariants, ", "),
	))
}
