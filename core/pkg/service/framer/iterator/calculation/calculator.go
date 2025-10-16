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
	"github.com/synnaxlabs/arc/runtime"
	"github.com/synnaxlabs/arc/runtime/constant"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/op"
	"github.com/synnaxlabs/arc/runtime/selector"
	"github.com/synnaxlabs/arc/runtime/stable"
	"github.com/synnaxlabs/arc/runtime/state"
	ntelem "github.com/synnaxlabs/arc/runtime/telem"
	"github.com/synnaxlabs/arc/runtime/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Calculator is an extension of the lua-based computron.Calculator to provide
// specific functionality for evaluating calculations on channels using frame data.
type Calculator struct {
	ch        channel.Channel
	telem     *ntelem.State
	scheduler *runtime.Scheduler
}

// ConfigValues is the configuration for an arc runtime.
type CalculatorConfig struct {
	ChannelSvc channel.Readable
	Channel    channel.Channel
	// Module is the compiled arc module that needs to be executed.
	//
	// [REQUIRED]
	Module arc.Module
}

var (
	_ config.Config[CalculatorConfig] = CalculatorConfig{}
	// DefaultConfig is the default configuration for opening a runtime. This
	// configuration is not valid on its own. Fields must be set according to the
	// ConfigValues documentation.
	DefaultCalculatorConfig = CalculatorConfig{}
)

func (c *Calculator) ReadFrom() []channel.Key {
	ch := make(set.Set[channel.Key])
	for k, _ := range c.telem.Readers {
		ch.Add(channel.Key(k))
	}
	return ch.Keys()
}

// Override implements config.Config.
func (c CalculatorConfig) Override(other CalculatorConfig) CalculatorConfig {
	c.Module = override.Zero(c.Module, other.Module)
	c.ChannelSvc = override.Nil(c.ChannelSvc, other.ChannelSvc)
	c.Channel = other.Channel
	return c
}

// Validate implements config.Config.
func (c CalculatorConfig) Validate() error {
	v := validate.New("arc.runtime")
	validate.NotNil(v, "module", c.Module)
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
	progState, err := state.NewState(ctx, cfg.Module.IR)
	if err != nil {
		return nil, err
	}

	telemState := ntelem.NewState()
	telemFactory := ntelem.NewTelemFactory(telemState)
	selectFactory := selector.NewFactory()
	constantFactory := constant.NewFactory()
	opFactory := op.NewFactory()
	stableFactory := stable.NewFactory(stable.FactoryConfig{})
	wasmFactory, err := wasm.NewFactory(ctx, wasm.FactoryConfig{
		Module: cfg.Module,
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
	if len(cfg.Module.WASM) > 0 {
		wasmFactory, err := wasm.NewFactory(ctx, wasm.FactoryConfig{
			Module: cfg.Module,
		})
		if err != nil {
			return nil, err
		}
		f = append(f, wasmFactory)
	}
	nodes := make(map[string]node.Node)
	for _, irNode := range cfg.Module.Nodes {
		n, err := f.Create(ctx, node.Config{
			Node:   irNode,
			Module: cfg.Module,
			State:  progState,
		})
		if err != nil {
			return nil, err
		}
		nodes[irNode.Key] = n
	}

	scheduler := runtime.NewScheduler(cfg.Module.IR, nodes)
	c := &Calculator{scheduler: scheduler, telem: telemState, ch: cfg.Channel}
	for _, k := range c.ReadFrom() {
		var ch channel.Channel
		if err = cfg.ChannelSvc.NewRetrieve().WhereKeys(k).Entry(&ch).Exec(ctx, nil); err != nil {
			return nil, err
		}
		telemState.Data[uint32(ch.Key())] = ntelem.Data{
			IndexKey: uint32(ch.Index()),
		}

	}
	return c, nil
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
func (c *Calculator) Next(ctx context.Context, fr framer.Frame) (telem.Series, error) {
	c.telem.Ingest(fr.ToStorage(), c.scheduler.MarkNodesChange)
	c.scheduler.Next(ctx)
	return c.telem.Writes[uint32(c.ch.Key())], nil
}
