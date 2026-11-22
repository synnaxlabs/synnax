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
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
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
	// Channel is the calculated channel whose expression will be compiled.
	//
	// [REQUIRED]
	Channel channel.Channel
}

var _ config.Config[Config] = Config{}

func (c Config) Override(other Config) Config {
	c.Channel = other.Channel
	c.ChannelService = override.Nil(c.ChannelService, other.ChannelService)
	return c
}

func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	v.NonZero("channel.key", c.Channel.Key())
	v.NotNil("channel_service", c.ChannelService)
	return v.Error()
}

const (
	calculationKey = "calculation"
	writeKey       = "write"
)

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
		channel.NewConsistentResolver(cfg.ChannelService.NewArcSymbolResolver(snap)),
	)
}

func preProcess(
	ctx context.Context,
	cfg Config,
	resolver arc.SymbolResolver,
) (arc.Program, error) {
	ana := channel.NewCalculationAnalyzer(resolver, parser.Config{
		AllowDashedNames: !cfg.ChannelService.ShouldValidateNames(),
	})
	result, err := ana.Analyze(ctx, cfg.Channel)
	if err != nil {
		return arc.Program{}, err
	}
	dt := result.ExpressionReturnType
	if cfg.Channel.DataType != result.ChanDataType {
		dt = types.FromTelem(cfg.Channel.DataType)
	}
	fn := ir.Function{
		Key:     calculationKey,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: dt}},
		Body:    ir.Body{Raw: fmt.Sprintf("{%s}", cfg.Channel.Expression)},
	}
	g := arc.Graph{Functions: ir.Functions{fn}}
	return arc.CompileGraph(ctx, g, arc.NewRoot(resolver),
		arc.WithAllowDashedNames(!cfg.ChannelService.ShouldValidateNames()))
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
}

// Compile builds a full execution Module for the given calculated channel. The module
// includes WASM bytecode, an execution graph with operation nodes, and the channel
// dependencies the calculation reads from and writes to. It reads db as
// channel.CompileConsistent describes.
func Compile(ctx context.Context, db *gorp.DB, cfgs ...Config) (Module, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return Module{}, err
	}
	return channel.CompileConsistent(ctx, db, func(
		snap gorp.Snapshot,
		lookups *channel.Lookups,
	) (Module, error) {
		return compile(
			ctx,
			snap,
			cfg,
			cfg.ChannelService.NewCheckedResolver(snap, lookups),
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

// addOperations chains ops between the calculation and write nodes of g, or connects
// the two nodes directly when ops is empty.
func addOperations(g *arc.Graph, ops []channel.Operation) {
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
		addNode(g, key, string(o.Type), xmsgpack.EncodedJSON{"duration": o.Duration})
		if o.ResetChannel != 0 {
			resetKey := fmt.Sprintf("on_reset_%d", o.ResetChannel)
			addNode(g, resetKey, "on", xmsgpack.EncodedJSON{"channel": o.ResetChannel})
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
// fed by a node that reads the channel. A channel that the expression names in quoted
// form, at one of quoted, also gets an input named in that form, fed by the same node.
func addReads(
	ctx context.Context,
	g *arc.Graph,
	resolver arc.SymbolResolver,
	reads map[uint32]string,
	quoted []string,
) error {
	nodes := make(map[uint32]string, len(reads))
	for k, v := range reads {
		sym, err := resolver.Resolve(ctx, v)
		if err != nil {
			return err
		}
		addNode(g, sym.Name, "on", xmsgpack.EncodedJSON{"channel": k})
		addInput(g, sym.Name, sym.Name, *sym.Type.Elem)
		nodes[k] = sym.Name
	}
	for _, path := range quoted {
		sym, err := resolver.Resolve(ctx, path)
		if err != nil {
			return err
		}
		if node, ok := nodes[uint32(sym.ID)]; ok {
			addInput(g, node, parser.QuoteChannelName(path), *sym.Type.Elem)
		}
	}
	return nil
}

// addInput adds an input named name to the calculation function of g, fed by the
// output of node.
func addInput(g *arc.Graph, node, name string, t types.Type) {
	g.Functions[0].Inputs = append(
		g.Functions[0].Inputs,
		types.Param{Name: name, Type: t},
	)
	addEdge(
		g,
		ir.Handle{Node: node, Param: ir.DefaultOutputParam},
		ir.Handle{Node: calculationKey, Param: name},
	)
}

// quotedPaths returns each channel path that the parsed expression body names in
// quoted form, once, in the order of the expression. expression is its source.
func quotedPaths(expression string, body antlr.Tree) []string {
	if body == nil || !strings.ContainsRune(expression, '\'') {
		return nil
	}
	var (
		paths []string
		walk  func(t antlr.Tree)
	)
	walk = func(t antlr.Tree) {
		if term, ok := t.(antlr.TerminalNode); ok {
			token := term.GetSymbol()
			if token.GetTokenType() != parser.ArcLexerCHANNEL_NAME {
				return
			}
			if path := parser.TokenName(token).Head; !slices.Contains(paths, path) {
				paths = append(paths, path)
			}
			return
		}
		for i := range t.GetChildCount() {
			walk(t.GetChild(i))
		}
	}
	walk(body)
	return paths
}

func compile(
	ctx context.Context,
	snap gorp.Snapshot,
	cfg Config,
	resolver arc.SymbolResolver,
) (Module, error) {
	preProcessed, err := preProcess(ctx, cfg, resolver)
	if err != nil {
		return Module{}, err
	}
	calcFn := preProcessed.Functions[0]
	g := arc.Graph{
		Functions: ir.Functions{
			ir.Function{
				Key:     calculationKey,
				Outputs: calcFn.Outputs,
				Body:    calcFn.Body,
			},
		},
		Inputs: map[string]xmsgpack.EncodedJSON{},
	}
	addNode(&g, calculationKey, calculationKey, nil)
	addNode(&g, writeKey, writeKey, xmsgpack.EncodedJSON{"channel": cfg.Channel.Key()})
	cfg.Channel.Operations = lo.Filter(
		cfg.Channel.Operations,
		func(item channel.Operation, _ int) bool {
			return item.Type != "none"
		},
	)
	addOperations(&g, cfg.Channel.Operations)
	quoted := quotedPaths(cfg.Channel.Expression, calcFn.Body.AST)
	if err = addReads(ctx, &g, resolver, calcFn.Channels.Read, quoted); err != nil {
		return Module{}, err
	}
	program, err := arc.CompileGraph(ctx, g, arc.NewRoot(resolver),
		arc.WithAllowDashedNames(!cfg.ChannelService.ShouldValidateNames()))
	if err != nil {
		return Module{}, err
	}
	deps, err := runtime.NewDependencies(ctx, snap, cfg.ChannelService, program)
	if err != nil {
		return Module{}, err
	}
	return Module{Channel: cfg.Channel, Dependencies: deps, Program: program}, nil
}
