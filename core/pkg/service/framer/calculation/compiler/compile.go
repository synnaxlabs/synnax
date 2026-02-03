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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	SymbolResolver arc.SymbolResolver
	ChannelService *channel.Service
	Channel        channel.Channel
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Channel = other.Channel
	c.SymbolResolver = override.Nil(c.SymbolResolver, other.SymbolResolver)
	c.ChannelService = override.Nil(c.ChannelService, other.ChannelService)
	return c
}

// Validate implements config.Config.
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

func preProcess(ctx context.Context, cfg Config) (arc.Module, error) {
	outputDataType := types.FromTelem(cfg.Channel.DataType)
	fn := ir.Function{
		Key:     calculationKey,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: outputDataType}},
		Body:    ir.Body{Raw: fmt.Sprintf("{%s}", cfg.Channel.Expression)},
	}
	g := arc.Graph{Functions: ir.Functions{fn}}
	return arc.CompileGraph(ctx, g, arc.WithResolver(cfg.SymbolResolver))
}

type Module struct {
	StateConfig runtime.ExtendedStateConfig
	arc.Module
	Channel channel.Channel
}

func Compile(ctx context.Context, cfgs ...Config) (Module, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return Module{}, err
	}
	preProcessed, err := preProcess(ctx, cfg)
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
		g.Edges = []graph.Edge{
			{
				Source: ir.Handle{Node: calculationKey, Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: writeKey, Param: ir.DefaultInputParam},
			},
		}
	} else {
		for i, o := range cfg.Channel.Operations {
			key := fmt.Sprintf("op_%d", i)
			nextKey := fmt.Sprintf("op_%d", i)
			g.Nodes = append(g.Nodes, graph.Node{
				Key:  fmt.Sprintf("op_%d", i),
				Type: o.Type,
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

	mod, err := arc.CompileGraph(ctx, g, arc.WithResolver(cfg.SymbolResolver))
	if err != nil {
		return Module{}, err
	}
	stateCfg, err := runtime.NewStateConfig(ctx, cfg.ChannelService, mod)
	if err != nil {
		return Module{}, err
	}
	return Module{Channel: cfg.Channel, StateConfig: stateCfg, Module: mod}, nil
}
