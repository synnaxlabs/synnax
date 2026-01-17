// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
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
	cfg       Config
	state     *state.State
	scheduler *scheduler.Scheduler
	stateCfg  arcruntime.ExtendedStateConfig
	start     telem.TimeStamp
}

type Config struct {
	Module compiler.Module
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Module = override.Zero(c.Module, other.Module)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("arc.runtime")
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
	cfg, err := config.New(DefaultConfig, cfgs...)
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

	sched := scheduler.New(cfg.Module.IR, nodes)
	return &Calculator{
		cfg:       cfg,
		scheduler: sched,
		state:     progState,
		stateCfg:  cfg.Module.StateConfig,
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
	c.state.Ingest(input.ToStorage())
	var (
		ofr         = output.ToStorage()
		currChanged bool
		changed     bool
	)
	for {
		c.scheduler.Next(ctx, telem.Since(c.start))
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
	return frame.NewFromStorage(ofr), true, nil
}

func (c *Calculator) Close() error {
	return nil
}
