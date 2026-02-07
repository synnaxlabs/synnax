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

	"github.com/synnaxlabs/arc/module"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/stl/channel"
	stlerrors "github.com/synnaxlabs/arc/stl/errors"
	"github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/time"
	"github.com/synnaxlabs/arc/stl/vars"
	"github.com/synnaxlabs/x/errors"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	wasmRuntime wazero.Runtime
	wasmModule  api.Module
}

func (m *Module) Close() error {
	ctx := context.TODO()
	return errors.Join(m.wasmModule.Close(ctx), m.wasmRuntime.Close(ctx))
}

type ModuleConfig struct {
	State      *state.State
	Module     module.Module
	STLModules []stl.Module
}

// DefaultModules constructs the default set of STL modules from the given state.
func DefaultModules(s *state.State) (
	modules []stl.Module,
	strMod *strings.Module,
	errMod *stlerrors.Module,
) {
	strMod = strings.NewModule(s.Strings)
	errMod = stlerrors.NewModule()
	modules = []stl.Module{
		channel.NewModule(s.Channel, s.Strings),
		vars.NewModule(s.Series, s.Strings),
		series.NewModule(s.Series),
		strMod,
		math.NewModule(),
		time.NewModule(),
		errMod,
	}
	return modules, strMod, errMod
}

func OpenModule(ctx context.Context, cfg ModuleConfig) (*Module, error) {
	wasmRuntime := wazero.NewRuntimeWithConfig(
		ctx, wazero.NewRuntimeConfigCompiler(),
	)
	modules := cfg.STLModules
	if len(modules) == 0 {
		modules, _, _ = DefaultModules(cfg.State)
	}
	hostRT := NewWazeroHostRuntime(ctx, wasmRuntime)
	for _, m := range modules {
		if err := m.BindTo(ctx, hostRT); err != nil {
			return nil, err
		}
	}
	if err := hostRT.Instantiate(); err != nil {
		return nil, err
	}
	wasmModule, err := wasmRuntime.Instantiate(ctx, cfg.Module.WASM)
	if err != nil {
		return nil, err
	}
	for _, m := range modules {
		if ms, ok := m.(stl.MemorySetter); ok {
			ms.SetMemory(wasmModule.Memory())
		}
	}
	return &Module{
		wasmModule:  wasmModule,
		wasmRuntime: wasmRuntime,
	}, nil
}
