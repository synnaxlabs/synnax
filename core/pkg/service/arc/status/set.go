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
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/zyn"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

const (
	bareSymbolName      = "set_status"
	qualifiedMemberName = "set"
	moduleName          = "status"
)

// bareSymbolProps describes the deprecated set_status form. Preserved verbatim
// for backwards compatibility; the qualified status.set form supersedes it.
var bareSymbolProps = types.Function(types.FunctionProperties{
	Config: types.Params{
		{Name: "status_key", Type: types.String()},
		{Name: "variant", Type: types.String()},
		{Name: "message", Type: types.String()},
		{Name: "name", Type: types.String(), Value: ""},
	},
	Inputs: types.Params{
		{Name: ir.DefaultOutputParam, Type: types.U8()},
	},
})

// qualifiedParams is the shared parameter list for the ExecBoth form. Inputs
// and Config reference it directly to satisfy the dual-shape mirror contract.
var qualifiedParams = types.Params{
	{Name: "key_or_name", Type: types.String()},
	{Name: "message", Type: types.String()},
	{Name: "variant", Type: types.String()},
}

var qualifiedSymbolProps = types.Function(types.FunctionProperties{
	Inputs:  qualifiedParams,
	Config:  qualifiedParams,
	Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
})

var (
	bareResolver = symbol.MapResolver{
		bareSymbolName: {
			Name:       bareSymbolName,
			Kind:       symbol.KindFunction,
			Exec:       symbol.ExecFlow,
			Type:       bareSymbolProps,
			Deprecated: "status.set",
		},
	}
	moduleResolver = &symbol.ModuleResolver{
		Name: moduleName,
		Members: symbol.MapResolver{
			qualifiedMemberName: {
				Name:              qualifiedMemberName,
				Kind:              symbol.KindFunction,
				Exec:              symbol.ExecBoth,
				Type:              qualifiedSymbolProps,
				AnalyzeCall:       analyzeStatusSetCall,
				AnalyzeFlowConfig: analyzeStatusSetFlowConfig,
			},
		},
	}
	SymbolResolver = symbol.CompoundResolver{bareResolver, moduleResolver}
)

type Module struct {
	stat    *status.Service
	strings *stlstrings.ProgramState
	memory  api.Memory
	ins     alamos.Instrumentation
}

func (m *Module) SetMemory(memory api.Memory) { m.memory = memory }

func NewModule(
	ctx context.Context,
	stat *status.Service,
	strings *stlstrings.ProgramState,
	rat wazero.Runtime,
	memory api.Memory,
	ins alamos.Instrumentation,
) (*Module, error) {
	m := &Module{stat: stat, strings: strings, memory: memory, ins: ins}
	if rat == nil {
		return m, nil
	}
	builder := rat.NewHostModuleBuilder(moduleName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyOrNameH, msgH, variantH uint32) uint32 {
			keyOrName, _ := m.strings.Get(keyOrNameH)
			msg, _ := m.strings.Get(msgH)
			variant, _ := m.strings.Get(variantH)
			return m.strings.Create(dispatchSet(ctx, m.stat, m.ins, keyOrName, msg, variant))
		}).Export(qualifiedMemberName)
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
	case bareSymbolName:
		var nodeCfg legacySetNodeConfig
		if err := legacySetNodeConfigSchema.Parse(cfg.Node.Config.ValueMap(), &nodeCfg); err != nil {
			return nil, err
		}
		var stat status.Status[any]
		if err := m.stat.NewRetrieve().
			Where(status.MatchKeys[any](nodeCfg.StatusKey)).
			Entry(&stat).
			Exec(ctx, nil); errors.Skip(err, query.ErrNotFound) != nil {
			return nil, err
		}
		stat.Key = nodeCfg.StatusKey
		stat.Name = nodeCfg.Name
		stat.Message = nodeCfg.Message
		stat.Variant = xstatus.Variant(nodeCfg.Variant)
		return &setNode{ins: cfg.Instrumentation, stat: stat, statusSvc: m.stat}, nil
	case qualifiedMemberName:
		vm := cfg.Node.Config.ValueMap()
		return &qualifiedSetNode{
			State:     cfg.State,
			stat:      m.stat,
			ins:       cfg.Instrumentation,
			keyOrName: vm["key_or_name"].(string),
			message:   vm["message"].(string),
			variant:   vm["variant"].(string),
		}, nil
	default:
		return nil, query.ErrNotFound
	}
}

type legacySetNodeConfig struct {
	StatusKey string `json:"status_key"`
	Message   string `json:"message"`
	Variant   string `json:"variant"`
	Name      string `json:"name"`
}

var legacySetNodeConfigSchema = zyn.Object(map[string]zyn.Schema{
	"status_key": zyn.String(),
	"message":    zyn.String(),
	"variant":    zyn.String(),
	"name":       zyn.String().Optional(),
})

type setNode struct {
	statusSvc *status.Service
	ins       alamos.Instrumentation
	stat      status.Status[any]
}

func (s *setNode) Init(node.Context) {}

func (s *setNode) Reset() {}

func (s *setNode) IsOutputTruthy(int) bool { return false }

func (s *setNode) Next(ctx node.Context) {
	s.stat.Time = telem.Now()
	if err := s.statusSvc.NewWriter(nil).Set(ctx, &s.stat); err != nil {
		s.ins.L.Error("error setting status", zap.Error(err))
	}
}

type qualifiedSetNode struct {
	*node.State
	stat      *status.Service
	ins       alamos.Instrumentation
	keyOrName string
	message   string
	variant   string
}

func (s *qualifiedSetNode) Next(ctx node.Context) {
	key := dispatchSet(ctx, s.stat, s.ins, s.keyOrName, s.message, s.variant)
	*s.Output(0) = telem.NewSeriesV[string](key)
	*s.OutputTime(0) = telem.NewSeriesV[telem.TimeStamp](telem.Now())
	ctx.MarkChanged(0)
}

// dispatchSet upserts a status by key (UUID) or by name. Returns the resulting
// key, or "" after logging via ins on failure.
func dispatchSet(
	ctx context.Context,
	stat *status.Service,
	ins alamos.Instrumentation,
	keyOrName, message, variantStr string,
) string {
	if !slices.Contains(allowedVariants, variantStr) {
		ins.L.Error("invalid status variant",
			zap.String("variant", variantStr),
			zap.Strings("allowed", allowedVariants))
		return ""
	}
	overlay := func(s *status.Status[any]) error {
		s.Message = message
		s.Variant = xstatus.Variant(variantStr)
		s.Time = telem.Now()
		return nil
	}
	var (
		key string
		err error
	)
	if _, perr := uuid.Parse(keyOrName); perr == nil {
		key = keyOrName
		err = stat.NewWriter(nil).Update(ctx, keyOrName, overlay)
	} else {
		err = stat.WithTx(ctx, func(tx gorp.Tx) error {
			var ierr error
			key, ierr = stat.NewWriter(tx).UpsertByName(ctx, keyOrName, overlay)
			return ierr
		})
	}
	if err != nil {
		ins.L.Error("status.set failed",
			zap.String("key_or_name", keyOrName), zap.Error(err))
		return ""
	}
	return key
}

var allowedVariants = []string{
	string(xstatus.VariantSuccess),
	string(xstatus.VariantInfo),
	string(xstatus.VariantWarning),
	string(xstatus.VariantError),
	string(xstatus.VariantLoading),
	string(xstatus.VariantDisabled),
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
	if !ok || slices.Contains(allowedVariants, value) {
		return
	}
	diags.Add(diagnostics.Errorf(expr,
		"%q is not a valid status variant: [%s]",
		value, strings.Join(allowedVariants, ", "),
	))
}
