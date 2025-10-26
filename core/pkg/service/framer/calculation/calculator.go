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

	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/stat"
	"github.com/synnaxlabs/arc/runtime/state"
	ntelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	arcruntime "github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Calculator is an extension of the lua-based computron.Calculator to provide
// specific functionality for evaluating calculations on channels using frame data.
type Calculator struct {
	ch         channel.Channel
	state      *state.State
	scheduler  *scheduler.Scheduler
	stateCfg   state.Config
	alignments map[channel.Key]telem.Alignment
	timeRange  telem.TimeRange
}

type CalculatorConfig struct {
	ChannelSvc channel.Readable
	Channel    channel.Channel
	Resolver   arc.SymbolResolver
}

var (
	_                       config.Config[CalculatorConfig] = CalculatorConfig{}
	DefaultCalculatorConfig                                 = CalculatorConfig{}
)

// Override implements config.Config.
func (c CalculatorConfig) Override(other CalculatorConfig) CalculatorConfig {
	c.ChannelSvc = override.Nil(c.ChannelSvc, other.ChannelSvc)
	c.Channel = other.Channel
	c.Resolver = override.Nil(c.Resolver, other.Resolver)
	return c
}

// Validate implements config.Config.
func (c CalculatorConfig) Validate() error {
	v := validate.New("arc.runtime")
	return v.Error()
}

func buildModule(ctx context.Context, cfg CalculatorConfig) (arc.Module, error) {
	g := arc.Graph{
		Functions: ir.Functions{
			{
				Key: "calculation",
				Outputs: types.Params{
					Keys:   []string{ir.DefaultOutputParam},
					Values: []types.Type{types.FromTelem(cfg.Channel.DataType)},
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
				ConfigValues: map[string]any{
					"channel": cfg.Channel.Key(),
				},
			},
		},
	}
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
				ConfigValues: map[string]any{
					"duration": o.Duration,
				},
			})
			if o.ResetChannel != 0 {
				resetKey := fmt.Sprintf("on_reset_%d", o.ResetChannel)
				g2.Nodes = append(g2.Nodes, graph.Node{
					Key:  resetKey,
					Type: "on",
					ConfigValues: map[string]any{
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
		g2.Functions[0].Inputs.Put(sym.Name, *sym.Type.ValueType)
		g2.Nodes = append(g2.Nodes, graph.Node{
			Key:          sym.Name,
			Type:         "on",
			ConfigValues: map[string]any{"channel": k},
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

// OpenCalculator opens a new calculator that evaluates the Expression of the provided
// channel. The requiredChannels provided must include ALL and ONLY the channels
// corresponding to the keys specified in ch.Requires.
//
// The calculator must be closed by calling Close() after use, or memory leaks will occur.
func OpenCalculator(
	ctx context.Context,
	cfgs ...CalculatorConfig,
) (*Calculator, error) {
	cfg, err := config.New(DefaultCalculatorConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	module, err := buildModule(ctx, cfg)
	if err != nil {
		return nil, err
	}
	stateCfg, err := arcruntime.NewStateConfig(ctx, cfg.ChannelSvc, module)
	if err != nil {
		return nil, err
	}
	progState := state.New(stateCfg)

	telemFactory := ntelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	statFactory := stat.NewFactory(stat.Config{})
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	wasmFactory, err := wasm.NewFactory(ctx, wasm.FactoryConfig{
		Module: module,
	})
	if err != nil {
		return nil, err
	}

	f := node.MultiFactory{
		opFactory,
		telemFactory,
		selectFactory,
		constantFactory,
		stableFactory,
		wasmFactory,
		statFactory,
	}

	f = append(f, wasmFactory)
	nodes := make(map[string]node.Node)
	for _, irNode := range module.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: module,
			State:  progState.Node(irNode.Key),
		})
		if err != nil {
			return nil, err
		}
		nodes[irNode.Key] = n
	}

	sched := scheduler.New(ctx, module.IR, nodes)
	sched.Init(ctx)
	alignments := make(map[channel.Key]telem.Alignment)
	for _, ch := range stateCfg.ChannelDigests {
		if ch.Index == 0 {
			alignments[channel.Key(ch.Key)] = telem.Alignment(0)
		} else {
			alignments[channel.Key(ch.Index)] = telem.Alignment(0)
		}
	}
	return &Calculator{
		scheduler:  sched,
		state:      progState,
		ch:         cfg.Channel,
		stateCfg:   stateCfg,
		alignments: alignments,
	}, nil
}

func (c *Calculator) ReadFrom() channel.Keys {
	ch := make([]channel.Key, 0, len(c.stateCfg.ChannelDigests)*2)
	for k := range c.stateCfg.ReactiveDeps {
		ch = append(ch, channel.Key(k))
	}
	return ch
}

// Channel returns information about the channel being calculated.
func (c *Calculator) Channel() channel.Channel { return c.ch }

// Next executes the next calculation step. It takes in the given frame and determines
// if enough data is available to perform the next set of calculations. The returned
// telem.Series will have a length equal to the number of new calculations completed.
// If no calculations are completed, the length of the series will be 0, and the caller
// is free to discard the returned value.
//
// Any error encountered during calculations is returned as well.
func (c *Calculator) Next(
	ctx context.Context,
	inputFrame,
	outputFrame framer.Frame,
) (framer.Frame, bool, error) {
	for rawI, rawKey := range inputFrame.RawKeys() {
		if inputFrame.ShouldExcludeRaw(rawI) {
			continue
		}
		v, ok := c.alignments[rawKey]
		s := inputFrame.RawSeriesAt(rawI)
		if ok && v == 0 {
			c.alignments[rawKey] = s.Alignment
		}
		if c.timeRange.Start == 0 || s.TimeRange.Start < c.timeRange.Start {
			c.timeRange.Start = s.TimeRange.Start
		}
		if c.timeRange.End == 0 || s.TimeRange.End > s.TimeRange.End {
			c.timeRange.End = s.TimeRange.End
		}
	}
	c.state.Ingest(inputFrame.ToStorage())
	var (
		ofr         = outputFrame.ToStorage()
		currChanged bool
		changed     bool
	)
	for {
		c.scheduler.Next(ctx)
		ofr, currChanged = c.state.FlushWrites(ofr)
		if !currChanged {
			break
		}
		changed = true
	}
	if !changed {
		return outputFrame, changed, nil
	}
	var alignment telem.Alignment
	for k, v := range c.alignments {
		alignment += v
		c.alignments[k] = 0
	}
	for rawI, s := range ofr.RawSeries() {
		if rawI < len(ofr.RawSeries())-2 {
			continue
		}
		s.Alignment = alignment
		s.TimeRange = c.timeRange
		ofr.SetRawSeriesAt(rawI, s)
	}
	return core.NewFrameFromStorage(ofr), true, nil
}

func (c *Calculator) Close() error {
	return nil
}
