// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func compile(ctx context.Context, cfg CalculatorConfig) (arc.Module, error) {
	g := arc.Graph{
		Functions: ir.Functions{
			{
				Key: "calculation",
				Outputs: types.Params{
					{Name: ir.DefaultOutputParam, Type: types.FromTelem(cfg.Channel.DataType)},
				},
				Body: ir.Body{Raw: fmt.Sprintf("{%s}", cfg.Channel.Expression)},
			},
		},
		Nodes: []graph.Node{},
	}
	preProcessed, err := arc.CompileGraph(ctx, g, arc.WithResolver(cfg.Resolver))
	if err != nil {
		return arc.Module{}, err
	}
	calcFn := preProcessed.Functions[0]
	g2 := arc.Graph{
		Functions: ir.Functions{
			ir.Function{
				Key:     "calculation",
				Outputs: calcFn.Outputs,
				Body:    calcFn.Body,
			},
		},
		Nodes: []graph.Node{
			{
				Key:  "calculation",
				Type: "calculation",
			},
			{
				Key:  "write",
				Type: "write",
				Config: map[string]any{
					"channel": cfg.Channel.Key(),
				},
			},
		},
	}
	cfg.Channel.Operations = lo.Filter(cfg.Channel.Operations, func(item channel.Operation, _ int) bool {
		return item.Type != "none"
	})
	if len(cfg.Channel.Operations) == 0 {
		g2.Edges = []graph.Edge{
			{
				Source: ir.Handle{Node: "calculation", Param: ir.DefaultOutputParam},
				Target: ir.Handle{Node: "write", Param: ir.DefaultInputParam},
			},
		}
	} else {
		for i, o := range cfg.Channel.Operations {
			key := fmt.Sprintf("op_%d", i)
			nextKey := fmt.Sprintf("op_%d", i)
			g2.Nodes = append(g2.Nodes, graph.Node{
				Key:  fmt.Sprintf("op_%d", i),
				Type: o.Type,
				Config: map[string]any{
					"duration": o.Duration,
				},
			})
			if o.ResetChannel != 0 {
				resetKey := fmt.Sprintf("on_reset_%d", o.ResetChannel)
				g2.Nodes = append(g2.Nodes, graph.Node{
					Key:  resetKey,
					Type: "on",
					Config: map[string]any{
						"channel": o.ResetChannel,
					},
				})
				g2.Edges = append(g2.Edges, graph.Edge{
					Source: ir.Handle{Node: resetKey, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: key, Param: "reset"},
				})
			}
			if i == 0 {
				g2.Edges = append(g2.Edges, graph.Edge{
					Source: ir.Handle{Node: "calculation", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: key, Param: ir.DefaultInputParam},
				})
			}
			if i == len(cfg.Channel.Operations)-1 {
				g2.Edges = append(g2.Edges, graph.Edge{
					Source: ir.Handle{Node: key, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "write", Param: ir.DefaultInputParam},
				})
			} else {
				g2.Edges = append(g2.Edges, graph.Edge{
					Source: ir.Handle{Node: key, Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: nextKey, Param: ir.DefaultInputParam},
				})
			}
		}
	}
	for k, v := range calcFn.Channels.Read {
		sym, err := cfg.Resolver.Resolve(ctx, v)
		if err != nil {
			return arc.Module{}, err
		}
		g2.Functions[0].Inputs = append(
			g2.Functions[0].Inputs, types.Param{Name: sym.Name, Type: *sym.Type.ValueType},
		)
		g2.Nodes = append(g2.Nodes, graph.Node{
			Key:    sym.Name,
			Type:   "on",
			Config: map[string]any{"channel": k},
		})
		g2.Edges = append(g2.Edges, graph.Edge{
			Source: ir.Handle{Node: sym.Name, Param: ir.DefaultOutputParam},
			Target: ir.Handle{Node: "calculation", Param: sym.Name},
		})
	}
	postProcessed, err := arc.CompileGraph(ctx, g2, arc.WithResolver(cfg.Resolver))
	if err != nil {
		return arc.Module{}, err
	}
	return postProcessed, nil
}
