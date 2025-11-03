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

	"github.com/synnaxlabs/arc"
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
	cfg        CalculatorConfig
	ch         channel.Channel
	state      *state.State
	scheduler  *scheduler.Scheduler
	stateCfg   arcruntime.ExtendedStateConfig
	alignments map[channel.Key]telem.Alignment
	timeRange  telem.TimeRange
}

type CalculatorConfig struct {
	ChannelSvc          channel.Readable
	Channel             channel.Channel
	Resolver            arc.SymbolResolver
	CalculateAlignments *bool
}

var (
	_                       config.Config[CalculatorConfig] = CalculatorConfig{}
	DefaultCalculatorConfig                                 = CalculatorConfig{
		CalculateAlignments: config.True(),
	}
)

// Override implements config.Config.
func (c CalculatorConfig) Override(other CalculatorConfig) CalculatorConfig {
	c.ChannelSvc = override.Nil(c.ChannelSvc, other.ChannelSvc)
	c.Channel = other.Channel
	c.Resolver = override.Nil(c.Resolver, other.Resolver)
	c.CalculateAlignments = override.Nil(c.CalculateAlignments, other.CalculateAlignments)
	return c
}

// Validate implements config.Config.
func (c CalculatorConfig) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "calculate_alignments", c.CalculateAlignments)
	return v.Error()
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
	module, err := compile(ctx, cfg)
	if err != nil {
		return nil, err
	}
	stateCfg, err := arcruntime.NewStateConfig(ctx, cfg.ChannelSvc, module)
	if err != nil {
		return nil, err
	}
	progState := state.New(stateCfg.State)

	telemFactory := ntelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	statFactory := stat.NewFactory(stat.Config{})
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
		Module: module,
	})
	if err != nil {
		return nil, err
	}
	wasmFactory, err := wasm.NewFactory(wasmMod)
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
	for _, ch := range stateCfg.State.ChannelDigests {
		if ch.Index == 0 {
			alignments[channel.Key(ch.Key)] = telem.Alignment(0)
		} else {
			alignments[channel.Key(ch.Index)] = telem.Alignment(0)
		}
	}
	return &Calculator{
		cfg:        cfg,
		scheduler:  sched,
		state:      progState,
		ch:         cfg.Channel,
		stateCfg:   stateCfg,
		alignments: alignments,
	}, nil
}

func (c *Calculator) ReadFrom() channel.Keys {
	return c.stateCfg.Reads.Keys()
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
	if *c.cfg.CalculateAlignments {
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
			if c.timeRange.End == 0 || s.TimeRange.End > c.timeRange.End {
				c.timeRange.End = s.TimeRange.End
			}
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
	c.state.ClearReads()
	if *c.cfg.CalculateAlignments {
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
	}
	return core.NewFrameFromStorage(ofr), true, nil
}

func (c *Calculator) Close() error {
	return nil
}
