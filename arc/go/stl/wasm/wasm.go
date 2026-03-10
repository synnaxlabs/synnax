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
	"math"
	"strings"

	runtimenode "github.com/synnaxlabs/arc/runtime/node"
	stlstrings "github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

type Module struct {
	Module        api.Module
	Memory        api.Memory
	Strings       *stlstrings.State
	NodeKeySetter NodeKeySetter
}

func (w *Module) Create(_ context.Context, cfg runtimenode.Config) (runtimenode.Node, error) {
	irFn, ok := cfg.Program.Functions.Find(cfg.Node.Type)
	if !ok {
		return nil, query.ErrNotFound
	}
	fn := w.Module.ExportedFunction(cfg.Node.Type)
	if fn == nil {
		return nil, query.ErrNotFound
	}
	// Entry nodes have no incoming edges and are not expression nodes.
	// They should only execute once per stage entry.
	isEntryNode := !strings.HasPrefix(cfg.Node.Key, "expression_") &&
		len(cfg.Program.Edges.GetInputs(cfg.Node.Key)) == 0

	configCount := len(cfg.Node.Config)
	params := make([]uint64, configCount+len(irFn.Inputs))
	for i, param := range cfg.Node.Config {
		if s, ok := param.Value.(string); ok {
			// String config params get a stable handle that persists across Flush calls.
			params[i] = uint64(w.Strings.CreateConfig(s))
			continue
		}
		val, err := ConvertConfigValue(param.Value)
		if err != nil {
			return nil, err
		}
		params[i] = val
	}

	base := cfg.Program.OutputMemoryBases[cfg.Node.Type]
	memOffsets := make([]uint32, len(irFn.Outputs))
	offset := base + 8
	for i, t := range irFn.Outputs {
		memOffsets[i] = offset
		offset += uint32(t.Type.Density())
	}

	n := &nodeImpl{
		Node:          cfg.State,
		ir:            cfg.Node,
		fn:            fn,
		mem:           w.Memory,
		fnOutputs:     irFn.Outputs,
		memOffsets:    memOffsets,
		outputValues:  make([]result, len(irFn.Outputs)),
		memBase:       base,
		params:        params,
		configCount:   configCount,
		offsets:       make([]int, len(irFn.Outputs)),
		isEntryNode:   isEntryNode,
		nodeKeySetter: w.NodeKeySetter,
	}
	return n, nil
}

// ConvertConfigValue converts a config value to uint64 for WASM function calls.
func ConvertConfigValue(v any) (uint64, error) {
	switch val := v.(type) {
	case int8:
		return uint64(val), nil
	case int16:
		return uint64(val), nil
	case int32:
		return uint64(val), nil
	case int64:
		return uint64(val), nil
	case uint8:
		return uint64(val), nil
	case uint16:
		return uint64(val), nil
	case uint32:
		return uint64(val), nil
	case uint64:
		return val, nil
	case float32:
		return uint64(math.Float32bits(val)), nil
	case float64:
		return math.Float64bits(val), nil
	case telem.TimeStamp:
		return uint64(val), nil
	default:
		err := errors.Newf("unsupported config value type: %T", v)
		zap.S().DPanic(err.Error())
		return 0, err
	}
}
