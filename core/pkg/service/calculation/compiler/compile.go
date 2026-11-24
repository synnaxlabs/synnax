// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compiler

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl/join"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	calculation "github.com/synnaxlabs/synnax/pkg/service/calculation/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/channel/resolver"
	"github.com/synnaxlabs/x/config"
	xmsgpack "github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a calculation compilation.
type Config struct {
	// ChannelService builds the symbol resolver used to analyze the expression and
	// looks up channel metadata for the compiled state config.
	//
	// [REQUIRED]
	ChannelService *channel.Service
	// Channel is the output channel of Calculation.
	//
	// [REQUIRED]
	Channel channel.Channel
	// Calculation is the calculation to compile.
	//
	// [REQUIRED]
	Calculation calculation.Calculation
}

var _ config.Config[Config] = Config{}

func (c Config) Override(other Config) Config {
	c.Channel = other.Channel
	c.Calculation = other.Calculation
	c.ChannelService = override.Nil(c.ChannelService, other.ChannelService)
	return c
}

func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	v.NonZero("channel.key", c.Channel.Key())
	v.Ternaryf(
		"calculation.output",
		c.Calculation.Output != c.Channel.Key(),
		"must be the key of the channel",
	)
	v.NotNil("channel_service", c.ChannelService)
	return v.Error()
}

const (
	calculationKey = "calculation"
	writeKey       = "write"
	joinKey        = "join"
)

// JoinType is the node type of the join that pairs the reads of a calculation by
// sample time. It holds U+0000, which the channel name rule refuses, so no channel or
// Arc text can name it or be shadowed by it.
const JoinType = "\x00join"

// PreProcess compiles the channel's expression to discover channel references and infer
// output types without building the full execution graph. It looks up channels in snap.
func PreProcess(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg Config,
) (arc.Program, error) {
	return preProcess(
		ctx,
		cfg,
		resolver.NewConsistent(resolver.New(cfg.ChannelService, snap)),
	)
}

func preProcess(
	ctx context.Context,
	cfg Config,
	symbols arc.SymbolResolver,
) (arc.Program, error) {
	ana := NewAnalyzer(symbols, parser.Config{
		AllowDashedNames: resolver.AllowDashedNames(cfg.ChannelService),
	})
	result, err := ana.AnalyzeExpression(
		ctx,
		cfg.Channel,
		cfg.Calculation.Expression,
		HasDerivative(cfg.Calculation.Operations),
	)
	if err != nil {
		return arc.Program{}, err
	}
	fn := ir.Function{
		Key: calculationKey,
		Outputs: types.Params{
			{Name: ir.DefaultOutputParam, Type: outputType(cfg, result)},
		},
		Body: ir.Body{Raw: fmt.Sprintf("{%s}", cfg.Calculation.Expression)},
	}
	g := arc.Graph{Functions: ir.Functions{fn}}
	return arc.CompileGraph(ctx, g, arc.NewRoot(symbols),
		arc.WithAllowDashedNames(resolver.AllowDashedNames(cfg.ChannelService)))
}

// outputType returns the type of the output of the calculation of cfg, which result
// analyzes: the expression's type, or the output channel's when they differ.
func outputType(cfg Config, result AnalysisResult) types.Type {
	if cfg.Channel.DataType != result.ChanDataType {
		return types.FromTelem(cfg.Channel.DataType)
	}
	return result.ExpressionReturnType
}

// Check compiles the calculation of cfg, which result analyzes, against snap, as
// Compile does without its retries, and returns the compile error.
func Check(
	ctx context.Context,
	snap gorp.Snapshot,
	result AnalysisResult,
	cfgs ...Config,
) error {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return err
	}
	_, err = compileAnalyzed(
		ctx,
		snap,
		cfg,
		resolver.NewConsistent(resolver.New(cfg.ChannelService, snap)),
		result,
	)
	return err
}

// Module is the compiled output for a single calculated channel, ready for
// execution by the framer's calculator runtime.
type Module struct {
	// Dependencies describes the channels read and written by this calculation.
	Dependencies runtime.Dependencies
	// Program is the compiled Arc program containing WASM bytecode.
	arc.Program
	// Channel is the calculated channel this module was compiled for.
	Channel channel.Channel
	// Calculation is the calculation this module was compiled from.
	Calculation calculation.Calculation
}

// Compile builds a full execution Module for the given calculated channel. The module
// includes WASM bytecode, an execution graph with operation nodes, and the channel
// dependencies the calculation reads from and writes to. It reads db as
// resolver.CompileConsistent describes.
func Compile(ctx context.Context, db *gorp.DB, cfgs ...Config) (Module, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return Module{}, err
	}
	return resolver.CompileConsistent(ctx, db, func(
		snap gorp.Snapshot,
		lookups *resolver.Lookups,
	) (Module, error) {
		return compile(
			ctx,
			snap,
			cfg,
			resolver.NewChecked(cfg.ChannelService, snap, lookups),
		)
	})
}

func addNode(g *arc.Graph, key, typ string, inputs xmsgpack.EncodedJSON) {
	g.Nodes = append(g.Nodes, graph.Node{Key: key})
	if inputs == nil {
		inputs = xmsgpack.EncodedJSON{}
	}
	inputs["type"] = typ
	g.Inputs[key] = inputs
}

func addEdge(g *arc.Graph, source, target ir.Handle) {
	g.Edges = append(g.Edges, graph.Edge{Source: source, Target: target})
}

// HasDerivative reports whether ops holds a derivative.
func HasDerivative(ops []calculation.Operation) bool {
	return slices.ContainsFunc(ops, func(op calculation.Operation) bool {
		_, ok := op.Variant.(calculation.DerivativeOperation)
		return ok
	})
}

// operationNode returns the node type of op and its window, which is zero for an
// operation with none.
func operationNode(op calculation.Operation) (string, calculation.Window) {
	switch v := op.Variant.(type) {
	case calculation.MinOperation:
		return string(calculation.MinOperationType), v.Window
	case calculation.MaxOperation:
		return string(calculation.MaxOperationType), v.Window
	case calculation.AvgOperation:
		return string(calculation.AvgOperationType), v.Window
	case calculation.DerivativeOperation:
		return string(calculation.DerivativeOperationType), calculation.Window{}
	default:
		panic(fmt.Sprintf("operation of unknown variant %T", op.Variant))
	}
}

// addOperations chains ops between the calculation and write nodes of g, or connects
// the two nodes directly when ops is empty.
func addOperations(g *arc.Graph, ops []calculation.Operation) {
	if len(ops) == 0 {
		addEdge(
			g,
			ir.Handle{Node: calculationKey, Param: ir.DefaultOutputParam},
			ir.Handle{Node: writeKey, Param: ir.DefaultInputParam},
		)
		return
	}
	for i, o := range ops {
		key := fmt.Sprintf("op_%d", i)
		typ, w := operationNode(o)
		addNode(g, key, typ, xmsgpack.EncodedJSON{"duration": w.Duration})
		if w.Reset != nil {
			resetKey := fmt.Sprintf("on_reset_%d", *w.Reset)
			addNode(g, resetKey, "on", xmsgpack.EncodedJSON{"channel": *w.Reset})
			addEdge(
				g,
				ir.Handle{Node: resetKey, Param: ir.DefaultOutputParam},
				ir.Handle{Node: key, Param: "reset"},
			)
		}
		if i == 0 {
			addEdge(
				g,
				ir.Handle{Node: calculationKey, Param: ir.DefaultOutputParam},
				ir.Handle{Node: key, Param: ir.DefaultInputParam},
			)
		}
		nextKey := fmt.Sprintf("op_%d", i+1)
		if i == len(ops)-1 {
			nextKey = writeKey
		}
		addEdge(
			g,
			ir.Handle{Node: key, Param: ir.DefaultOutputParam},
			ir.Handle{Node: nextKey, Param: ir.DefaultInputParam},
		)
	}
}

// addReads adds an input to the calculation function of g for each channel in reads,
// fed through one join by a node that reads the channel. The join pairs the reads by
// sample time on the stamps of primary. It returns the symbol of the join, or nil when
// reads is empty.
func addReads(
	ctx context.Context,
	g *arc.Graph,
	symbols arc.SymbolResolver,
	reads map[uint32]string,
	primary *channel.Key,
) (*symbol.Symbol, error) {
	if len(reads) == 0 {
		return nil, nil
	}
	var (
		props      types.FunctionProperties
		primaryIdx int
	)
	// The inputs take the order of their keys, so a tie between inputs breaks the same
	// way on every compile.
	for i, k := range slices.Sorted(maps.Keys(reads)) {
		sym, err := symbols.Resolve(ctx, reads[k])
		if err != nil {
			return nil, err
		}
		elem := *sym.Type.Elem
		g.Functions[0].Inputs = append(
			g.Functions[0].Inputs,
			types.Param{Name: sym.Name, Type: elem},
		)
		addNode(g, sym.Name, "on", xmsgpack.EncodedJSON{"channel": k})
		in, out := join.InputParam(i), join.OutputParam(i)
		props.Inputs = append(props.Inputs, types.Param{Name: in, Type: elem})
		props.Outputs = append(props.Outputs, types.Param{Name: out, Type: elem})
		addEdge(
			g,
			ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam},
			ir.Handle{Node: joinKey, Param: in},
		)
		addEdge(
			g,
			ir.Handle{Node: joinKey, Param: out},
			ir.Handle{Node: calculationKey, Param: sym.Name},
		)
		if primary != nil && channel.Key(k) == *primary {
			primaryIdx = i
		}
	}
	props.Inputs = append(props.Inputs, types.Param{
		Name:  join.PrimaryParam,
		Type:  types.I64(),
		Value: int64(primaryIdx),
	})
	addNode(g, joinKey, JoinType, nil)
	return &symbol.Symbol{
		Name: JoinType,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(props),
	}, nil
}

func compile(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg Config,
	symbols arc.SymbolResolver,
) (Module, error) {
	result, err := NewAnalyzer(symbols, parser.Config{
		AllowDashedNames: resolver.AllowDashedNames(cfg.ChannelService),
	}).AnalyzeExpression(
		ctx,
		cfg.Channel,
		cfg.Calculation.Expression,
		HasDerivative(cfg.Calculation.Operations),
	)
	if err != nil {
		return Module{}, err
	}
	return compileAnalyzed(ctx, snap, cfg, symbols, result)
}

// compileAnalyzed compiles the calculation of cfg, which result analyzes.
func compileAnalyzed(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg Config,
	symbols arc.SymbolResolver,
	result AnalysisResult,
) (Module, error) {
	g := arc.Graph{
		Functions: ir.Functions{
			ir.Function{
				Key: calculationKey,
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: outputType(cfg, result)},
				},
				Body: ir.Body{Raw: fmt.Sprintf("{%s}", cfg.Calculation.Expression)},
			},
		},
		Inputs: map[string]xmsgpack.EncodedJSON{},
	}
	addNode(&g, calculationKey, calculationKey, nil)
	addNode(&g, writeKey, writeKey, xmsgpack.EncodedJSON{"channel": cfg.Channel.Key()})
	addOperations(&g, cfg.Calculation.Operations)
	joinSym, err := addReads(ctx, &g, symbols, result.Reads, cfg.Calculation.Primary)
	if err != nil {
		return Module{}, err
	}
	var extras []*symbol.Symbol
	if joinSym != nil {
		extras = append(extras, joinSym)
	}
	program, err := arc.CompileGraph(ctx, g, arc.NewRoot(symbols, extras...),
		arc.WithAllowDashedNames(resolver.AllowDashedNames(cfg.ChannelService)))
	if err != nil {
		return Module{}, err
	}
	deps, err := runtime.NewDependencies(ctx, snap, cfg.ChannelService, program)
	if err != nil {
		return Module{}, err
	}
	return Module{
		Channel:      cfg.Channel,
		Calculation:  cfg.Calculation,
		Dependencies: deps,
		Program:      program,
	}, nil
}
