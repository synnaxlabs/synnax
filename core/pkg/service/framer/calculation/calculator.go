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
	"github.com/synnaxlabs/arc/runtime"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
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
	"github.com/synnaxlabs/x/validate"
)

// Calculator is an extension of the lua-based computron.Calculator to provide
// specific functionality for evaluating calculations on channels using frame data.
type Calculator struct {
	ch        channel.Channel
	state     *state.State
	scheduler *runtime.Scheduler
	stateCfg  state.Config
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

func buildModule(ctx context.Context, cfg CalculatorConfig) arc.Module {
	graph := arc.Graph{
		Functions: ir.Functions{
			{
				Key: "calculation",
				Outputs: types.Params{
					Keys:   []string{ir.DefaultOutputParam},
					Values: []types.Type{types.FromTelem(cfg.Channel.DataType)},
				},
				Body: ir.Body{Raw: cfg.Channel.Expression},
			},
		},
		Nodes: []graph.Node{},
	}
	g, err := arc.CompileGraph(ctx, graph, arc.WithResolver(cfg.Resolver))
	if err != nil {
		panic(err)
	}
	fmt.Println(g)
	return arc.Module{}
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
	module := buildModule(ctx, cfg)
	stateCfg, err := arcruntime.NewStateConfig(ctx, cfg.ChannelSvc, module)
	if err != nil {
		return nil, err
	}
	progState := state.New(stateCfg)

	telemFactory := ntelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
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

	scheduler := runtime.NewScheduler(module.IR, nodes)
	scheduler.Init(ctx)
	return &Calculator{
		scheduler: scheduler,
		state:     progState,
		ch:        cfg.Channel,
		stateCfg:  stateCfg,
	}, nil
}

func (c *Calculator) ReadFrom() channel.Keys {
	ch := make([]channel.Key, 0, len(c.stateCfg.ChannelDigests)*2)
	for _, v := range c.stateCfg.ChannelDigests {
		ch = append(ch, channel.Key(v.Key), channel.Key(v.Index))
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
func (c *Calculator) Next(ctx context.Context, inputFrame, outputFrame framer.Frame) (framer.Frame, bool, error) {
	c.state.Ingest(inputFrame.ToStorage(), c.scheduler.MarkNodesChange)
	c.scheduler.Next(ctx)
	ofr, changed := c.state.FlushWrites(outputFrame.ToStorage())
	if !changed {
		return outputFrame, false, nil
	}
	return core.NewFrameFromStorage(ofr), true, nil
}
