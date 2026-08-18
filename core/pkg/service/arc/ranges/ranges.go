// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranges

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/service/arc/internal/taskreporter"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/synnaxlabs/x/zyn"
	"github.com/tetratelabs/wazero"
)

const (
	createMemberName = "create"
	endMemberName    = "end"
	moduleName       = "ranges"
)

// createDoc is the LSP hover body for ranges.create. The renderer prepends the title
// from the symbol name and kind, so it is omitted here.
var createDoc = doc.New(
	doc.Paragraph("Creates a range that starts now and stays \"In Progress\"."),
	doc.Divider(),
	doc.Code("arc", "range_key := trigger -> ranges.create{name=\"Terminal Count\"}"),
	doc.Divider(),
	doc.Paragraph("Outputs the range key; close the range later with ranges.end."),
)

// endDoc is the LSP hover body for ranges.end.
var endDoc = doc.New(
	doc.Paragraph("Sets the end bound of an existing range."),
	doc.Divider(),
	doc.Code("arc", "trigger -> ranges.end{key=range_key}"),
	doc.Divider(),
	doc.Paragraph("Sets the end to now. Outputs the range key."),
)

// moduleDoc is the LSP hover body for the ranges module itself.
var moduleDoc = doc.New(
	doc.Paragraph("Creates and manages ranges."),
)

// newCreateSymbolType returns a fresh ranges.create function type per call so analysis
// never mutates a shared symbol. Inputs with a default are optional.
func newCreateSymbolType() types.Type {
	params := types.Params{
		{Name: "name", Type: types.String()},
		{Name: "parent", Type: types.String(), Value: ""},
		{Name: "color", Type: types.String(), Value: ""},
	}
	return types.Function(types.FunctionProperties{
		Inputs:  params,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	})
}

// newEndSymbolType returns a fresh ranges.end function type per call so analysis never
// mutates a shared symbol.
func newEndSymbolType() types.Type {
	params := types.Params{
		{Name: "key", Type: types.String()},
	}
	return types.Function(types.FunctionProperties{
		Inputs:  params,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package contributes:
// the ranges module with its create and end members.
func NewSymbols() []*symbol.Symbol {
	createMember := &symbol.Symbol{
		Name:             createMemberName,
		Kind:             symbol.KindFunction,
		Exec:             symbol.ExecBoth,
		Type:             newCreateSymbolType(),
		Trigger:          symbol.TriggerOnly,
		Doc:              createDoc,
		AnalyzeArguments: analyzeCreateArguments,
	}
	endMember := &symbol.Symbol{
		Name:    endMemberName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecBoth,
		Type:    newEndSymbolType(),
		Trigger: symbol.TriggerOnly,
		Doc:     endDoc,
	}
	mod := &symbol.Symbol{Name: moduleName, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(createMember)
	mod.AddChild(endMember)
	return []*symbol.Symbol{mod}
}

type module struct {
	rng    *ranger.Service
	report taskreporter.Reporter
}

// ModuleConfig wires the Arc ranges module into a wazero runtime.
type ModuleConfig struct {
	Ranger   *ranger.Service
	Strings  *strings.ProgramState
	Runtime  wazero.Runtime
	Reporter taskreporter.Reporter
}

func NewModule(ctx context.Context, cfg ModuleConfig) (node.Factory, error) {
	v := validate.New("arc.ranges")
	validate.NotNil(v, "ranger", cfg.Ranger)
	validate.NotNil(v, "reporter", cfg.Reporter)
	if cfg.Runtime != nil {
		validate.NotNil(v, "strings", cfg.Strings)
	}
	if err := v.Error(); err != nil {
		return nil, err
	}
	m := &module{rng: cfg.Ranger, report: cfg.Reporter}
	if cfg.Runtime == nil {
		return m, nil
	}
	heap := cfg.Strings
	builder := cfg.Runtime.NewHostModuleBuilder(moduleName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, nameH, parentH, colorH uint32) uint32 {
			name, nOK := heap.Get(nameH)
			parent, pOK := heap.Get(parentH)
			colorHex, cOK := heap.Get(colorH)
			if !nOK || !cOK || !pOK {
				m.report(ctx, status.VariantWarning,
					"ranges.create: invalid string handle from WASM runtime")
				return 0
			}
			return heap.Create(
				dispatchCreate(ctx, m.rng, m.report, name, parent, colorHex),
			)
		}).Export(createMemberName)
	builder = builder.NewFunctionBuilder().
		WithFunc(func(ctx context.Context, keyH uint32) uint32 {
			key, ok := heap.Get(keyH)
			if !ok {
				m.report(
					ctx,
					status.VariantWarning,
					"ranges.end: invalid string handle from WASM runtime",
				)
				return 0
			}
			return heap.Create(dispatchEnd(ctx, m.rng, m.report, key))
		}).Export(endMemberName)
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *module) Create(_ context.Context, cfg node.Config) (node.Node, error) {
	switch cfg.Node.Type {
	case createMemberName:
		if err := createSchema.Validate(cfg.Node.Inputs.ValueMap()); err != nil {
			return nil, errors.Wrap(err, "ranges.create inputs")
		}
		return &createNode{State: cfg.State, rng: m.rng, report: m.report}, nil

	case endMemberName:
		if err := endSchema.Validate(cfg.Node.Inputs.ValueMap()); err != nil {
			return nil, errors.Wrap(err, "ranges.end inputs")
		}
		return &endNode{State: cfg.State, rng: m.rng, report: m.report}, nil
	default:
		return nil, query.ErrNotFound
	}
}

var createSchema = zyn.Object(map[string]zyn.Schema{
	"name":   zyn.String(),
	"parent": zyn.String(),
	"color":  zyn.String(),
})

type createNode struct {
	*node.State
	rng    *ranger.Service
	report taskreporter.Reporter
}

func (n *createNode) Next(ctx node.Context) {
	key := dispatchCreate(ctx, n.rng, n.report,
		n.StringInput("name"), n.StringInput("parent"), n.StringInput("color"))
	*n.Output(0) = telem.NewSeriesV(key)
	*n.OutputTime(0) = telem.NewSeriesV(telem.Now())
	ctx.MarkChanged(0)
}

// dispatchCreate creates an open range that starts now, parsing the color and parent
// key, reporting failures as warnings so the task keeps running. On any failure no
// range is created.
func dispatchCreate(
	ctx context.Context,
	rng *ranger.Service,
	report taskreporter.Reporter,
	name, parent, colorHex string,
) string {
	var c *color.Color
	if colorHex != "" {
		parsed, err := color.FromCSS(colorHex)
		if err != nil {
			report(ctx, status.VariantWarning,
				fmt.Sprintf("ranges.create: invalid color %q: %v", colorHex, err))
			return ""
		}
		c = &parsed
	}
	r := ranger.Range{
		Name:      name,
		TimeRange: telem.TimeRange{Start: telem.Now(), End: telem.TimeStampMax},
		Color:     c,
	}
	if parent != "" {
		uid, err := uuid.Parse(parent)
		if err != nil {
			report(ctx, status.VariantWarning,
				fmt.Sprintf("ranges.create: invalid parent key %q", parent))
			return ""
		}
		r.Parent = &ranger.Range{Key: uid}
	}
	if err := rng.NewWriter(nil).Create(ctx, &r); err != nil {
		report(ctx, status.VariantWarning, fmt.Sprintf("ranges.create: %v", err))
		return ""
	}
	return r.Key.String()
}

var endSchema = zyn.Object(map[string]zyn.Schema{
	"key": zyn.String(),
})

type endNode struct {
	*node.State
	rng    *ranger.Service
	report taskreporter.Reporter
}

func (n *endNode) Next(ctx node.Context) {
	key := dispatchEnd(ctx, n.rng, n.report, n.StringInput("key"))
	*n.Output(0) = telem.NewSeriesV(key)
	*n.OutputTime(0) = telem.NewSeriesV(telem.Now())
	ctx.MarkChanged(0)
}

// dispatchEnd sets the end bound on the range identified by key to now, reporting
// failures as warnings so the task keeps running.
func dispatchEnd(
	ctx context.Context,
	rng *ranger.Service,
	report taskreporter.Reporter,
	key string,
) string {
	uid, err := uuid.Parse(key)
	if err != nil {
		report(
			ctx,
			status.VariantWarning,
			fmt.Sprintf("ranges.end: invalid range key %q", key),
		)
		return ""
	}
	if err := rng.NewWriter(nil).SetEnd(ctx, uid, telem.Now()); err != nil {
		report(ctx, status.VariantWarning, fmt.Sprintf("ranges.end: %v", err))
		return ""
	}
	return uid.String()
}

const colorIndex = 2

// analyzeCreateArguments validates the color argument across both call forms: it binds
// by name ("color") or, when positional, by index.
func analyzeCreateArguments(diags *diagnostics.Diagnostics, args []symbol.Argument) {
	for _, arg := range args {
		if arg.Name == "color" || (arg.Name == "" && arg.Index == colorIndex) {
			checkColorLiteral(diags, arg.Expr)
			return
		}
	}
}

func checkColorLiteral(diags *diagnostics.Diagnostics, expr parser.IExpressionContext) {
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
	if !ok {
		return
	}
	// An empty color is valid and means "no color", matching dispatchCreate's runtime
	// behavior; only a non-empty literal must be a parseable CSS color.
	if value == "" {
		return
	}
	if _, err := color.FromCSS(value); err != nil {
		diags.Add(diagnostics.Errorf(expr, "%v", err))
	}
}
