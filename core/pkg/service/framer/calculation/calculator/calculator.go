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

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/scheduler"
	stlchannel "github.com/synnaxlabs/arc/stl/channel"
	"github.com/synnaxlabs/arc/stl/constant"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	stlop "github.com/synnaxlabs/arc/stl/op"
	"github.com/synnaxlabs/arc/stl/selector"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stable"
	"github.com/synnaxlabs/arc/stl/stat"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/wasm"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	arcruntime "github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

type calcState struct {
	nodes   *node.ProgramState
	channel *stlchannel.ProgramState
	series  *series.ProgramState
	strings *stlstrings.ProgramState
}

// Calculator is an engine for executing expressions and operations in calculated
// channels.
type Calculator struct {
	state     calcState
	scheduler *scheduler.Scheduler
	stateCfg  arcruntime.ExtendedStateConfig
	cfg       Config
	start     telem.TimeStamp
	closer    xio.MultiCloser
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
) (_ *Calculator, err error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}

	var cs calcState
	cs.channel = stlchannel.NewProgramState(cfg.Module.StateConfig.ChannelDigests)
	cs.series = series.NewProgramState()
	cs.strings = stlstrings.NewProgramState()

	channelMod, err := stlchannel.NewModule(ctx, cs.channel, cs.strings, nil)
	if err != nil {
		return nil, err
	}

	f := node.CompoundFactory{
		channelMod,
		selector.NewModule(),
		constant.NewModule(),
		stlop.NewModule(),
		stable.NewModule(),
		&stat.Module{},
	}

	var closers xio.MultiCloser
	defer func() {
		if err != nil {
			err = errors.Join(err, closers.Close())
		}
	}()

	if len(cfg.Module.WASM) > 0 {
		var statefulMod *stateful.Module
		var stringsMod *stlstrings.Module
		var errorsMod *stlerrors.Module
		wasmRT := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
		closers = append(closers, xio.CloserFunc(func() error {
			return wasmRT.Close(ctx)
		}))
		if statefulMod, err = stateful.NewModule(ctx, cs.series, cs.strings, wasmRT); err != nil {
			return nil, err
		}
		if _, err = series.NewModule(ctx, cs.series, wasmRT); err != nil {
			return nil, err
		}
		if stringsMod, err = stlstrings.NewModule(ctx, cs.strings, wasmRT, nil); err != nil {
			return nil, err
		}
		if _, err = stlmath.NewModule(ctx, wasmRT); err != nil {
			return nil, err
		}
		if errorsMod, err = stlerrors.NewModule(ctx, nil, wasmRT); err != nil {
			return nil, err
		}

		guest, guestErr := wasmRT.Instantiate(ctx, cfg.Module.WASM)
		if guestErr != nil {
			return nil, guestErr
		}
		stringsMod.SetMemory(guest.Memory())
		errorsMod.SetMemory(guest.Memory())
		closers = append(closers,
			xio.CloserFunc(func() error { return guest.Close(ctx) }),
		)
		f = append(f, &wasm.Module{
			Module:        guest,
			Memory:        guest.Memory(),
			Strings:       cs.strings,
			NodeKeySetter: statefulMod,
		})
	}

	cs.nodes = node.New(cfg.Module.StateConfig.IR)
	nodes := make(map[string]node.Node)
	for _, irNode := range cfg.Module.Nodes {
		n, nodeErr := f.Create(ctx, node.Config{
			Node:    irNode,
			Program: cfg.Module.Program,
			State:   cs.nodes.Node(irNode.Key),
		})
		if nodeErr != nil {
			return nil, nodeErr
		}
		nodes[irNode.Key] = n
	}

	sched := scheduler.New(cfg.Module.IR, nodes, 0)
	c := &Calculator{
		cfg:       cfg,
		scheduler: sched,
		state:     cs,
		stateCfg:  cfg.Module.StateConfig,
		closer:    closers,
		start:     telem.Now(),
	}
	closers = nil
	return c, nil
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
	c.state.channel.Ingest(input.ToStorage())
	var (
		ofr         = output.ToStorage()
		currChanged bool
		changed     bool
	)
	for {
		c.scheduler.Next(ctx, telem.Since(c.start), node.ReasonChannelInput)
		ofr, currChanged = c.state.channel.Flush(ofr)
		// Series and strings must be cleared after every flush, not just at the end
		// of the loop. On each iteration the scheduler may create new series/string
		// handles via WASM; if we don't clear them before the next iteration, stale
		// handles accumulate and the counter keeps growing.
		c.state.series.Clear()
		c.state.strings.Clear()
		if !currChanged {
			break
		}
		changed = true
	}
	// ClearReads must run unconditionally, not just when changed is true. Otherwise,
	// when a required input channel never sends data, consumed series accumulate
	// indefinitely in channel.reads.
	c.state.channel.ClearReads()
	if !changed {
		return output, false, nil
	}
	return frame.NewFromStorage(ofr), true, nil
}

func (c *Calculator) Close() error {
	return c.closer.Close()
}
