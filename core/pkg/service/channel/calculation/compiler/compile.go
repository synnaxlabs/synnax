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

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config configures a calculation compilation.
type Config struct {
	// SymbolResolver resolves channel names and STL symbols during compilation.
	SymbolResolver arc.SymbolResolver
	// ChannelService is used to look up channel metadata for the state config.
	ChannelService *channel.Service
	// Channel is the calculated channel whose expression will be compiled.
	Channel channel.Channel
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

func (c Config) Override(other Config) Config {
	c.Channel = other.Channel
	c.SymbolResolver = override.Nil(c.SymbolResolver, other.SymbolResolver)
	c.ChannelService = override.Nil(c.ChannelService, other.ChannelService)
	return c
}

func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "resolver", c.SymbolResolver)
	validate.NonZero(v, "channel.key", c.Channel.Key())
	validate.NotNil(v, "channel_service", c.ChannelService)
	return v.Error()
}

const (
	calculationKey = "calculation"
	writeKey       = "write"
)

// PreProcess compiles the channel's expression to discover channel references
// and infer output types without building the full execution graph.
func PreProcess(ctx context.Context, cfg Config) (arc.Program, error) {
	ana := analyzer.New(cfg.SymbolResolver)
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
	return arc.CompileGraph(ctx, g, arc.WithResolver(cfg.SymbolResolver))
}

// Module is the compiled output for a single calculated channel, ready for
// execution by the framer's calculator runtime.
type Module struct {
	// StateConfig describes the channels read and written by this calculation.
	StateConfig runtime.ExtendedStateConfig
	// Program is the compiled Arc program containing WASM bytecode.
	arc.Program
	// Channel is the calculated channel this module was compiled for.
	Channel channel.Channel
}

// Compile builds a full execution Module for the given calculated channel. The
// module includes WASM bytecode, an execution graph with operation nodes, and a
// state config mapping channel keys to read/write slots.
func Compile(ctx context.Context, cfgs ...Config) (Module, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return Module{}, err
	}
	preProcessed, err := PreProcess(ctx, cfg)
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
		Nodes: []graph.Node{
			{Key: calculationKey, Type: calculationKey},
			{
				Key:    writeKey,
				Type:   writeKey,
				Config: map[string]any{"channel": cfg.Channel.Key()},
			},
		},
	}
	cfg.Channel.Operations = lo.Filter(cfg.Channel.Operations, func(item channel.Operation, _ int) bool {
		return item.Type != "none"
	})
	if len(cfg.Channel.Operations) == 0 {
		g.Edges = []graph.Edge{{
			Source: ir.Handle{Node: calculationKey, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: writeKey, Param: ir.DefaultInputParam},
		}}
	} else {
		for i, o := range cfg.Channel.Operations {
			key := fmt.Sprintf("op_%d", i)
			nextKey := fmt.Sprintf("op_%d", i+1)
			g.Nodes = append(g.Nodes, graph.Node{
				Key:  fmt.Sprintf("op_%d", i),
				Type: string(o.Type),
				Config: map[string]any{
					"duration": o.Duration,
				},
			})
			if o.ResetChannel != 0 {
				resetKey := fmt.Sprintf("on_reset_%d", o.ResetChannel)
				g.Nodes = append(g.Nodes, graph.Node{
					Key:  resetKey,
					Type: "on",
					Config: map[string]any{
						"channel": o.ResetChannel,
					},
				})
				g.Edges = append(g.Edges, graph.Edge{
					Source: ir.Handle{Node: resetKey, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: key, Param: "reset"},
				})
			}
			if i == 0 {
				g.Edges = append(g.Edges, graph.Edge{
					Source: ir.Handle{Node: calculationKey, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: key, Param: ir.DefaultInputParam},
				})
			}
			if i == len(cfg.Channel.Operations)-1 {
				g.Edges = append(g.Edges, graph.Edge{
					Source: ir.Handle{Node: key, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: writeKey, Param: ir.DefaultInputParam},
				})
			} else {
				g.Edges = append(g.Edges, graph.Edge{
					Source: ir.Handle{Node: key, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: nextKey, Param: ir.DefaultInputParam},
				})
			}
		}
	}

	for k, v := range calcFn.Channels.Read {
		sym, err := cfg.SymbolResolver.Resolve(ctx, v)
		if err != nil {
			return Module{}, err
		}
		g.Functions[0].Inputs = append(
			g.Functions[0].Inputs,
			types.Param{Name: sym.Name, Type: *sym.Type.Elem},
		)
		g.Nodes = append(g.Nodes, graph.Node{
			Key:    sym.Name,
			Type:   "on",
			Config: map[string]any{"channel": k},
		})
		g.Edges = append(g.Edges, graph.Edge{
			Source: ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: calculationKey, Param: sym.Name},
		})
	}

	program, err := arc.CompileGraph(ctx, g, arc.WithResolver(cfg.SymbolResolver))
	if err != nil {
		return Module{}, err
	}
	stateCfg, err := runtime.NewStateConfig(ctx, cfg.ChannelService, program)
	if err != nil {
		return Module{}, err
	}
	return Module{Channel: cfg.Channel, StateConfig: stateCfg, Program: program}, nil
}
