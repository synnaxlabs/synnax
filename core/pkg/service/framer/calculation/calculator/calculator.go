// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator

import (
	"context"
	"fmt"

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
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Calculator is an engine for executing expressions and operations in calculated
// channels.
type Calculator struct {
	cfg        Config
	state      *state.State
	scheduler  *scheduler.Scheduler
	stateCfg   arcruntime.ExtendedStateConfig
	alignments map[channel.Key]telem.Alignment
	timeRange  telem.TimeRange
}

type Config struct {
	Module              compiler.Module
	CalculateAlignments *bool
}

var (
	_                       config.Config[Config] = Config{}
	DefaultCalculatorConfig                       = Config{
		CalculateAlignments: config.True(),
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.CalculateAlignments = override.Nil(c.CalculateAlignments, other.CalculateAlignments)
	c.Module = override.Zero(c.Module, other.Module)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "calculate_alignments", c.CalculateAlignments)
	validate.NonZeroable(v, "module", c.Module)
	return v.Error()
}

// Open opens a new calculator that evaluates the Expression of the provided
// channel. The requiredChannels provided must include ALL and ONLY the channels
// corresponding to the keys specified in ch.Requires.
//
// The calculator must be closed by calling Close() after use, or memory leaks will occur.
func Open(
	ctx context.Context,
	cfgs ...Config,
) (*Calculator, error) {
	cfg, err := config.New(DefaultCalculatorConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	progState := state.New(cfg.Module.StateConfig.State)
	telemFactory := ntelem.NewTelemFactory()
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	wasmMod, err := wasm.OpenModule(ctx, wasm.ModuleConfig{
		Module: cfg.Module.Module,
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
		stat.Factory,
	}
	nodes := make(map[string]node.Node)
	for _, irNode := range cfg.Module.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: cfg.Module.Module,
			State:  progState.Node(irNode.Key),
		})
		if err != nil {
			return nil, err
		}
		nodes[irNode.Key] = n
	}

	sched := scheduler.New(ctx, cfg.Module.IR, nodes)
	sched.Init(ctx)
	alignments := make(map[channel.Key]telem.Alignment)
	for _, ch := range cfg.Module.StateConfig.State.ChannelDigests {
		if ch.Key == uint32(cfg.Module.Channel.Key()) {
			continue
		}
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
		stateCfg:   cfg.Module.StateConfig,
		alignments: alignments,
	}, nil
}

func (c *Calculator) WriteTo() channel.Keys {
	return c.stateCfg.Writes.Keys()
}

func (c *Calculator) ReadFrom() channel.Keys {
	return c.stateCfg.Reads.Keys()
}

func (c *Calculator) Channel() channel.Channel { return c.cfg.Module.Channel }

// String returns a human-readable representation of the calculator including channel
// properties like name, key, index, data type, and dependency count.
func (c *Calculator) String() string {
	ch := c.cfg.Module.Channel
	indexStr := ""
	if idx := ch.Index(); idx != 0 {
		indexStr = fmt.Sprintf(" index=%d", idx)
	}
	return fmt.Sprintf("%s (key=%d%s type=%s deps=%d)",
		ch.Name,
		ch.Key(),
		indexStr,
		ch.DataType,
		len(c.ReadFrom()),
	)
}

// Next executes the next calculation step. It takes in the given frame and determines
// if enough data is available to perform the next set of calculations. The returned
// telem.Series will have a length equal to the number of new calculations completed.
// If no calculations are completed, the length of the series will be 0, and the caller
// is free to discard the returned value.
//
// Any error encountered during calculations is returned as well.
func (c *Calculator) Next(
	ctx context.Context,
	input,
	output framer.Frame,
) (framer.Frame, bool, error) {
	if *c.cfg.CalculateAlignments {
		for rawI, rawKey := range input.RawKeys() {
			if input.ShouldExcludeRaw(rawI) {
				continue
			}
			v, ok := c.alignments[rawKey]
			s := input.RawSeriesAt(rawI)
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
	c.state.Ingest(input.ToStorage())
	var (
		ofr         = output.ToStorage()
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
		return output, false, nil
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
