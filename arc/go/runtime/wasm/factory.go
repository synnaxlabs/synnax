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
	"fmt"
	"math"
	"strings"

	node2 "github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/runtime/state"
	runtimebindings "github.com/synnaxlabs/arc/runtime/wasm/bindings"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero/api"
)

type factory struct {
	wasm    api.Module
	runtime *runtimebindings.Runtime
}

func (w *factory) Create(_ context.Context, cfg node2.Config) (node2.Node, error) {
	irFn, ok := cfg.Module.Functions.Find(cfg.Node.Type)
	if !ok {
		return nil, query.ErrNotFound
	}
	wasmFn := w.wasm.ExportedFunction(cfg.Node.Type)
	// Entry nodes have no incoming edges and are not expression nodes.
	// They should only execute once per stage entry.
	isEntryNode := !strings.HasPrefix(cfg.Node.Key, "expression_") &&
		len(cfg.Module.Edges.GetInputs(cfg.Node.Key)) == 0

	configCount := len(cfg.Node.Config)
	params := make([]uint64, configCount+len(irFn.Inputs))
	for i, param := range cfg.Node.Config {
		params[i] = convertConfigValue(param.Value)
	}

	n := &nodeImpl{
		Node: cfg.State,
		ir:   cfg.Node,
		wasm: WrapFunction(
			wasmFn,
			w.wasm.Memory(),
			irFn.Outputs,
			cfg.Module.OutputMemoryBases[cfg.Node.Type],
		),
		runtime:     w.runtime,
		params:      params,
		configCount: configCount,
		offsets:     make([]int, len(irFn.Outputs)),
		isEntryNode: isEntryNode,
	}
	return n, nil
}

// convertConfigValue converts a config value to uint64 for WASM function calls.
func convertConfigValue(v any) uint64 {
	switch val := v.(type) {
	case int8:
		return uint64(val)
	case int16:
		return uint64(val)
	case int32:
		return uint64(val)
	case int64:
		return uint64(val)
	case uint8:
		return uint64(val)
	case uint16:
		return uint64(val)
	case uint32:
		return uint64(val)
	case uint64:
		return val
	case float32:
		return uint64(math.Float32bits(val))
	case float64:
		return math.Float64bits(val)
	case telem.TimeStamp:
		return uint64(val)
	default:
		panic(fmt.Sprintf("unsupported config value type: %T", v))
	}
}

type FactoryConfig struct {
	Module *Module
	State  *state.State
}

func NewFactory(mod *Module) (node2.Factory, error) {
	return &factory{wasm: mod.wasmModule, runtime: mod.runtime}, nil
}
