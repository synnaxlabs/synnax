// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"context"

	"github.com/synnaxlabs/arc/program"
	runtimenode "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl/channel"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/stateful"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/x/errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Runtime wraps a wazero runtime with instantiated STL host modules
// and a WASM guest module.
type Runtime struct {
	wasmRuntime wazero.Runtime
	wasmModule  api.Module
	strings     *stlstrings.State
	stringsMod  *stlstrings.Module
	series      *series.State
	channels    *channel.State
	stateful    *stateful.Module
	timeMod     *time.Module
}

// RuntimeConfig configures a WASM runtime.
type RuntimeConfig struct {
	Program        program.Program
	State          *state.State
	ChannelDigests []channel.Digest
}

// Open creates a new WASM runtime, instantiates all STL host modules,
// and loads the compiled WASM guest module.
func Open(ctx context.Context, cfg RuntimeConfig) (*Runtime, error) {
	wasmRuntime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	rt := &Runtime{
		wasmRuntime: wasmRuntime,
		strings:     stlstrings.NewState(),
		series:      series.NewState(),
		channels:    channel.NewState(cfg.ChannelDigests),
	}
	var err error
	rt.stateful, err = stateful.NewModule(ctx, rt.series, rt.strings, wasmRuntime)
	if err != nil {
		return nil, errors.Wrap(err, "instantiate stateful module")
	}
	if _, err = series.NewModule(ctx, rt.series, wasmRuntime); err != nil {
		return nil, errors.Wrap(err, "instantiate series module")
	}
	rt.stringsMod, err = stlstrings.NewModule(ctx, rt.strings, wasmRuntime, nil)
	if err != nil {
		return nil, errors.Wrap(err, "instantiate strings module")
	}
	if _, err = stlmath.NewModule(ctx, wasmRuntime); err != nil {
		return nil, errors.Wrap(err, "instantiate math module")
	}
	if _, err = stlerrors.NewModule(ctx, nil, wasmRuntime); err != nil {
		return nil, errors.Wrap(err, "instantiate errors module")
	}
	rt.timeMod, err = time.NewModule(ctx, wasmRuntime)
	if err != nil {
		return nil, errors.Wrap(err, "instantiate time module")
	}
	if _, err = channel.NewModule(ctx, rt.channels, rt.strings, wasmRuntime); err != nil {
		return nil, errors.Wrap(err, "instantiate channel module")
	}
	rt.wasmModule, err = wasmRuntime.Instantiate(ctx, cfg.Program.WASM)
	if err != nil {
		return nil, errors.Wrap(err, "instantiate WASM guest module")
	}
	rt.stringsMod.SetMemory(rt.wasmModule.Memory())
	return rt, nil
}

// ChannelState returns the channel state for testing.
func (rt *Runtime) ChannelState() *channel.State {
	return rt.channels
}

// Close releases all resources held by the runtime.
func (rt *Runtime) Close() error {
	ctx := context.TODO()
	return errors.Join(rt.wasmModule.Close(ctx), rt.wasmRuntime.Close(ctx))
}

// NewFactory creates a node factory from a Runtime.
func NewFactory(rt *Runtime) runtimenode.Factory {
	return &Module{
		Module:        rt.wasmModule,
		Memory:        rt.wasmModule.Memory(),
		Strings:       rt.strings,
		NodeKeySetter: rt.stateful,
	}
}
